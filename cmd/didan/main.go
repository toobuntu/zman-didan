// Command didan generates a Chabad-minhag iCalendar file.
//
// Usage:
//
//	didan generate --year 5786 --zip 17601
//	didan generate --year 5786 --zip 17601 --lang ah
//	didan generate --year 5786 --lat 40.0732 --lon -76.3209 --tzid America/New_York --name Lancaster
//	didan generate --start 2025-11-01 --end 2025-11-30 --zip 17601
//
// Build:
//
//	go build -o bin/didan ./cmd/didan
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/toobuntu/zman-didan/internal/generator"
	"github.com/toobuntu/zman-didan/internal/types"
)

var validLangs = map[string]struct{}{
	"he":           {},
	"he-x-NoNikud": {},
	"a":            {},
	"ah":           {},
	"ah-x-NoNikud": {},
	"s":            {},
	"sh":           {},
}

func main() {
	root := &cobra.Command{
		Use:   "didan",
		Short: "Generate a Chabad-minhag iCalendar file",
	}

	var (
		year      int
		startStr  string
		endStr    string
		zip       string
		lat       float64
		lon       float64
		tzid      string
		name      string
		geoname   string
		lang      string
		candles   int
		tosfos    int
		output    string
		refresh   bool
		emojis    bool
		noClobber bool
	)

	generate := &cobra.Command{
		Use:   "generate",
		Short: "Generate the calendar",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, ok := validLangs[lang]; !ok {
				return fmt.Errorf("--lang must be one of: he, he-x-NoNikud, a, ah, ah-x-NoNikud, s, sh")
			}
			if _, err := os.Stat(output); os.IsNotExist(err) {
				return fmt.Errorf("output directory does not exist: %s", output)
			}

			cfg := types.Config{
				ZIP:       zip,
				TZID:      tzid,
				Name:      name,
				Lang:      lang,
				Candles:   candles,
				Tosfos:    tosfos,
				Output:    output,
				Refresh:   refresh,
				Emojis:    emojis,
				NoClobber: noClobber,
			}

			// ---- Location mode ----
			hasZIP := zip != ""
			hasLatLon := cmd.Flags().Changed("lat") || cmd.Flags().Changed("lon")
			hasGeoName := geoname != ""

			switch {
			case boolCount(hasZIP, hasLatLon, hasGeoName) == 0:
				return fmt.Errorf("specify a location: --zip, --lat/--lon/--tzid/--name, or --geoname")
			case boolCount(hasZIP, hasLatLon, hasGeoName) > 1:
				return fmt.Errorf("--zip, --lat/--lon, and --geoname are mutually exclusive")
			case hasLatLon:
				if !cmd.Flags().Changed("lat") || !cmd.Flags().Changed("lon") {
					return fmt.Errorf("--lat and --lon must be used together")
				}
				if tzid == "" {
					return fmt.Errorf("--tzid is required with --lat/--lon (e.g. America/New_York)")
				}
				if name == "" {
					return fmt.Errorf("--name is required with --lat/--lon (display name for chabad.org)")
				}
				if _, err := time.LoadLocation(tzid); err != nil {
					return fmt.Errorf("--tzid %q is not a valid IANA timezone: %w", tzid, err)
				}
				cfg.LocationMode = types.LocationLatLon
				cfg.Lat = lat
				cfg.Lon = lon
			case hasGeoName:
				cfg.LocationMode = types.LocationGeoName
				cfg.GeoNameID = geoname
			default:
				cfg.LocationMode = types.LocationZIP
			}

			// ---- Date range ----
			hasYear := cmd.Flags().Changed("year")
			hasRange := startStr != "" || endStr != ""

			switch {
			case hasYear && hasRange:
				return fmt.Errorf("--year and --start/--end are mutually exclusive")
			case !hasYear && !hasRange:
				return fmt.Errorf("specify a time period: --year or --start/--end")
			case hasRange:
				if startStr == "" || endStr == "" {
					return fmt.Errorf("--start and --end must be used together")
				}
				start, err := time.Parse("2006-01-02", startStr)
				if err != nil {
					return fmt.Errorf("--start: invalid date %q (want YYYY-MM-DD)", startStr)
				}
				end, err := time.Parse("2006-01-02", endStr)
				if err != nil {
					return fmt.Errorf("--end: invalid date %q (want YYYY-MM-DD)", endStr)
				}
				if !end.After(start) {
					return fmt.Errorf("--end must be after --start")
				}
				cfg.StartDate = start
				cfg.EndDate = end
			default:
				cfg.Year = year
			}

			return generator.Run(cfg)
		},
	}

	f := generate.Flags()

	f.IntVar(&year, "year", 0, "Hebrew year (e.g. 5786); mutually exclusive with --start/--end")
	f.StringVar(&startStr, "start", "", "Start date YYYY-MM-DD; mutually exclusive with --year")
	f.StringVar(&endStr, "end", "", "End date YYYY-MM-DD; mutually exclusive with --year")

	f.StringVar(&zip, "zip", "", "US ZIP code")
	f.Float64Var(&lat, "lat", 0, "Latitude (decimal); requires --lon, --tzid, --name")
	f.Float64Var(&lon, "lon", 0, "Longitude (decimal); requires --lat, --tzid, --name")
	f.StringVar(&tzid, "tzid", "", "IANA timezone (e.g. America/New_York); required with --lat/--lon")
	f.StringVar(&name, "name", "", "Location display name; required with --lat/--lon")
	f.StringVar(&geoname, "geoname", "", "GeoNames.org numeric ID")

	f.StringVar(&lang, "lang", "he", "Language: he (default), he-x-NoNikud, a, ah, ah-x-NoNikud, s, sh")
	f.IntVar(&candles, "candles", 25, "Minutes before shkiah for candle lighting")
	f.IntVar(&tosfos, "tosfos", 4, "Minutes added to havdala (tosfos Shabbos/Yom Tov); 0 to disable")
	f.StringVar(&output, "output", ".", "Output directory")
	f.BoolVar(&refresh, "refresh", false, "Bypass all caches; re-fetch everything from the network")
	f.BoolVar(&emojis, "emojis", true, "Prefix event SUMMARY with emoji (🕯️, 🍏🍯, etc.); --emojis=false to disable")
	f.BoolVar(&noClobber, "no-clobber", false, "Refuse to overwrite an existing output file")

	root.AddCommand(generate)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func boolCount(vals ...bool) int {
	n := 0
	for _, v := range vals {
		if v {
			n++
		}
	}
	return n
}
