// Package fastday synthesises standalone begin and end VEVENT entries for
// fast days. The existing all-day holiday event is left in place.
//
// Only all-day Hebcal holiday events with Subcat "fast" are processed.
// Hebcal also returns timed fast-begin and fast-end events (when c=on)
// with the same Subcat; filtering to AllDay avoids synthesising duplicate
// events from those timed entries.
package fastday

import (
	"fmt"
	"strings"
	"time"

	"github.com/toobuntu/zman-didan/internal/types"
)

// Build returns new events for each fast day:
//   - "[Fast Name] Begins" at alos (or prev-day shkiah for Tisha B'Av / Yom Kippur)
//   - "[Fast Name] Ends"   at tzeis
func Build(events []types.HebcalEvent, zmanimMap map[string]types.ZmanimDay, locationID, tzid string) []types.HebcalEvent {
	tz, _ := time.LoadLocation(tzid)
	var synthesised []types.HebcalEvent

	for _, ev := range events {
		if ev.Subcat != "fast" || !ev.AllDay {
			continue
		}
		key := ev.Date.In(tz).Format("2006-01-02")
		z, ok := zmanimMap[key]
		if !ok {
			fmt.Printf("Warning: no zmanim for fast day %s on %s\n", ev.Title, key)
			continue
		}
		prevKey := ev.Date.In(tz).AddDate(0, 0, -1).Format("2006-01-02")
		prevZ, hasPrev := zmanimMap[prevKey]

		begin, end := fastTimes(ev, z, prevZ, hasPrev)
		name := fastBaseName(ev.Title)
		synthesised = append(synthesised,
			beginEvent(ev, name, begin, locationID),
			endEvent(ev, name, end, locationID),
		)
	}
	return synthesised
}

func fastTimes(ev types.HebcalEvent, z, prevZ types.ZmanimDay, hasPrev bool) (begin, end time.Time) {
	end = z.Tzeis
	t := strings.ToLower(ev.Title)
	if (strings.Contains(t, "tisha b'av") || strings.Contains(t, "yom kippur")) &&
		hasPrev && !prevZ.Shkiah.IsZero() {
		begin = prevZ.Shkiah
	} else {
		begin = z.Alos
	}
	return
}

// fastBaseName extracts a clean name from a Hebcal event title by stripping
// leading emoji and the Hebrew portion (after the "/" separator).
// "✡️ Tzom Gedaliah / צוֹם גְּדַלְיָה" → "Tzom Gedaliah"
func fastBaseName(title string) string {
	fields := strings.Fields(title)
	var out []string
	for _, f := range fields {
		if isEmoji(f) {
			continue
		}
		if f == "/" {
			break
		}
		out = append(out, f)
	}
	return strings.TrimSpace(strings.Join(out, " "))
}

// isEmoji reports whether s consists entirely of non-ASCII Unicode characters
// above U+2000 (the start of general punctuation / symbol blocks).
func isEmoji(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r <= 0x2000 {
			return false
		}
	}
	return true
}

func beginEvent(src types.HebcalEvent, name string, t time.Time, locationID string) types.HebcalEvent {
	return types.HebcalEvent{
		Date: t, AllDay: false, Category: "fast-begin",
		Title: name + " Begins", Hebrew: src.Hebrew,
		UID: fmt.Sprintf("didan-%s-fast-begin-%s", t.Format("2006-01-02"), locationID),
		Alarms: []types.Alarm{
			{TriggerMinutes: -120, Description: "Event reminder"},
			{TriggerMinutes: -30, Description: "Event reminder"},
		},
	}
}

func endEvent(src types.HebcalEvent, name string, t time.Time, locationID string) types.HebcalEvent {
	return types.HebcalEvent{
		Date: t, AllDay: false, Category: "fast-end",
		Title: name + " Ends", Hebrew: src.Hebrew,
		UID: fmt.Sprintf("didan-%s-fast-end-%s", t.Format("2006-01-02"), locationID),
		Alarms: []types.Alarm{
			{TriggerMinutes: -15, Description: "Event reminder"},
			{TriggerMinutes: 0, Description: "Event reminder"},
		},
	}
}
