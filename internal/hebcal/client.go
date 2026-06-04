// Package hebcal fetches and normalises Hebrew calendar data from the
// Hebcal JSON API.
//
// Language modes and Hebcal lg= mapping:
//
//	h   → lg=he          (Hebrew with nikud)
//	hn  → lg=he-x-NoNikud  (Hebrew, Hebcal strips nikud)
//	a   → lg=a           (Ashkenazi transliteration only)
//	ah  → lg=ah          (Ashkenazi + Hebrew, no nikud from API)
//	ahn → lg=ah          (Ashkenazi + Hebrew, nikud stripped client-side)
//	s   → lg=s           (Sefardi)
//	sh  → lg=sh          (Sefardi + Hebrew)
//	shn → lg=sh          (Sefardi + Hebrew, nikud stripped client-side)
//
// Nikud enrichment: for ah/ahn modes the Hebcal API does not return nikud
// in the hebrew field. A second request with lg=he is made to obtain nikud,
// and ev.Hebrew is replaced from that response. Both requests are cached via
// HTTPCache so the second call costs nothing after the first run.
//
// Leyning fields of interest in the API response (all under items[].leyning):
//   - haftarah          Ashkenazi standard haftarah
//   - haftarah_chabad   Chabad haftarah when it differs from Ashkenazi standard
//   - haftarah_sephardic  Sephardic haftarah when it differs
//   - ashkenazi_standard  Same as haftarah (Ashkenazi standard)
//   - ashkenazi_litvish   Lithuanian Ashkenazi variant; potentially closest to
//     Chabad Yiddish pronunciation, but unverified against actual Chabad usage
package hebcal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/toobuntu/zman-didan/internal/cache"
	"github.com/toobuntu/zman-didan/internal/types"
)

const (
	baseURL      = "https://www.hebcal.com/hebcal"
	httpCacheTTL = 24 * time.Hour
)

// langToAPIParam maps didan lang codes to Hebcal lg= parameter values.
// Codes not listed here are passed directly.
var langToAPIParam = map[string]string{
	"h":   "he",
	"hn":  "he-x-NoNikud",
	"ahn": "ah",
	"shn": "sh",
}

var slugRe = regexp.MustCompile(`/[hs]/([a-z][a-z0-9-]+)-\d{4}`)

// Client fetches from the Hebcal API.
type Client struct {
	httpClient *http.Client
	httpCache  *cache.HTTPCache
}

// NewClient returns a Client with sensible timeout defaults.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		httpCache:  cache.NewHTTP(httpCacheTTL),
	}
}

// FetchYear retrieves all events and the resolved location for cfg.
// Returns fromCache=true when the primary response was served from disk.
func (c *Client) FetchYear(cfg types.Config) (events []types.HebcalEvent, loc types.Location, fromCache bool, err error) {
	u := buildURL(cfg, "")
	body, fromCache, err := c.get(u, cfg.Refresh)
	if err != nil {
		return nil, types.Location{}, false, err
	}
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, types.Location{}, false, fmt.Errorf("parsing Hebcal response: %w", err)
	}
	loc = normalizeLocation(resp.Location, cfg)
	events = make([]types.HebcalEvent, 0, len(resp.Items))
	for _, item := range resp.Items {
		ev, err := normalizeEvent(item, loc.TZID)
		if err != nil {
			fmt.Printf("Warning: skipping event %q: %v\n", item.Title, err)
			continue
		}
		events = append(events, ev)
	}

	// For Ashkenazi modes, the API does not return nikud in the hebrew field.
	// Fetch again with lg=he to replace ev.Hebrew with nikud-bearing text.
	if needsNikudEnrichment(cfg.Lang) {
		c.enrichHebrew(events, cfg)
	}

	return events, loc, fromCache, nil
}

// needsNikudEnrichment reports whether a second lg=he fetch is needed to
// populate ev.Hebrew with nikud.
func needsNikudEnrichment(lang string) bool {
	return lang == "ah" || lang == "ahn"
}

// enrichHebrew fetches the calendar again with lg=he and replaces ev.Hebrew
// by position. Both fetches are cached so the second call is free on repeat runs.
func (c *Client) enrichHebrew(events []types.HebcalEvent, cfg types.Config) {
	u := buildURL(cfg, "he")
	body, _, err := c.get(u, cfg.Refresh)
	if err != nil {
		fmt.Printf("Warning: nikud enrichment fetch failed: %v\n", err)
		return
	}
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return
	}
	// Replace ev.Hebrew by index position — both API calls use identical
	// parameters except lg=, so event ordering is guaranteed to match.
	for i := range events {
		if i >= len(resp.Items) {
			break
		}
		events[i].Hebrew = resp.Items[i].Hebrew
	}
}

