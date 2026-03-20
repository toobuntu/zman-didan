// Package patcher replaces hebcal's placeholder candle lighting and havdalah
// times with Chabad-authoritative times from the chabad.org ICS feed.
//
// Tosfos Shabbos: the havdalah time is advanced by cfg.Tosfos minutes beyond
// the chabad.org tzeis time, fulfilling the halachic requirement to add
// mundane time to Shabbos at its end. The default is 4 minutes.
// The event Title is rebuilt to reflect both the corrected time and the
// offset, e.g. "Havdalah (+4): 8:07 PM". The transliterator converts
// "Havdalah" → "Havdala" downstream, so the base word is left in its
// Hebcal form here.
package patcher

import (
	"fmt"
	"strings"
	"time"

	"github.com/toobuntu/zman-didan/internal/chabad"
	"github.com/toobuntu/zman-didan/internal/types"
)

// PatchCandles replaces DTSTART on candle lighting and havdalah events with
// Chabad-authoritative times. Havdalah is offset by tosfos minutes.
func PatchCandles(events []types.HebcalEvent, candleTimes map[time.Time]chabad.CandleDay, tzid string, tosfos int) {
	tz, err := time.LoadLocation(tzid)
	if err != nil {
		tz = time.UTC
	}
	for i := range events {
		ev := &events[i]
		key := midnight(ev.Date, tz)
		switch ev.Category {
		case "candles":
			if day, ok := candleTimes[key]; ok && !day.Candles.IsZero() {
				ev.Date = day.Candles
				ev.AllDay = false
				ev.Title = rebuildCandleTitle(ev.Title, day.Candles.In(tz))
			}
		case "havdalah":
			if day, ok := candleTimes[key]; ok && !day.Havdalah.IsZero() {
				t := day.Havdalah.Add(time.Duration(tosfos) * time.Minute)
				ev.Date = t
				ev.AllDay = false
				ev.Title = rebuildHavdalaTitle(ev.Title, t.In(tz), tosfos)
			}
		}
	}
}

// rebuildCandleTitle reconstructs the candle lighting SUMMARY with the
// patched time, replacing the stale Hebcal time.
// "Candle lighting: 6:36pm" → "Candle lighting: 6:52 PM"
func rebuildCandleTitle(original string, t time.Time) string {
	return fmt.Sprintf("%s: %s", titleBase(original), t.Format("3:04 PM"))
}

// rebuildHavdalaTitle reconstructs the havdalah SUMMARY with the patched
// time and tosfos offset.
// "Havdalah: 8:03pm", tosfos=4 → "Havdalah (+4): 8:07 PM"
// "Havdalah: 8:03pm", tosfos=0 → "Havdalah: 8:03 PM"
func rebuildHavdalaTitle(original string, t time.Time, tosfos int) string {
	base := titleBase(original)
	if tosfos > 0 {
		return fmt.Sprintf("%s (+%d): %s", base, tosfos, t.Format("3:04 PM"))
	}
	return fmt.Sprintf("%s: %s", base, t.Format("3:04 PM"))
}

// titleBase strips the time suffix from a Hebcal event title.
// "Candle lighting: 6:36pm" → "Candle lighting"
// "Havdalah: 8:03pm"       → "Havdalah"
// "Havdalah" (no colon)    → "Havdalah"
func titleBase(s string) string {
	if idx := strings.Index(s, ": "); idx >= 0 {
		return s[:idx]
	}
	return s
}

func midnight(t time.Time, tz *time.Location) time.Time {
	local := t.In(tz)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, tz)
}
