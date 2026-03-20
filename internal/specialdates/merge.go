// Package specialdates fetches, transforms, and merges Chabad Yomei d'Pagra
// from the hebcal-hosted chabad-special-dates.ics feed.
//
// The feed covers multiple years. Events are filtered to the provided date
// range to avoid including entries from other Hebrew years.
//
// Rebbe names are normalized using the embedded rebbes.json table, which
// provides both name→honorific mapping and Hebrew birth/death years. The
// death year enables disambiguation when two figures share a verbose name
// (e.g. two rebbes both named Menachem Mendel), and birth year enables
// "(N years ago)" in birthday descriptions.
//
// Operation order matters:
//  1. convertEvent → reformatDescription uses verbose names for birth-year
//     lookup, since normalization has not yet run.
//  2. normalizeNames replaces verbose names with honorifics in Title and
//     Description.
//  3. transliterator.Apply runs Ashkenazi substitutions.
//  4. applyChitasSummaries appends Chitas text by Title (now normalized).
package specialdates

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	ical "github.com/arran4/golang-ical"

	"github.com/toobuntu/zman-didan/internal/embeddata"
	"github.com/toobuntu/zman-didan/internal/transliterator"
	"github.com/toobuntu/zman-didan/internal/types"
)

const specialDatesURL = "https://download.hebcal.com/ical/chabad-special-dates.ics"

// rebbeEntry mirrors one record in rebbes.json.
// Only birth_year and death_year are used at runtime; the full date fields
// are preserved in the JSON for reference.
type rebbeEntry struct {
	Honorific    string   `json:"honorific"`
	VerboseNames []string `json:"verbose_names"`
	BirthYear    int      `json:"birth_year"`
	DeathYear    int      `json:"death_year"`
}

type dpgraEntry struct {
	Summary string `json:"summary"`
}

// yahrzeitRe matches Hebcal yahrzeit description boilerplate.
// Groups: (1) verbose name, (2) N (ordinal), (3) day, (4) Hebrew month,
//
//	(5) Hebrew year of observance.
var yahrzeitRe = regexp.MustCompile(
	`Hebcal joins you in remembering (.+?), whose (\d+)(?:st|nd|rd|th) Yahrzeit occurs on .+?, corresponding to the (\d+)(?:st|nd|rd|th) of ([A-Za-z]+), (\d+)`)

// birthdayRe matches Hebcal birthday/event description boilerplate.
// Groups: (1) event text, (2) day, (3) Hebrew month, (4) Hebrew year of observance.
var birthdayRe = regexp.MustCompile(
	`(.+?) occurs on .+?, corresponding to the (\d+)(?:st|nd|rd|th) of ([A-Za-z]+), (\d+)`)

// ---- lazy-loaded rebbe table ----

var (
	rebbeOnce sync.Once
	rebbeData []rebbeEntry
)

func getRebbe() []rebbeEntry {
	rebbeOnce.Do(func() {
		if err := json.Unmarshal(embeddata.RebbesJSON, &rebbeData); err != nil {
			fmt.Printf("Warning: parsing rebbes.json: %v\n", err)
			rebbeData = nil
		}
	})
	return rebbeData
}

// findByVerboseName returns the rebbeEntry whose verbose_names contains name.
// When deathYear > 0, the match is further required to agree on death year,
// enabling disambiguation when two figures share a verbose name.
func findByVerboseName(name string, deathYear int) *rebbeEntry {
	for i := range getRebbe() {
		e := &getRebbe()[i]
		for _, v := range e.VerboseNames {
			if v == name {
				if deathYear > 0 && e.DeathYear != deathYear {
					continue
				}
				return e
			}
		}
	}
	return nil
}

// buildNameMap returns a flat map of verbose name → honorific for all rebbes.
func buildNameMap() map[string]string {
	m := make(map[string]string)
	for _, e := range getRebbe() {
		for _, v := range e.VerboseNames {
			m[v] = e.Honorific
		}
	}
	return m
}

// Merge fetches special dates, filters to the given range, and applies the
// transformation pipeline: rebbe normalization → transliteration → Chitas.
func Merge(tzid string, stripNikud bool, rangeStart, rangeEnd time.Time) ([]types.HebcalEvent, error) {
	body, err := fetch()
	if err != nil {
		return nil, err
	}
	events, err := parse(body, tzid)
	if err != nil {
		return nil, err
	}

	if !rangeStart.IsZero() && !rangeEnd.IsZero() {
		filtered := events[:0]
		for _, ev := range events {
			if !ev.Date.Before(rangeStart) && !ev.Date.After(rangeEnd) {
				filtered = append(filtered, ev)
			}
		}
		events = filtered
	}

	normalizeNames(events, buildNameMap())
	transliterator.Apply(events, stripNikud)

	if err := applyChitasSummaries(events); err != nil {
		fmt.Printf("Warning: Chitas summaries: %v\n", err)
	}
	return events, nil
}

func normalizeNames(events []types.HebcalEvent, names map[string]string) {
	for i := range events {
		ev := &events[i]
		for verbose, honorific := range names {
			ev.Title = strings.ReplaceAll(ev.Title, verbose, honorific)
			ev.Description = strings.ReplaceAll(ev.Description, verbose, honorific)
		}
	}
}

