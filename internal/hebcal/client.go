// Package hebcal fetches and normalises Hebrew calendar data from the
// Hebcal JSON API.
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

	"github.com/toobuntu/zman-didan/internal/types"
)

const baseURL = "https://www.hebcal.com/hebcal"

var langAPIParam = map[string]string{
	"ah-x-NoNikud": "ah",
}

var slugRe = regexp.MustCompile(`/[hs]/([a-z][a-z0-9-]+)-\d{4}`)

// Client fetches from the Hebcal API.
type Client struct {
	httpClient *http.Client
}

// NewClient returns a Client with sensible timeout defaults.
func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

// FetchYear retrieves all events and the resolved location for cfg.
func (c *Client) FetchYear(cfg types.Config) ([]types.HebcalEvent, types.Location, error) {
	u := buildURL(cfg)
	body, err := c.get(u)
	if err != nil {
		return nil, types.Location{}, err
	}
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, types.Location{}, fmt.Errorf("parsing Hebcal response: %w", err)
	}
	loc := normalizeLocation(resp.Location, cfg)
	events := make([]types.HebcalEvent, 0, len(resp.Items))
	for _, item := range resp.Items {
		ev, err := normalizeEvent(item, loc.TZID)
		if err != nil {
			fmt.Printf("Warning: skipping event %q: %v\n", item.Title, err)
			continue
		}
		events = append(events, ev)
	}
	return events, loc, nil
}

func buildURL(cfg types.Config) string {
	lg := cfg.Lang
	if mapped, ok := langAPIParam[lg]; ok {
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

func (c *Client) get(rawURL string) ([]byte, error) {
	resp, err := c.httpClient.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetching Hebcal: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Hebcal returned HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
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
	// Leyning mixes string aliyah keys with non-string values; decoded raw.
	Leyning map[string]json.RawMessage `json:"leyning,omitempty"`
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
	// For lat/lon mode, Hebcal may not return a city name; fall back to --name.
	if city == "" {
		city = cfg.Name
	}
	return types.Location{
		ZIP:       zip,
		TZID:      tzid,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		City:      city,
		Country:   loc.Country,
		Name:      cfg.Name,
	}
}

func normalizeEvent(item apiItem, tzid string) (types.HebcalEvent, error) {
	date, allDay, err := parseDate(item.Date, tzid)
	if err != nil {
		return types.HebcalEvent{}, fmt.Errorf("parsing date %q: %w", item.Date, err)
	}
	return types.HebcalEvent{
		Date:     date,
		AllDay:   allDay,
		Category: item.Category,
		Subcat:   item.Subcat,
		Title:    item.Title,
		Hebrew:   item.Hebrew,
		Memo:     item.Memo,
		Link:     item.Link,
		HDate:    item.HDate,
		Slug:     extractSlug(item.Link),
		Leyning:  normalizeLeyning(item.Leyning),
	}, nil
}

// normalizeLeyning extracts string-valued fields from the raw leyning map.
// Non-string values are silently skipped:
//   - "triennial": nested object (Conservative cycle — not used)
//   - "haftaraNumV": integer verse count
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
