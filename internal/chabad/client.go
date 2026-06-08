// Package chabad fetches Chabad-authoritative zmanim and candle lighting times.
//
// # RSS parsing
//
// The zmanim RSS feed is not a stable structured API. Title labels vary by
// day type, holiday context, and undocumented server-side logic. Known
// variability includes:
//
//   - Misheyakir label: weekday = "Earliest Tallit and Tefillin (Misheyakir)",
//     Shabbos/YT = "Earliest Tallit (Misheyakir)". Both map to the same field.
//   - Missing delimiter: "Sunset (Shkiah)Fast Begins" (no space before Fast)
//   - Pipe-delimited composites: "Candle Lighting | Fast Begins"
//   - Combined ends: "Holiday/Fast Ends", "Shabbat/Holiday Ends"
//   - Chametz deadlines: "Finish Eating Chametz before" (Erev Pesach only)
//
// The parser uses a Normalize → Split → Classify pipeline:
//  1. normalizeLabel: fix missing delimiters
//  2. splitLabel: split on " | " into individual semantic tokens
//  3. classifyLabel: map each token to a canonical field and/or event flag
//
// Tokens mapping to a canonical field update ZmanimDay. Tokens flagged as
// events (or with no field mapping) are stored in ZmanimDay.Events, keyed by
// the normalised label. This preserves all data without requiring an
// exhaustive dataset — a full 19-year Hebrew calendar cycle would be needed
// to enumerate every possible label variant.
//
// # Geolocation
//
// For lat/lon locations, chabad.org uses (undocumented):
//   - locationType=3
//   - coords=LAT,LON      comma-separated decimal degrees
//   - tzname=IANA*TZ      IANA timezone with "/" replaced by "*"
//   - n=NAME              display name — required for lat/lon requests
//
// # tdate encoding
//
// The candle ICS endpoint requires literal slashes in M/D/YYYY. Go's
// url.Values.Encode() percent-encodes them to %2F causing HTTP 403.
// tdate is therefore appended raw after encoding all other params.
//
// # User-Agent
//
// chabad.org blocks Go's default "Go-http-client/1.1" with 403.
// A generic browser User-Agent is sent instead.
package chabad

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"

	"github.com/toobuntu/zman-didan/internal/cache"
	"github.com/toobuntu/zman-didan/internal/types"
)

const (
	zmanimRSSURL   = "https://www.chabad.org/tools/rss/zmanim.xml"
	candleICSURL   = "https://www.chabad.org/calendar/candlelighting/candlelighting.ics.asp"
	candleCacheTTL = 24 * time.Hour

	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// ---- RSS classifier ----

type classifierRule struct {
	match func(string) bool
	field string // "" means no canonical field
	event bool   // store in ZmanimDay.Events
}

// substr returns a matcher that reports whether s is a substring of the label.
func substr(s string) func(string) bool {
	return func(label string) bool { return strings.Contains(label, s) }
}

// anySubstr returns a matcher that reports whether any of ss is a substring.
func anySubstr(ss ...string) func(string) bool {
	return func(label string) bool {
		for _, s := range ss {
			if strings.Contains(label, s) {
				return true
			}
		}
		return false
	}
}

// classifierRules are evaluated in order; the first match wins per token.
// "Misheyakir" matches both weekday and Shabbos/YT label variants since both
// contain "Misheyakir" as a substring (the Shabbos/YT label omits "and
// Tefillin"). This substring approach replaced an exact-match map in 789265a;
// reverting to exact matching silently drops every Shabbos/Yom-Tov zman.
var classifierRules = []classifierRule{
	{match: substr("Alot Hashachar"), field: "alos"},
	{match: substr("Misheyakir"), field: "misheyakir"},
	{match: anySubstr("Hanetz Hachamah", "Sunrise"), field: "sunrise"},
	{match: anySubstr("Latest Shema", "Latest Kriat Shema"), field: "latest_shema"},
	{match: anySubstr("Chatzot Hayom", "Midday"), field: "chatzos"},
	{match: substr("Plag"), field: "plag_hamincha"},
	{match: anySubstr("Shkiah", "Sunset"), field: "shkiah"},
	// Pure nightfall — field only.
	{match: anySubstr("Tzeit Hakochavim", "Nightfall"), field: "tzeis"},
	// Contextual "ends" — set tzeis and store as event so callers can
	// distinguish "Shabbat Ends" from plain nightfall.
	{match: anySubstr(
		"Shabbat Ends", "Shabbos Ends",
		"Holiday Ends", "Yom Tov Ends",
		"Holiday/Fast Ends", "Shabbat/Holiday Ends",
		"Fast Ends",
	), field: "tzeis", event: true},
	{match: anySubstr("Chatzot HaLailah", "Midnight"), field: "chatzos_halaila"},
	// Event-only entries: no canonical field, stored in ZmanimDay.Events.
	// "Candle Lighting" in the RSS is context-specific (Erev YK candles are
	// before shkiah, not tzeis) — do not map to a field.
	{match: substr("Candle Lighting"), event: true},
	{match: substr("Fast Begins"), event: true},
	{match: substr("Chametz"), event: true},
}

// classifyLabel returns the canonical field name and event flag for label.
// Returns ("", true) for unrecognised labels, preserving them as events.
func classifyLabel(label string) (field string, isEvent bool) {
	for _, r := range classifierRules {
		if r.match(label) {
			return r.field, r.event
		}
	}
	return "", true // unknown label → event only
}

// normalizeLabel fixes known RSS formatting defects before splitting.
// The feed occasionally emits ")Word" without a delimiter, e.g.:
// "Sunset (Shkiah)Fast Begins" → "Sunset (Shkiah) | Fast Begins"
func normalizeLabel(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		b.WriteRune(r)
		if r == ')' && i+1 < len(runes) && runes[i+1] >= 'A' && runes[i+1] <= 'Z' {
			b.WriteString(" | ")
		}
	}
	return b.String()
}

