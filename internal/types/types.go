/*
 * SPDX-FileCopyrightText: Copyright 2026 Todd Schulman
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

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
	TZID      string // IANA timezone identifier, e.g. "America/New_York"
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
	Link        string // not written to iCal output; retained for internal use
	HDate       string
	Slug        string
	Leyning     *Leyning
	Alarms      []Alarm
	UID         string
}

// Leyning holds Torah reading data attached to parashat events.
//
// HaftarahChabad is populated from the Hebcal API's haftarah_chabad field
// once hebcal-rest-api v6.4.1+ is deployed to www.hebcal.com. When present
// it takes precedence over the embedded haftorah_chabad.json table. The field
// is empty until deployment; haftorah.Patch falls back to the embedded table.
type Leyning struct {
	Torah          string
	Haftarah       string
	HaftarahChabad string // from Hebcal API haftarah_chabad field; preferred when non-empty
	Maftir         string
	Aliyot         map[string]string
}

// ZmanimDay holds halachic times for a single date from chabad.org RSS.
// All Time fields are in the location's local timezone.
// ChatzosHalaila falls after midnight — its calendar date is date+1.
//
// Misheyakir label differs by day type:
//   - Weekday:    "Earliest Tallit and Tefillin (Misheyakir)"
//   - Shabbos/YT: "Earliest Tallit (Misheyakir)"
//
// Both variants are valid — Tallis is worn on Shabbos. The RSS uses the
// shorter label because Tefillin is not worn, but the zman itself is
// fully relevant for the earliest time to don a Tallis.
//
// Events holds times from RSS labels that do not map to a canonical field:
// chametz deadlines on Erev Pesach, fast-start markers, contextual "ends"
// labels that also carry semantic meaning, etc. Keys are normalised labels.
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
	Events         map[string]time.Time // normalised label → time
}

// LocationMode describes how the user specified their location.
type LocationMode int

const (
	LocationZIP     LocationMode = iota // --zip
	LocationLatLon                      // --lat + --lon + --tzid + --name
	LocationGeoName                     // --geoname (GeoNames.org numeric ID)
)

// Config holds the validated, parsed CLI arguments for a single run.
type Config struct {
	LocationMode LocationMode
	ZIP          string
	Lat          float64
	Lon          float64
	TZID         string
	GeoNameID    string
	Name         string

	Year      int
	StartDate time.Time
	EndDate   time.Time

	Lang      string
	Candles   int // minutes before shkiah for candle lighting (default 25)
	Tosfos    int // minutes added to havdala for tosfos Shabbos (default 4)
	Output    string
	Refresh   bool // bypass all caches
	Emojis    bool // prefix SUMMARY with Hebcal-style emoji (default true)
	NoClobber bool // refuse to overwrite existing output file
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
