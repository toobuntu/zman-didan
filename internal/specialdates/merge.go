// Package specialdates fetches, transforms, and merges Chabad Yomei d'Pagra
// from the hebcal-hosted chabad-special-dates.ics feed.
//
// The feed covers multiple years. Events are filtered to those within the
// provided date range to avoid including entries from other Hebrew years.
package specialdates

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"

	"github.com/toobuntu/zman-didan/internal/embeddata"
	"github.com/toobuntu/zman-didan/internal/transliterator"
	"github.com/toobuntu/zman-didan/internal/types"
)

const specialDatesURL = "https://download.hebcal.com/ical/chabad-special-dates.ics"

type dpgraEntry struct {
	Summary string `json:"summary"`
}

// rebbeNames maps verbose name forms used in chabad-special-dates.ics titles
// and descriptions to standard Chabad honorific names.
// Applied before transliteration so the final output uses these forms.
var rebbeNames = map[string]string{
	// Patronymic forms (appear in titles)
	"R. Schneur Zalman of Liadi":      "the Alter Rebbe",
	"R. DovBer Schneuri":              "the Mitteler Rebbe",
	"R. Dovber Schneuri":              "the Mitteler Rebbe",
	"R. Menachem Mendel Schneersohn":  "the Tzemach Tzeddek",
	"R. Shmuel Schneersohn":           "the Rebbe Maharash",
	"R. Shalom DovBer Schneersohn":    "the Rebbe Rashab",
	"R. Sholom Dovber Schneersohn":    "the Rebbe Rashab",
	"R. Sholom DovBer of Lubavitch":   "the Rebbe Rashab",
	"R. Yosef Yitzchak Schneersohn":   "the Frierdiker Rebbe",
	"R. Menachem Mendel Schneerson":   "the Lubavitcher Rebbe",
	// Keep these unchanged but explicit so they don't get renamed inadvertently
	"R. Levi Yitzchak Schneerson":     "R. Levi Yitzchak Schneerson",
	"Rebbetzin Chana Schneerson":      "Rebbetzin Chana",
}

// yahrzeitRe matches the Hebcal yahrzeit description format:
// "Hebcal joins you in remembering NAME, whose Nth Yahrzeit occurs on DAY,
// DATE, corresponding to the ORD of HMONTH, HYEAR."
var yahrzeitRe = regexp.MustCompile(
	`Hebcal joins you in remembering (.+?), whose (\d+)(?:st|nd|rd|th) Yahrzeit occurs on .+?, corresponding to the (\d+)(?:st|nd|rd|th) of ([A-Za-z]+), (\d+)`)

// birthdayRe matches Hebcal birthday/event description:
// "Birthday of NAME occurs on DAY, DATE, corresponding to the ORD of HMONTH, HYEAR."
// Also handles "Yahrtzeit of NAME" and similar patterns.
var birthdayRe = regexp.MustCompile(
	`(.+?) occurs on .+?, corresponding to the (\d+)(?:st|nd|rd|th) of ([A-Za-z]+), (\d+)`)

// Merge fetches special dates, filters to the given range, applies
// transliteration, and appends Chitas summaries.
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

	normalizeRebbeNames(events)
	transliterator.Apply(events, stripNikud)

	if err := applyChitasSummaries(events); err != nil {
		fmt.Printf("Warning: Chitas summaries: %v\n", err)
	}
	return events, nil
}

func normalizeRebbeNames(events []types.HebcalEvent) {
	for i := range events {
		ev := &events[i]
		for verbose, honorific := range rebbeNames {
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

// reformatDescription replaces Hebcal's verbose boilerplate with a concise
// format appropriate for a calendar description.
//
// Yahrzeit: "Hebcal joins you in remembering NAME, whose Nth Yahrzeit occurs
//   on DAY, DATE, corresponding to the ORD of HMONTH, HYEAR."
//   → "Histalkus of NAME on DAY_NUM HMONTH DEATH_YEAR (N years ago)"
//
// Birthday/other: "EVENT occurs on DAY, DATE, corresponding to the ORD of HMONTH, HYEAR."
//   → "EVENT on DAY_NUM HMONTH HYEAR"
func reformatDescription(s string) string {
	// Yahrzeit pattern.
	if m := yahrzeitRe.FindStringSubmatch(s); m != nil {
		name := strings.TrimSpace(m[1])
		n, _ := strconv.Atoi(m[2])
		dayNum := m[3]
		hmonth := m[4]
		hyear, _ := strconv.Atoi(m[5])
		deathYear := hyear - n
		return fmt.Sprintf("Histalkus of %s on %s %s %d (%d years ago)",
			name, dayNum, hmonth, deathYear, n)
	}

	// Birthday / other occurrence pattern — strip the "occurs on ..." clause.
	if m := birthdayRe.FindStringSubmatch(s); m != nil {
		event := strings.TrimSpace(m[1])
		dayNum := m[2]
		hmonth := m[3]
		hyear := m[4]
		return fmt.Sprintf("%s · %s %s %s", event, dayNum, hmonth, hyear)
	}

	// Unknown pattern: strip leading Hebcal branding and boilerplate.
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
