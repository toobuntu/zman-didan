// Package generator orchestrates the full didan pipeline.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/toobuntu/zman-didan/internal/alarm"
	"github.com/toobuntu/zman-didan/internal/attacher"
	"github.com/toobuntu/zman-didan/internal/cache"
	"github.com/toobuntu/zman-didan/internal/chabad"
	"github.com/toobuntu/zman-didan/internal/cleaner"
	"github.com/toobuntu/zman-didan/internal/fastday"
	"github.com/toobuntu/zman-didan/internal/haftorah"
	"github.com/toobuntu/zman-didan/internal/hebcal"
	"github.com/toobuntu/zman-didan/internal/icalwriter"
	"github.com/toobuntu/zman-didan/internal/patcher"
	"github.com/toobuntu/zman-didan/internal/specialdates"
	"github.com/toobuntu/zman-didan/internal/transliterator"
	"github.com/toobuntu/zman-didan/internal/types"
)

func useAshkenazi(lang string) bool {
	switch lang {
	case "a", "ah", "ah-x-NoNikud":
		return true
	}
	return false
}

// Run executes the full pipeline for cfg and writes the .ics file.
func Run(cfg types.Config) error {
	outPath := outputPath(cfg)
	if cfg.NoClobber {
		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("output file already exists (use --refresh or remove it): %s", outPath)
		}
	}

	// 1. Hebcal.
	if cfg.UsingDateRange() {
		fmt.Printf("Fetching Hebcal calendar %s → %s...\n",
			cfg.StartDate.Format("2006-01-02"), cfg.EndDate.Format("2006-01-02"))
	} else {
		fmt.Printf("Fetching Hebcal calendar for Hebrew year %d...\n", cfg.Year)
	}
	hc := hebcal.NewClient()
	events, loc, err := hc.FetchYear(cfg)
	if err != nil {
		return fmt.Errorf("Hebcal fetch: %w", err)
	}
	fmt.Printf("  %d events — %s (%s)\n", len(events), loc.City, loc.TZID)

	for i := range events {
		if events[i].Description == "" {
			events[i].Description = events[i].Memo
		}
	}

	// Drop timed fast-begin/end events that Hebcal returns via c=on. We
	// synthesise our own timed events in fastday.Build using Chabad-
	// authoritative times from chabad.org. Keeping both would produce
	// duplicate events with slightly different times and no alarms on the
	// Hebcal version. The all-day fast event is kept; it drives fastday.Build.
	events = dropTimedHebcalFastEvents(events)

	// 2. Chabad candle lighting + havdalah.
	// The candle ICS is always fetched from chabad.org; it is not cached.
	fmt.Println("Fetching candle lighting from chabad.org...")
	zmCache := cache.New()
	if !cfg.Refresh {
		if err := zmCache.Prune(); err != nil {
			fmt.Printf("Warning: cache prune: %v\n", err)
		}
	}
	cc := chabad.NewClient(zmCache)
	tdate := candleStartDate(events, loc)
	candleTimes, err := cc.FetchCandlesYear(tdate, loc, cfg)
	if err != nil {
		return fmt.Errorf("Chabad candle ICS: %w", err)
	}
	fmt.Printf("  %d candle/havdalah entries\n", len(candleTimes))

	// 3. Zmanim per relevant date (disk-cached; reports cache vs network).
	zmanimDates := collectZmanimDates(events, loc.TZID)
	total := len(zmanimDates)
	zmanimMap := make(map[string]types.ZmanimDay, total)
	var nCached, nFetched int
	for _, d := range zmanimDates {
		z, fromCache, err := cc.FetchZmanimInZone(d, loc, cfg)
		if err != nil {
			fmt.Printf("  Warning: zmanim unavailable for %s: %v\n", d.Format("2006-01-02"), err)
			continue
		}
		zmanimMap[d.Format("2006-01-02")] = z
		if fromCache {
			nCached++
		} else {
			nFetched++
		}
	}
	switch {
	case nFetched > 0 && nCached > 0:
		fmt.Printf("  %d zmanim (%d from chabad.org, %d from cache)\n", total, nFetched, nCached)
	case nFetched > 0:
		fmt.Printf("  Fetched %d zmanim from chabad.org\n", nFetched)
	default:
		fmt.Printf("  %d zmanim (all from cache)\n", nCached)
	}

	// 4–12. Pipeline stages.
	patcher.PatchCandles(events, candleTimes, loc.TZID, cfg.Tosfos)
	attacher.AttachZmanim(events, zmanimMap, loc.TZID)
	events = append(events, fastday.Build(events, zmanimMap, cfg.LocationID(), loc.TZID)...)
	alarm.Rebuild(events)

	if err := haftorah.Patch(events); err != nil {
		fmt.Printf("Warning: haftorah patch: %v\n", err)
	}

	stripNikud := cfg.Lang == "ah-x-NoNikud"
	if useAshkenazi(cfg.Lang) {
		transliterator.Apply(events, stripNikud)
	}

	cleaner.Clean(events)

	// Special dates: always fetched from Hebcal; filtered to calendar range.
	fmt.Println("Fetching Chabad special dates from Hebcal...")
	rangeStart, rangeEnd := calendarDateRange(events, loc.TZID)
	special, err := specialdates.Merge(loc.TZID, stripNikud, rangeStart, rangeEnd)
	if err != nil {
		fmt.Printf("Warning: special dates: %v\n", err)
	} else {
		events = append(events, special...)
		fmt.Printf("  %d Yomei d'Pagra events merged\n", len(special))
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Date.Before(events[j].Date)
	})

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating output file %s: %w", outPath, err)
	}
	defer f.Close()

	if err := icalwriter.Write(f, events, loc, calendarYear(cfg), cfg.Lang, cfg.Emojis); err != nil {
		return fmt.Errorf("writing iCal: %w", err)
	}
	fmt.Printf("Written: %s\n", outPath)
	return nil
}

