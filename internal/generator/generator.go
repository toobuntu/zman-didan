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

// httpCacheTTL is the TTL for the HTTP response cache. The Hebcal and chabad.org
// responses are stable for a given set of request parameters, so 7 days is
// conservative. --refresh is the explicit override.
const httpCacheTTL = 7 * 24 * time.Hour

// useAshkenazi reports whether the lang mode requires Ashkenazi substitutions.
func useAshkenazi(lang string) bool {
	switch lang {
	case "a", "ah", "ahn":
		return true
	}
	return false
}

// stripNikudForLang reports whether nikud should be stripped client-side.
func stripNikudForLang(lang string) bool {
	switch lang {
	case "hn", "ahn", "shn":
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

	httpCache := cache.NewHTTP(httpCacheTTL)

	// 1. Hebcal calendar.
	hc := hebcal.NewClient()
	events, loc, calFromCache, err := hc.FetchYear(cfg)
	if err != nil {
		return fmt.Errorf("Hebcal: %w", err)
	}
	if cfg.UsingDateRange() {
		fmt.Printf("[%s] %d events %s → %s — %s (%s)\n",
			hostTag(calFromCache, "hebcal.com"),
			len(events),
			cfg.StartDate.Format("2006-01-02"),
			cfg.EndDate.Format("2006-01-02"),
			loc.City, loc.TZID)
	} else {
		fmt.Printf("[%s] %d events Hebrew year %d — %s (%s)\n",
			hostTag(calFromCache, "hebcal.com"),
			len(events), cfg.Year, loc.City, loc.TZID)
	}

	for i := range events {
		if events[i].Description == "" {
			events[i].Description = events[i].Memo
		}
	}
	events = dropTimedHebcalFastEvents(events)

	// 2. Chabad candle lighting + havdalah.
	zmCache := cache.New()
	if !cfg.Refresh {
		if err := zmCache.Prune(); err != nil {
			fmt.Printf("Warning: cache prune: %v\n", err)
		}
	}
	cc := chabad.NewClient(zmCache)
	tdate := candleStartDate(events, loc)
	candleTimes, candleFromCache, err := cc.FetchCandlesYear(tdate, loc, cfg)
	if err != nil {
		return fmt.Errorf("Chabad candle ICS: %w", err)
	}
	fmt.Printf("[%s] %d candle/havdalah entries\n",
		hostTag(candleFromCache, "chabad.org"), len(candleTimes))

	// 3. Zmanim per relevant date (per-date cache).
	zmanimDates := collectZmanimDates(events, loc.TZID)
	total := len(zmanimDates)
	zmanimMap := make(map[string]types.ZmanimDay, total)
	var nCached, nDownloaded int
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
			nDownloaded++
		}
	}
	switch {
	case nDownloaded > 0 && nCached > 0:
		fmt.Printf("[chabad.org+cache] %d zmanim (%d downloaded, %d cached)\n", total, nDownloaded, nCached)
	case nDownloaded > 0:
		fmt.Printf("[chabad.org] %d zmanim\n", nDownloaded)
	default:
		fmt.Printf("[cache] %d zmanim\n", nCached)
	}

	// 4–12. Pipeline stages.
	patcher.PatchCandles(events, candleTimes, loc.TZID, cfg.Tosfos)
	attacher.AttachZmanim(events, zmanimMap, loc.TZID)
	events = append(events, fastday.Build(events, zmanimMap, cfg.LocationID(), loc.TZID)...)
	alarm.Rebuild(events)

	if err := haftorah.Patch(events); err != nil {
		fmt.Printf("Warning: haftorah patch: %v\n", err)
	}

	stripNikud := stripNikudForLang(cfg.Lang)
	if useAshkenazi(cfg.Lang) {
		transliterator.Apply(events, stripNikud)
	}

	cleaner.Clean(events)

	rangeStart, rangeEnd := calendarDateRange(events, loc.TZID)
	special, sdFromCache, err := specialdates.Merge(loc.TZID, stripNikud, useAshkenazi(cfg.Lang), rangeStart, rangeEnd, httpCache, cfg.Refresh)
	if err != nil {
		fmt.Printf("Warning: special dates: %v\n", err)
	} else {
		events = append(events, special...)
		fmt.Printf("[%s] %d Yomei d'Pagra events\n",
			hostTag(sdFromCache, "hebcal.com"), len(special))
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

// hostTag returns the source host or "cache" for a progress line tag.
func hostTag(fromCache bool, host string) string {
	if fromCache {
		return "cache"
	}
	return host
}

// dropTimedHebcalFastEvents removes timed fast-begin/end events from Hebcal.
// fastday.Build synthesises replacements. The all-day fast event is kept.
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

// calendarDateRange returns midnight of the first and last event calendar dates.
// Midnight boundaries ensure all-day events (which parse as midnight) are not
// excluded by timed events that fall later on the same start date.
func calendarDateRange(events []types.HebcalEvent, tzid string) (start, end time.Time) {
	tz, _ := time.LoadLocation(tzid)
	for _, ev := range events {
		d := ev.Date.In(tz)
		dayMidnight := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, tz)
		if start.IsZero() || dayMidnight.Before(start) {
			start = dayMidnight
		}
		if end.IsZero() || dayMidnight.After(end) {
			end = dayMidnight
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
