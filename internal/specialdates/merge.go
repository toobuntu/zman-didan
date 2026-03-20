// Package specialdates fetches, transforms, and merges Chabad Yomei d'Pagra
// from the hebcal-hosted chabad-special-dates.ics feed.
//
// The feed covers multiple years. Events are filtered to the provided date
// range to avoid including entries from other Hebrew years.
//
// Pipeline order within this package matters:
//  1. parse → convertEvent → reformatDescription: uses verbose names for
//     huledes/histalkus year lookup; normalization has not yet run.
//  2. normalizeNames: replaces verbose names with honorifics in Title and
//     Description.
//  3. shortenTitles: rewrites verbose SUMMARY to short calendar-friendly form.
//  4. transliterator.Apply: Ashkenazi substitutions.
//  5. applyChitasSummaries: appends Chitas text by (now-normalized) Title.
//
// verbose_names in rebbes.json serve two roles:
//   - buildNameMap: string replacement in normalizeNames (needs exact feed strings)
//   - findByVerboseName: year lookup in reformatDescription (name extracted by regex)
//
// Both roles require knowing the exact strings the Hebcal feed uses.
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
// Only HuledesYear, DOBGregorian, HistalkusYear, and HistalkusGregorian are
// used at runtime. The Hebrew date strings are preserved in JSON for reference.
type rebbeEntry struct {
	Honorific          string   `json:"honorific"`
	VerboseNames       []string `json:"verbose_names"`
	HuledesYear        int      `json:"huledes_year"`
	DOBGregorian       string   `json:"dob_gregorian"`
	HistalkusYear      int      `json:"histalkus_year"`
	HistalkusGregorian string   `json:"histalkus_gregorian"`
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

// yahrzeitTitleRe matches the chabad-special-dates.ics SUMMARY for yahrzeits,
// which follow the pattern "the NAME's [Nth] Yahrzeit".
// The ordinal ("106th") is optional — some entries omit it.
// Group 1: the honorific name (without "the ").
var yahrzeitTitleRe = regexp.MustCompile(`^the (.+?)'s (?:\d+\w+ )?Yahrzeit`)

// birthdayTitleRe matches "Birthday of [the] NAME" after normalizeNames.
// Group 1: the name (with or without leading "the ").
var birthdayTitleRe = regexp.MustCompile(`^Birthday of (?:the )?(.+)$`)

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
// When histalkusYear > 0, also requires that field to match, disambiguating
// two figures who share a verbose name.
func findByVerboseName(name string, histalkusYear int) *rebbeEntry {
	for i := range getRebbe() {
		e := &getRebbe()[i]
		for _, v := range e.VerboseNames {
			if v == name {
				if histalkusYear > 0 && e.HistalkusYear != histalkusYear {
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
// full transformation pipeline.
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
	shortenTitles(events)
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

// shortenTitles rewrites verbose SUMMARY values into short calendar titles.
// Runs after normalizeNames so honorifics are already in place.
//
// The chabad-special-dates.ics SUMMARY format is:
//   - "the Rebbe Rashab's 106th Yahrzeit (2nd of Nisan)"
//   - "Birthday of the Rebbe (11th of Nisan)"
//
// After normalizeNames, verbose rebbe names become honorifics. After
// shortenTitles:
//   - "the Rebbe Rashab's 106th Yahrzeit (2nd of Nisan)" → "Rebbe Rashab's Histalkus"
//   - "Birthday of the Rebbe (11th of Nisan)"            → "Rebbe's Birthday"
//
// Leading "the" is dropped in possessive constructions — "the Rebbe's Birthday"
// is natural speech but looks odd as a standalone calendar title.
func shortenTitles(events []types.HebcalEvent) {
	for i := range events {
		ev := &events[i]
		ev.Title = shortenTitle(ev.Title)
	}
}

// shortenTitle converts one verbose feed SUMMARY into a short calendar title.
// The feed appends " (date)" parentheticals such as " (2nd of Nisan)"; these
// are stripped before pattern matching so ordinal-plus-date variants are handled.
func shortenTitle(s string) string {
	// Strip trailing " (date)" parenthetical before matching, e.g. " (2nd of Nisan)".
	base := s
	if idx := strings.LastIndex(s, " ("); idx > 0 && strings.HasSuffix(s, ")") {
		base = s[:idx]
	}

	if m := yahrzeitTitleRe.FindStringSubmatch(base); m != nil {
		return m[1] + "'s Histalkus"
	}
	if m := birthdayTitleRe.FindStringSubmatch(base); m != nil {
		return m[1] + "'s Birthday"
	}
	return s
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
// Runs before normalizeNames, so verbose rebbe names are still present.
//
// Histalkus (yahrzeit):
//
//	"Hebcal joins you in remembering NAME, whose Nth Yahrzeit occurs on
//	DAY, DATE, corresponding to the ORD of HMONTH, HYEAR."
//	→ "Histalkus of NAME at age AGE on DAY HMONTH HISTALKUS_YEAR (DOB_GREG) — N years ago"
//
// The parenthetical holds the Gregorian DOB (birth date), giving biographical
// context on the yahrzeit. After normalizeNames, NAME becomes the honorific.
//
// Birthday:
//
//	"Birthday of X occurs on DAY, DATE, corresponding to the ORD of HMONTH, HYEAR."
//	→ "Birthday of X on DAY HMONTH HULEDES_YEAR (DOB_GREG) — N years ago"  (year known)
//	→ "Birthday of X on DAY HMONTH OBSERVANCE_YEAR"                         (year unknown)
func reformatDescription(s string) string {
	if m := yahrzeitRe.FindStringSubmatch(s); m != nil {
		name := strings.TrimSpace(m[1])
		n, _ := strconv.Atoi(m[2])
		dayNum, hmonth := m[3], m[4]
		hyear, _ := strconv.Atoi(m[5])
		histalkusYear := hyear - n

		age, dobParens := "", ""
		if e := findByVerboseName(name, histalkusYear); e != nil {
			if e.HuledesYear > 0 {
				age = fmt.Sprintf(" at age %d", histalkusYear-e.HuledesYear)
			}
			if e.DOBGregorian != "" {
				dobParens = " (" + formatGregorian(e.DOBGregorian) + ")"
			}
		}
		return fmt.Sprintf("Histalkus of %s%s on %s %s %d%s — %d years ago",
			name, age, dayNum, hmonth, histalkusYear, dobParens, n)
	}

	if m := birthdayRe.FindStringSubmatch(s); m != nil {
		event := strings.TrimSpace(m[1])
		dayNum, hmonth := m[2], m[3]
		hyear, _ := strconv.Atoi(m[4])
		verboseName := strings.TrimPrefix(event, "Birthday of ")
		if e := findByVerboseName(verboseName, 0); e != nil && e.HuledesYear > 0 {
			n := hyear - e.HuledesYear
			dobParens := ""
			if e.DOBGregorian != "" {
				dobParens = " (" + formatGregorian(e.DOBGregorian) + ")"
			}
			return fmt.Sprintf("Birthday of %s on %s %s %d%s — %d years ago",
				verboseName, dayNum, hmonth, e.HuledesYear, dobParens, n)
		}
		return fmt.Sprintf("Birthday of %s on %s %s %d", verboseName, dayNum, hmonth, hyear)
	}

	// Unknown pattern: strip leading boilerplate only.
	if idx := strings.Index(s, " begins at sundown on "); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	s = strings.TrimPrefix(s, "Hebcal joins you in remembering ")
	return strings.TrimSpace(s)
}

// formatGregorian parses "2006-01-02" and returns "02 Jan 2006" (zero-padded
// day, abbreviated month, four-digit year).
func formatGregorian(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("02 Jan 2006")
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