// dropTimedHebcalFastEvents removes timed fast-begin and fast-end events that
// Hebcal returns via c=on alongside the all-day fast event. These are identified
// by being timed (AllDay=false) with Subcat=="fast" — the same combination that
// fastday.Build uses to synthesise its own versions with Chabad-authoritative
// times and configured alarms.
func dropTimedHebcalFastEvents(events []types.HebcalEvent) []types.HebcalEvent {
	out := make([]types.HebcalEvent, 0, len(events))
	for _, ev := range events {
		if !ev.AllDay && ev.Subcat == "fast" {
			continue
		}
		out = append(out, ev)
	}
	return out
}

func collectZmanimDates(events []types.HebcalEvent, tzid string) []time.Time {
	tz, _ := time.LoadLocation(tzid)
	seen := make(map[string]bool)
	var dates []time.Time

	add := func(t time.Time) {
		local := t.In(tz)
		d := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, tz)
		key := d.Format("2006-01-02")
		if !seen[key] {
			seen[key] = true
			dates = append(dates, d)
		}
	}

	for _, ev := range events {
		if !needsZmanim(ev) {
			continue
		}
		add(ev.Date)
		if ev.Subcat == "fast" && isTishaBavOrYK(ev) {
			add(ev.Date.AddDate(0, 0, -1))
		}
	}

	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}

func needsZmanim(ev types.HebcalEvent) bool {
	switch ev.Category {
	case "candles", "havdalah":
		return true
	}
	if ev.Subcat == "fast" {
		return true
	}
	return isChanuka(ev) || isPesachNight(ev) || isTishaBav(ev)
}

func isChanuka(ev types.HebcalEvent) bool {
	return ev.Category == "holiday" && containsFold(ev.Title, "chanuk")
}

func isPesachNight(ev types.HebcalEvent) bool {
	return ev.Category == "holiday" &&
		containsFold(ev.Title, "pesach") &&
		containsFold(ev.HDate, "15 Nisan")
}

func isTishaBav(ev types.HebcalEvent) bool {
	return ev.Category == "holiday" && containsFold(ev.Title, "tisha b'av")
}

func isTishaBavOrYK(ev types.HebcalEvent) bool {
	t := strings.ToLower(ev.Title)
	return strings.Contains(t, "tisha b'av") || strings.Contains(t, "yom kippur")
}

func candleStartDate(events []types.HebcalEvent, loc types.Location) string {
	tz, _ := time.LoadLocation(loc.TZID)
	for _, ev := range events {
		if ev.Category == "candles" {
			d := ev.Date.In(tz)
			return fmt.Sprintf("%d/%d/%d", int(d.Month()), d.Day(), d.Year())
		}
	}
	return "9/1/2025"
}

func calendarDateRange(events []types.HebcalEvent, tzid string) (start, end time.Time) {
	tz, _ := time.LoadLocation(tzid)
	for _, ev := range events {
		d := ev.Date.In(tz)
		if start.IsZero() || d.Before(start) {
			start = d
		}
		if end.IsZero() || d.After(end) {
			end = d
		}
	}
	return
}

func outputPath(cfg types.Config) string {
	locPart := cfg.LocationID()
	var name string
	if cfg.UsingDateRange() {
		name = fmt.Sprintf("didan_%s_%s_%s.ics",
			cfg.StartDate.Format("20060102"),
			cfg.EndDate.Format("20060102"),
			locPart)
	} else {
		name = fmt.Sprintf("didan_%d_%s.ics", cfg.Year, locPart)
	}
	return filepath.Join(cfg.Output, name)
}

func calendarYear(cfg types.Config) int {
	if cfg.UsingDateRange() {
		return cfg.StartDate.Year()
	}
	return cfg.Year
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