// splitLabel normalises and splits a composite label on " | ".
func splitLabel(s string) []string {
	parts := strings.Split(normalizeLabel(s), " | ")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// CandleDay holds the candle lighting and havdalah times for one calendar date.
// The map key is "YYYY-MM-DD" in the location's local timezone.
//
// IsYomTov distinguishes "Light Holiday Candles" (YT) from "Light Shabbat
// Candles" (Shabbos).
//
// AfterHavdala is true for "Light Holiday Candles after" — second-night Yom
// Tov candles that must be lit after Havdalah from the first day.
type CandleDay struct {
	Candles      time.Time
	Havdalah     time.Time
	IsYomTov     bool
	AfterHavdala bool
}

// Client fetches from chabad.org.
type Client struct {
	httpClient  *http.Client
	zmanimCache *cache.ZmanimCache
	candleCache *cache.HTTPCache
}

// NewClient returns a ready Client.
func NewClient(c *cache.ZmanimCache) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		zmanimCache: c,
		candleCache: cache.NewHTTP(candleCacheTTL),
	}
}

// FetchZmanimInZone returns zmanim for date. fromCache is true when the result
// was served from the local disk cache rather than downloaded from chabad.org.
func (c *Client) FetchZmanimInZone(date time.Time, loc types.Location, cfg types.Config) (z types.ZmanimDay, fromCache bool, err error) {
	locationID := cfg.LocationID()
	if !cfg.Refresh {
		if cached, ok := c.zmanimCache.Get(date, locationID); ok {
			return cached, true, nil
		}
	}
	z, err = c.fetchRemoteZmanim(date, loc, cfg.Candles)
	if err != nil {
		return types.ZmanimDay{}, false, err
	}
	if setErr := c.zmanimCache.Set(date, locationID, z); setErr != nil {
		fmt.Printf("Warning: could not cache zmanim for %s: %v\n", date.Format("2006-01-02"), setErr)
	}
	return z, false, nil
}

