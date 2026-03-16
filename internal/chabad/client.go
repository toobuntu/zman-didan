// Package chabad fetches Chabad-authoritative zmanim and candle lighting times.
//
// Zmanim source:         chabad.org RSS feed, one request per date, cached.
// Candle lighting source: chabad.org ICS endpoint, one request per year.
//
// ChatzosHalaila: the RSS feed for date D returns midnight (Chatzot HaLailah)
// as an AM time on D+1 (the night of D→D+1). parseLocalTime detects this and
// constructs the Time value on D+1.
//
// Geolocation: for lat/lon locations, chabad.org uses (undocumented):
//   - locationType=3
//   - coords=LAT,LON      comma-separated decimal degrees
//   - tzname=IANA*TZ      IANA timezone with "/" replaced by "*"
//   - n=NAME              display name — required for lat/lon requests
//
// tdate encoding: the candle ICS endpoint requires literal slashes in
// M/D/YYYY. url.Values.Encode() would percent-encode them to %2F, causing
// HTTP 403. tdate is therefore appended raw after encoding all other params.
//
// User-Agent: chabad.org blocks Go's default "Go-http-client/1.1" with a
// 403. A generic browser User-Agent is sent instead.
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
	zmanimRSSURL = "https://www.chabad.org/tools/rss/zmanim.xml"
	candleICSURL = "https://www.chabad.org/calendar/candlelighting/candlelighting.ics.asp"

	// userAgent mimics a standard browser. chabad.org returns 403 for the
	// default Go HTTP client user agent.
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var rssToField = map[string]string{
	"Dawn (Alot Hashachar)":                    "alos",
	"Earliest Tallit and Tefillin (Misheyakir)": "misheyakir",
	"Sunrise (Hanetz Hachamah)":                 "sunrise",
	"Latest Shema":                              "latest_shema",
	"Midday (Chatzot Hayom)":                    "chatzos",
	"Plag Hamincha":                             "plag_hamincha",
	"Sunset (Shkiah)":                           "shkiah",
	"Nightfall (Tzeit Hakochavim)":              "tzeis",
	"Midnight (Chatzot HaLailah)":               "chatzos_halaila",
	"Shaah Zmanit (proportional hour)":          "shaah_zmanit",
}

// CandleDay holds the candle lighting and havdalah times for one calendar date.
type CandleDay struct {
	Candles  time.Time
	Havdalah time.Time
}

// Client fetches from chabad.org.
type Client struct {
	httpClient *http.Client
	cache      *cache.ZmanimCache
}

// NewClient returns a ready Client.
func NewClient(c *cache.ZmanimCache) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      c,
	}
}

// FetchZmanimInZone returns zmanim for date. fromCache reports whether the
// result was served from the local cache (true) or fetched from chabad.org.
// A zero Misheyakir in the returned ZmanimDay is expected on Shabbos —
// chabad.org omits that zman when Tefillin is not worn.
func (c *Client) FetchZmanimInZone(date time.Time, loc types.Location, cfg types.Config) (z types.ZmanimDay, fromCache bool, err error) {
	cacheKey := cfg.LocationID()
	if !cfg.Refresh {
		if cached, ok := c.cache.Get(date, cacheKey); ok {
			return cached, true, nil
		}
	}
	z, err = c.fetchRemote(date, loc, cfg.Candles)
	if err != nil {
		return types.ZmanimDay{}, false, err
	}
	if setErr := c.cache.Set(date, cacheKey, z); setErr != nil {
		fmt.Printf("Warning: could not cache zmanim for %s: %v\n", date.Format("2006-01-02"), setErr)
	}
	return z, false, nil
}

// FetchCandlesYear fetches a full year of candle lighting and havdalah times.
// tdate is the start date in "M/D/YYYY" format.
func (c *Client) FetchCandlesYear(tdate string, loc types.Location, cfg types.Config) (map[time.Time]CandleDay, error) {
	rawURL := buildCandleURL(tdate, loc, cfg)
	body, err := c.httpGet(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetching Chabad candle ICS: %w", err)
	}
	return parseCandleICS(body, loc.TZID)
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

// setLocationParams populates location query parameters.
// ZIP: locationId + locationType=2.
// Lat/lon: locationType=3 + coords + tzname + n (required display name).
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

func (c *Client) fetchRemote(date time.Time, loc types.Location, candles int) (types.ZmanimDay, error) {
	rawURL := buildZmanimURL(date, loc, candles)
	body, err := c.httpGet(rawURL)
	if err != nil {
		return types.ZmanimDay{}, fmt.Errorf("fetching zmanim for %s: %w", date.Format("2006-01-02"), err)
	}
	return parseZmanimRSS(body, date, loc.TZID)
}

func (c *Client) httpGet(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", rawURL, err)
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

	if strings.HasPrefix(title, "Shaah Zmanit") {
		idx := strings.Index(title, " - ")
		if idx < 0 {
			return nil
		}
		raw := strings.Fields(strings.TrimSpace(title[idx+3:]))[0]
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

	dashIdx := strings.Index(title, " - ")
	if dashIdx < 0 {
		return nil
	}
	prefix := title[:dashIdx]
	fieldKey := matchPrefix(prefix)
	if fieldKey == "" {
		return nil
	}
	rest := title[dashIdx+3:]
	ddIdx := strings.Index(rest, " --")
	if ddIdx < 0 {
		return nil
	}
	timeStr := strings.TrimSpace(rest[:ddIdx])
	t, err := parseLocalTime(timeStr, date, fieldKey == "chatzos_halaila", tz)
	if err != nil {
		return err
	}
	setField(z, fieldKey, t)
	return nil
}

func matchPrefix(s string) string {
	for prefix, field := range rssToField {
		if strings.HasPrefix(s, prefix) {
			return field
		}
	}
	return ""
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

func parseCandleICS(body []byte, tzid string) (map[time.Time]CandleDay, error) {
	tz, err := time.LoadLocation(tzid)
	if err != nil {
		return nil, fmt.Errorf("loading timezone %q: %w", tzid, err)
	}
	cal, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing candle ICS: %w", err)
	}
	result := make(map[time.Time]CandleDay)
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
		dateKey := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), 0, 0, 0, 0, tz)

		summary := ev.GetProperty(ical.ComponentPropertySummary)
		if summary == nil {
			continue
		}
		text := summary.Value
		day := result[dateKey]
		switch {
		case strings.Contains(text, "Light Candles"),
			strings.Contains(text, "Light Holiday Candles"),
			strings.Contains(text, "Light Shabbat Candles"):
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
