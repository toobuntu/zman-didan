// Package patcher replaces Hebcal's placeholder candle lighting and havdalah
// times with Chabad-authoritative times from the chabad.org ICS feed.
//
// Tosfos Shabbos: havdalah is advanced by cfg.Tosfos minutes beyond the
// chabad.org tzeis time. The event title is rebuilt to show both the corrected
// time and the offset: "Havdalah (+4): 8:07 PM". The transliterator converts
// "Havdalah" → "Havdala" downstream, so the base word is left in Hebcal form.
//
// Map key: candleTimes uses "YYYY-MM-DD" string keys (not time.Time) to avoid
// Go's time.Time equality semantics, which include the *Location pointer.
// Multiple time.LoadLocation calls for the same TZID may return different
// pointers, causing map lookups to fail silently if time.Time were used as key.
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
				ev.Title, ev.Hebrew = rebuildCandle(ev.Hebrew, day.Candles.In(tz), day.IsYomTov, day.AfterHavdala)
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

// rebuildCandle returns the new Title and Hebrew for a candle lighting event.
//
// Title forms by case:
//
//	Regular Shabbos:      "Candle lighting: 6:52 PM"    (familiar term, no prefix)
//	Yom Tov:              "YT candles: 7:05 PM"
//	YT after Havdala:     "YT candles after: 8:12 PM"
//
// The "after" case ("Light Holiday Candles after" from chabad.org) indicates
// second-night Yom Tov candles lit after Havdalah. The Hebrew suffix "אחר"
// (after) conveys the same distinction in the bilingual SUMMARY.
//
// "Sh candles" is not used for regular Shabbos — "Candle lighting" is the
// universally understood term and avoids abbreviation where none is needed.
// The "YT" prefix is reserved for Yom Tov cases where the distinction from
// Shabbos candles is actually informative.
func rebuildCandle(hebrew string, t time.Time, isYomTov, afterHavdala bool) (title, newHebrew string) {
	timeStr := t.Format("3:04 PM")
	switch {
	case afterHavdala:
		title = "YT candles after: " + timeStr
		newHebrew = strings.TrimSpace(hebrew) + " אחר"
	case isYomTov:
		title = "YT candles: " + timeStr
		newHebrew = hebrew
	default:
		title = "Candle lighting: " + timeStr
		newHebrew = hebrew
	}
	return title, newHebrew
}

// rebuildHavdalaTitle reconstructs the havdalah SUMMARY.
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
func titleBase(s string) string {
	if idx := strings.Index(s, ": "); idx >= 0 {
		return s[:idx]
	}
	return s
}