// FetchCandlesYear retrieves a full year of candle lighting and havdalah times.
// tdate is the start date in "M/D/YYYY" format.
// fromCache is true when the response was served from disk.
// The returned map is keyed by "YYYY-MM-DD" in the location's local timezone.
func (c *Client) FetchCandlesYear(tdate string, loc types.Location, cfg types.Config) (map[string]CandleDay, bool, error) {
	rawURL := buildCandleURL(tdate, loc, cfg)
	var (
		body      []byte
		fromCache bool
		err       error
	)
	if !cfg.Refresh {
		if cached, ok := c.candleCache.Get(rawURL); ok {
			body, fromCache = cached, true
		}
	}
	if body == nil {
		body, err = c.httpGetWithUA(rawURL)
		if err != nil {
			return nil, false, fmt.Errorf("downloading candle ICS from chabad.org: %w", err)
		}
		if setErr := c.candleCache.Set(rawURL, body); setErr != nil {
			fmt.Printf("Warning: could not cache candle ICS: %v\n", setErr)
		}
	}
	days, err := parseCandleICS(body, loc.TZID)
	return days, fromCache, err
}

// ---- URL builders ----

func buildZmanimURL(date time.Time, loc types.Location, candles int) string {
	params := url.Values{}
	params.Set("bDef", "0")
	params.Set("before", strconv.Itoa(candles))
	params.Set("tdate", date.Format("2006-01-02"))
	setLocationParams(params, loc)
	return zmanimRSSURL + "?" + params.Encode()
}

// buildCandleURL constructs the candle ICS URL.
// tdate (M/D/YYYY) is appended raw to avoid percent-encoding of slashes.
func buildCandleURL(tdate string, loc types.Location, cfg types.Config) string {
	params := url.Values{}
	params.Set("before", strconv.Itoa(cfg.Candles))
	params.Set("bdef", "0")
	params.Set("weeks", "52")
	setLocationParams(params, loc)
	return candleICSURL + "?" + params.Encode() + "&tdate=" + tdate
}

func setLocationParams(params url.Values, loc types.Location) {
	if loc.ZIP != "" {
		params.Set("locationId", loc.ZIP)
		params.Set("locationType", "2")
		return
	}
	params.Set("locationType", "3")
	params.Set("coords", fmt.Sprintf("%f,%f", loc.Latitude, loc.Longitude))
	params.Set("tzname", ianaToChabad(loc.TZID))
	if loc.Name != "" {
		params.Set("n", loc.Name)
	}
}

// ianaToChabad converts an IANA timezone to chabad.org's format:
// "America/New_York" → "America*New_York"
func ianaToChabad(tzid string) string {
	return strings.ReplaceAll(tzid, "/", "*")
}

// ---- internal ----

func (c *Client) fetchRemoteZmanim(date time.Time, loc types.Location, candles int) (types.ZmanimDay, error) {
	body, err := c.httpGetWithUA(buildZmanimURL(date, loc, candles))
	if err != nil {
		return types.ZmanimDay{}, fmt.Errorf("downloading zmanim for %s: %w", date.Format("2006-01-02"), err)
	}
	return parseZmanimRSS(body, date, loc.TZID)
}

func (c *Client) httpGetWithUA(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(resp.Body)
}

// ---- RSS parsing ----

type rssDoc struct {
	Items []rssItem `xml:"channel>item"`
}

type rssItem struct {
	Title string `xml:"title"`
}

func parseZmanimRSS(body []byte, date time.Time, tzid string) (types.ZmanimDay, error) {
	var doc rssDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return types.ZmanimDay{}, fmt.Errorf("parsing zmanim RSS: %w", err)
	}
	tz, err := time.LoadLocation(tzid)
	if err != nil {
		return types.ZmanimDay{}, fmt.Errorf("loading timezone %q: %w", tzid, err)
	}
	z := types.ZmanimDay{Date: date}
	for _, item := range doc.Items {
		if err := applyRSSItem(item.Title, &z, date, tz); err != nil {
			fmt.Printf("Warning: skipping RSS item %q: %v\n", item.Title, err)
		}
	}
	return z, nil
}

