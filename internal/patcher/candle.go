// Package patcher replaces Hebcal's placeholder candle lighting and havdalah
// times with Chabad-authoritative times from the chabad.org ICS feed.
//
// Tosfos Shabbos: havdalah is advanced by cfg.Tosfos minutes beyond chabad.org
// tzeis, fulfilling the halachic requirement to add mundane time to Shabbos at
// its end. The default offset is 4 minutes. The event Title is rebuilt to show
// both the corrected time and the offset: "Havdalah (+4): 8:07 PM".
// The transliterator converts "Havdalah" → "Havdala" downstream, so the base
// word is left in Hebcal form here.
//
// Map key: candleTimes is keyed by "YYYY-MM-DD" in the location's local
// timezone, matching how parseCandleICS builds the map. Using string keys
// avoids Go's time.Time equality semantics, which include the *Location pointer
// and would silently miss lookups when different LoadLocation calls return
// different pointers for the same TZID.
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
func PatchCandles(events []types.HebcalEvent, candleTimes map[string]chabad.CandleDay, tzid string, tosfos int) {
	tz, err := time.LoadLocation(tzid)
	if err != nil {
		tz = time.UTC
	}
	for i := range events {
		ev := &events[i]
		key := ev.Date.In(tz).Format("2006-01-02")
		switch ev.Category {
		case "candles":
			if day, ok := candleTimes[key]; ok && !day.Candles.IsZero() {
				ev.Date = day.Candles
				ev.AllDay = false
				ev.Title = rebuildCandleTitle(ev.Title, day.Candles.In(tz), day.AfterHavdala)
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

// rebuildCandleTitle reconstructs the candle lighting SUMMARY.
// afterHavdala is true for second-night Yom Tov candles (lit after Havdalah).
// "Candle lighting: 6:36pm" → "Candle lighting: 6:52 PM"
// "Candle lighting: 6:36pm" (after) → "Candle lighting after Havdala: 8:02 PM"
func rebuildCandleTitle(original string, t time.Time, afterHavdala bool) string {
	base := titleBase(original)
	if afterHavdala {
		return fmt.Sprintf("%s after Havdala: %s", base, t.Format("3:04 PM"))
	}
	return fmt.Sprintf("%s: %s", base, t.Format("3:04 PM"))
}

// rebuildHavdalaTitle reconstructs the havdalah SUMMARY with the patched time
// and tosfos offset.
// "Havdalah: 8:03pm", tosfos=4 → "Havdalah (+4): 8:07 PM"
// "Havdalah: 8:03pm", tosfos=0 → "Havdalah: 8:03 PM"
func rebuildHavdalaTitle(original string, t time.Time, tosfos int) string {
	base := titleBase(original)
	if tosfos > 0 {
		return fmt.Sprintf("%s (+%d): %s", base, tosfos, t.Format("3:04 PM"))
	}
	return fmt.Sprintf("%s: %s", base, t.Format("3:04 PM"))
}

// titleBase strips the time suffix (and trailing modifiers) from a Hebcal title.
// "Candle lighting: 6:36pm" → "Candle lighting"
// "Havdalah: 8:03pm"       → "Havdalah"
func titleBase(s string) string {
	if idx := strings.Index(s, ": "); idx >= 0 {
		return s[:idx]
	}
	return s
}
