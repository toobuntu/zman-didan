// Package types defines the shared data structures used throughout the
// didan pipeline.
package types

import (
	"fmt"
	"time"
)

// Location holds the user's geographic context resolved from the Hebcal API.
type Location struct {
	ZIP       string
	TZID      string  // IANA timezone identifier, e.g. "America/New_York"
	Latitude  float64
	Longitude float64
	City      string
	Country   string
	// Name is used as the display name (n=) for lat/lon requests to
	// chabad.org, which requires it for non-ZIP locations.
	Name string
}

// Alarm describes a single VALARM trigger relative to the event start.
// TriggerMinutes is negative for "before event", 0 for "at event time".
type Alarm struct {
	TriggerMinutes int
	Description    string
}

// HebcalEvent is a normalised, mutable event that flows through the pipeline.
type HebcalEvent struct {
	Date        time.Time
	AllDay      bool
	Category    string // "candles", "havdalah", "holiday", "parashat", "rosh_chodesh", …
	Subcat      string // "major", "minor", "fast", "special-shabbat", …
	Title       string
	Hebrew      string
	Description string
	Memo        string
	Link        string // not written to output; retained for internal use only
	HDate       string
	Slug        string
	Leyning     *Leyning
	Alarms      []Alarm
	UID         string
}

// Leyning holds Torah reading data attached to parashat events.
type Leyning struct {
	Torah    string
	Haftarah string
	Maftir   string
	Aliyot   map[string]string
}

// ZmanimDay holds halachic times for a single date from chabad.org RSS.
// All Time fields are in the location's local timezone.
// ChatzosHalaila falls after midnight — its calendar date is date+1.
// Fields may be zero if chabad.org did not include them for that day
// (e.g. Misheyakir is omitted on Shabbos since Tefillin is not worn).
type ZmanimDay struct {
	Date           time.Time
	Alos           time.Time
	Misheyakir     time.Time
	Sunrise        time.Time
	LatestShema    time.Time
	Chatzos        time.Time
	PlagHamincha   time.Time
	Shkiah         time.Time
	Tzeis          time.Time
	ChatzosHalaila time.Time
	ShaahZmanitMin float64
}

// LocationMode describes how the user specified their location.
type LocationMode int

const (
	LocationZIP      LocationMode = iota // --zip
	LocationLatLon                       // --lat + --lon + --tzid + --name
	LocationGeoName                      // --geoname (GeoNames.org numeric ID)
)

// Config holds the validated, parsed CLI arguments for a single run.
type Config struct {
	LocationMode LocationMode
	ZIP          string
	Lat          float64
	Lon          float64
	TZID         string
	GeoNameID    string
	Name         string // display name — required for LatLon (chabad.org n= param)

	Year      int
	StartDate time.Time
	EndDate   time.Time

	Lang    string
	Candles int
	Output  string
	Refresh bool
	Emojis  bool // include Hebcal-style emoji prefixes in SUMMARY
}

// UsingDateRange reports whether an explicit date range was specified.
func (c Config) UsingDateRange() bool {
	return !c.StartDate.IsZero() && !c.EndDate.IsZero()
}

// LocationID returns a short string identifying the location for cache keys
// and output filenames.
func (c Config) LocationID() string {
	switch c.LocationMode {
	case LocationLatLon:
		return fmt.Sprintf("%.4f_%.4f", c.Lat, c.Lon)
	case LocationGeoName:
		return "g" + c.GeoNameID
	default:
		return c.ZIP
	}
}