func buildURL(cfg types.Config, langOverride string) string {
	lg := cfg.Lang
	if langOverride != "" {
		lg = langOverride
	} else if mapped, ok := langToAPIParam[lg]; ok {
		lg = mapped
	}

	params := url.Values{}
	params.Set("v", "1")
	params.Set("cfg", "json")
	params.Set("i", "off")
	params.Set("lg", lg)
	params.Set("leyning", "on")
	params.Set("maj", "on")
	params.Set("min", "on")
	params.Set("mf", "on")
	params.Set("ss", "on")
	params.Set("mod", "off")
	params.Set("nx", "on")
	params.Set("s", "on")
	params.Set("c", "on")
	params.Set("M", "on")
	params.Set("b", fmt.Sprintf("%d", cfg.Candles))

	if cfg.UsingDateRange() {
		params.Set("start", cfg.StartDate.Format("2006-01-02"))
		params.Set("end", cfg.EndDate.Format("2006-01-02"))
	} else {
		params.Set("year", fmt.Sprintf("%d", cfg.Year))
		params.Set("yt", "H")
	}

	switch cfg.LocationMode {
	case types.LocationLatLon:
		params.Set("geo", "pos")
		params.Set("latitude", fmt.Sprintf("%f", cfg.Lat))
		params.Set("longitude", fmt.Sprintf("%f", cfg.Lon))
		params.Set("tzid", cfg.TZID)
	case types.LocationGeoName:
		params.Set("geo", "geoname")
		params.Set("geonameid", cfg.GeoNameID)
	default:
		params.Set("geo", "zip")
		params.Set("zip", cfg.ZIP)
	}

	return baseURL + "?" + params.Encode()
}

// get retrieves rawURL, using the HTTPCache unless refresh is true.
// Returns the body and whether it was served from cache.
// Go's default HTTP transport adds Accept-Encoding: gzip automatically and
// handles transparent decompression; no explicit header is needed.
func (c *Client) get(rawURL string, refresh bool) ([]byte, bool, error) {
	if !refresh {
		if body, ok := c.httpCache.Get(rawURL); ok {
			return body, true, nil
		}
	}
	resp, err := c.httpClient.Get(rawURL)
	if err != nil {
		return nil, false, fmt.Errorf("downloading from hebcal.com: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("hebcal.com returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}
	if setErr := c.httpCache.Set(rawURL, body); setErr != nil {
		fmt.Printf("Warning: could not cache Hebcal response: %v\n", setErr)
	}
	return body, false, nil
}

// ---- JSON response shapes ----

type apiResponse struct {
	Location apiLocation `json:"location"`
	Items    []apiItem   `json:"items"`
}

type apiLocation struct {
	ZIP       string  `json:"zip"`
	TZID      string  `json:"tzid"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Title     string  `json:"title"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
}

type apiItem struct {
	Title    string                     `json:"title"`
	Date     string                     `json:"date"`
	Category string                     `json:"category"`
	Subcat   string                     `json:"subcat,omitempty"`
	Hebrew   string                     `json:"hebrew,omitempty"`
	Memo     string                     `json:"memo,omitempty"`
	Link     string                     `json:"link,omitempty"`
	HDate    string                     `json:"hdate,omitempty"`
	Leyning  map[string]json.RawMessage `json:"leyning,omitempty"`
}

func normalizeLocation(loc apiLocation, cfg types.Config) types.Location {
	zip := loc.ZIP
	if zip == "" {
		zip = cfg.ZIP
	}
	tzid := loc.TZID
	if tzid == "" {
		tzid = cfg.TZID
	}
	if tzid == "" {
		tzid = "America/New_York"
	}
	city := loc.Title
	if city == "" {
		city = loc.City
	}
	if city == "" {
		city = cfg.Name
	}
	return types.Location{
		ZIP: zip, TZID: tzid, Latitude: loc.Latitude, Longitude: loc.Longitude,
		City: city, Country: loc.Country, Name: cfg.Name,
	}
}

func normalizeEvent(item apiItem, tzid string) (types.HebcalEvent, error) {
	date, allDay, err := parseDate(item.Date, tzid)
	if err != nil {
		return types.HebcalEvent{}, fmt.Errorf("parsing date %q: %w", item.Date, err)
	}
	return types.HebcalEvent{
		Date: date, AllDay: allDay, Category: item.Category, Subcat: item.Subcat,
		Title: item.Title, Hebrew: item.Hebrew, Memo: item.Memo, Link: item.Link,
		HDate: item.HDate, Slug: extractSlug(item.Link), Leyning: normalizeLeyning(item.Leyning),
	}, nil
}

// normalizeLeyning extracts string-valued fields from the raw leyning map.
// Non-string values (triennial object, haftaraNumV integer) are silently skipped.
//
// haftarah_chabad is populated by Hebcal when the Chabad reading differs from
// Ashkenazi standard (live on www.hebcal.com as of 2026-03-30). Non-null only
// when Chabad custom differs; null means Chabad follows the standard haftarah.
func normalizeLeyning(m map[string]json.RawMessage) *types.Leyning {
	if len(m) == 0 {
		return nil
	}
	aliyot := make(map[string]string)
	l := &types.Leyning{Aliyot: aliyot}
	for k, raw := range m {
		if k == "triennial" {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		switch k {
		case "torah":
			l.Torah = s
		case "haftarah":
			l.Haftarah = s
		case "haftarah_chabad":
			l.HaftarahChabad = s
		case "maftir":
			l.Maftir = s
		default:
			aliyot[k] = s
		}
	}
	return l
}

func extractSlug(link string) string {
	m := slugRe.FindStringSubmatch(link)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimRight(m[1], "-")
}

func parseDate(s, tzid string) (time.Time, bool, error) {
	if strings.Contains(s, "T") {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, false, err
		}
		loc, err := time.LoadLocation(tzid)
		if err != nil {
			return time.Time{}, false, fmt.Errorf("unknown timezone %q: %w", tzid, err)
		}
		return t.In(loc), false, nil
	}
	loc, err := time.LoadLocation(tzid)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("unknown timezone %q: %w", tzid, err)
	}
	t, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return time.Time{}, false, err
	}
	return t, true, nil
}