func fetch() ([]byte, error) {
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Get(specialDatesURL)
	if err != nil {
		return nil, fmt.Errorf("fetching special dates ICS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("special dates ICS returned HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func parse(body []byte, tzid string) ([]types.HebcalEvent, error) {
	tz, err := time.LoadLocation(tzid)
	if err != nil {
		return nil, fmt.Errorf("loading timezone %q: %w", tzid, err)
	}
	cal, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing special dates ICS: %w", err)
	}
	var events []types.HebcalEvent
	for _, ev := range cal.Events() {
		if he := convertEvent(ev, tz); he != nil {
			events = append(events, *he)
		}
	}
	return events, nil
}

func convertEvent(ev *ical.VEvent, tz *time.Location) *types.HebcalEvent {
	dtstart := ev.GetProperty(ical.ComponentPropertyDtStart)
	if dtstart == nil {
		return nil
	}
	t, err := time.ParseInLocation("20060102", dtstart.Value, tz)
	if err != nil {
		t, err = time.Parse("20060102T150405Z", dtstart.Value)
		if err != nil {
			return nil
		}
		t = t.In(tz)
	}
	title, desc, uid := "", "", ""
	if p := ev.GetProperty(ical.ComponentPropertySummary); p != nil {
		title = p.Value
	}
	if p := ev.GetProperty(ical.ComponentPropertyDescription); p != nil {
		desc = reformatDescription(p.Value)
	}
	if p := ev.GetProperty(ical.ComponentPropertyUniqueId); p != nil {
		uid = p.Value
	}
	return &types.HebcalEvent{
		Date: t, AllDay: true, Category: "holiday", Subcat: "chabad",
		Title: title, Description: desc, UID: uid,
	}
}

// reformatDescription replaces Hebcal's verbose boilerplate with concise text.
//
// This runs before normalizeNames, so verbose rebbe names are still present.
// rebbes.json is keyed by verbose_names, enabling birth/death year lookup
// before normalization converts them to honorifics.
//
// Yahrzeit:
//
//	"Hebcal joins you in remembering NAME, whose Nth Yahrzeit occurs on
//	DAY, DATE, corresponding to the ORD of HMONTH, HYEAR."
//
// The computed death year (HYEAR − N) is passed to findByVerboseName to
// disambiguate figures sharing a verbose name.
//
//	→ "Histalkus of NAME on DAY_NUM HMONTH DEATH_YEAR (N years ago)"
//
// After normalizeNames runs, NAME in the stored description becomes the honorific.
//
// Birthday:
//
//	"Birthday of X occurs on DAY, DATE, corresponding to the ORD of HMONTH, HYEAR."
//	→ "Birthday of X · DAY_NUM HMONTH BIRTH_YEAR (N years ago)"  (birth year known)
//	→ "Birthday of X · DAY_NUM HMONTH OBSERVANCE_YEAR"           (birth year unknown)
func reformatDescription(s string) string {
	if m := yahrzeitRe.FindStringSubmatch(s); m != nil {
		name := strings.TrimSpace(m[1])
		n, _ := strconv.Atoi(m[2])
		dayNum, hmonth := m[3], m[4]
		hyear, _ := strconv.Atoi(m[5])
		deathYear := hyear - n
		return fmt.Sprintf("Histalkus of %s on %s %s %d (%d years ago)",
			name, dayNum, hmonth, deathYear, n)
	}
	if m := birthdayRe.FindStringSubmatch(s); m != nil {
		event := strings.TrimSpace(m[1])
		dayNum, hmonth := m[2], m[3]
		hyear, _ := strconv.Atoi(m[4])
		// Strip "Birthday of " prefix to get the verbose name for lookup.
		verboseName := strings.TrimPrefix(event, "Birthday of ")
		if e := findByVerboseName(verboseName, 0); e != nil && e.BirthYear > 0 {
			n := hyear - e.BirthYear
			return fmt.Sprintf("%s · %s %s %d (%d years ago)",
				event, dayNum, hmonth, e.BirthYear, n)
		}
		return fmt.Sprintf("%s · %s %s %d", event, dayNum, hmonth, hyear)
	}
	// Unknown pattern: strip leading boilerplate only.
	if idx := strings.Index(s, " begins at sundown on "); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	s = strings.TrimPrefix(s, "Hebcal joins you in remembering ")
	return strings.TrimSpace(s)
}

func applyChitasSummaries(events []types.HebcalEvent) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(embeddata.YomeiDpagraJSON, &raw); err != nil {
		return fmt.Errorf("parsing yomei_dpagra.json: %w", err)
	}
	for i := range events {
		ev := &events[i]
		rawEntry, ok := raw[ev.Title]
		if !ok {
			continue
		}
		var entry dpgraEntry
		if err := json.Unmarshal(rawEntry, &entry); err != nil {
			continue
		}
		if entry.Summary == "" {
			continue
		}
		if ev.Description != "" {
			ev.Description += "\n\n" + entry.Summary
		} else {
			ev.Description = entry.Summary
		}
	}
	return nil
}