func applyRSSItem(title string, z *types.ZmanimDay, date time.Time, tz *time.Location) error {
	title = strings.TrimSpace(title)

	// Shaah Zmanit has a duration value, not a clock time — handled separately.
	if strings.HasPrefix(title, "Shaah Zmanit") {
		return parseShaahZmanit(title, z)
	}

	// Standard format: "{label} - {H:MM AM/PM} -- ({date})"
	dashIdx := strings.Index(title, " - ")
	if dashIdx < 0 {
		return nil
	}
	prefix := strings.TrimSpace(title[:dashIdx])
	rest := title[dashIdx+3:]
	ddIdx := strings.Index(rest, " --")
	if ddIdx < 0 {
		return nil
	}
	timeStr := strings.TrimSpace(rest[:ddIdx])

	labels := splitLabel(prefix)
	for _, lbl := range labels {
		field, isEvent := classifyLabel(lbl)
		if field != "" {
			t, err := parseLocalTime(timeStr, date, field == "chatzos_halaila", tz)
			if err != nil {
				return err
			}
			setField(z, field, t)
		}
		if isEvent || field == "" {
			t, _ := parseLocalTime(timeStr, date, false, tz)
			if z.Events == nil {
				z.Events = make(map[string]time.Time)
			}
			z.Events[lbl] = t
		}
	}
	return nil
}

func parseShaahZmanit(title string, z *types.ZmanimDay) error {
	idx := strings.Index(title, " - ")
	if idx < 0 {
		return nil
	}
	raw := strings.Fields(strings.TrimSpace(title[idx+3:]))[0] // "60:30"
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	m, err1 := strconv.Atoi(parts[0])
	s, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return nil
	}
	z.ShaahZmanitMin = float64(m) + float64(s)/60.0
	return nil
}

func parseLocalTime(timeStr string, date time.Time, isChatzosHalaila bool, tz *time.Location) (time.Time, error) {
	t, err := time.Parse("3:04 PM", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing time %q: %w", timeStr, err)
	}
	d := date
	if isChatzosHalaila && t.Hour() < 12 {
		d = date.AddDate(0, 0, 1)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), 0, 0, tz), nil
}

func setField(z *types.ZmanimDay, field string, t time.Time) {
	switch field {
	case "alos":
		z.Alos = t
	case "misheyakir":
		z.Misheyakir = t
	case "sunrise":
		z.Sunrise = t
	case "latest_shema":
		z.LatestShema = t
	case "chatzos":
		z.Chatzos = t
	case "plag_hamincha":
		z.PlagHamincha = t
	case "shkiah":
		z.Shkiah = t
	case "tzeis":
		z.Tzeis = t
	case "chatzos_halaila":
		z.ChatzosHalaila = t
	}
}

// ---- ICS candle parsing ----

// parseCandleICS parses the chabad.org candle lighting ICS and returns a map
// keyed by "YYYY-MM-DD" in the location's local timezone. The string key
// avoids time.Time/*Location pointer-equality pitfalls (see patcher/candle.go).
func parseCandleICS(body []byte, tzid string) (map[string]CandleDay, error) {
	tz, err := time.LoadLocation(tzid)
	if err != nil {
		return nil, fmt.Errorf("loading timezone %q: %w", tzid, err)
	}
	cal, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing candle ICS: %w", err)
	}
	result := make(map[string]CandleDay)
	for _, ev := range cal.Events() {
		dtstart := ev.GetProperty(ical.ComponentPropertyDtStart)
		if dtstart == nil {
			continue
		}
		utcTime, err := time.Parse("20060102T150405Z", dtstart.Value)
		if err != nil {
			continue
		}
		localTime := utcTime.In(tz)
		dateKey := localTime.Format("2006-01-02")

		summary := ev.GetProperty(ical.ComponentPropertySummary)
		if summary == nil {
			continue
		}
		text := summary.Value
		day := result[dateKey]
		switch {
		case strings.Contains(text, "Light Holiday Candles"):
			day.Candles = localTime
			day.IsYomTov = true
			day.AfterHavdala = strings.Contains(text, " after")
		case strings.Contains(text, "Light Shabbat Candles"),
			strings.Contains(text, "Light Candles"):
			day.Candles = localTime
		case strings.Contains(text, "Shabbat Ends"),
			strings.Contains(text, "Holiday Ends"),
			strings.Contains(text, "Yom Tov Ends"):
			day.Havdalah = localTime
		default:
			continue
		}
		result[dateKey] = day
	}
	return result, nil
}
