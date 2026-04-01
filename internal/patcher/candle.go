// SPDX-FileCopyrightText: 2026 toobuntu
// SPDX-License-Identifier: GPL-3.0-or-later

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
				ev.Title = rebuildHavdalaTitle(t.In(tz), tosfos)
			}
		}
	}
}

// rebuildCandle returns the new Title and Hebrew for a candle lighting event.
//
//	Regular Shabbos:   "Candle lighting: 6:52 PM"   /  "הדלקת נרות"
//	Yom Tov:           "YT candles: 7:05 PM"         /  "הדלקת נרות"
//	YT after Havdala:  "YT candles after: 8:12 PM"   /  "הדלקת נרות אחר"
//
// "YT" prefix is used only when the distinction from Shabbos candles is
// informative. "Candle lighting" is used for regular Shabbos — it is the
// universally familiar term and avoids unnecessary abbreviation.
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

// rebuildHavdalaTitle constructs the havdalah SUMMARY with the patched time
// and tosfos offset. Always uses the English base "Havdalah" regardless of
// the original title language — the time component is inherently LTR, and
// deriving the base from a Hebrew original title causes RTL display problems
// in calendar apps (the whole string reads reversed).
//
//	tosfos=4 → "Havdalah (+4): 8:07 PM"
//	tosfos=0 → "Havdalah: 8:03 PM"
//
// The transliterator converts "Havdalah" → "Havdala" downstream for
// Ashkenazi modes.
func rebuildHavdalaTitle(t time.Time, tosfos int) string {
	if tosfos > 0 {
		return fmt.Sprintf("Havdalah (+%d): %s", tosfos, t.Format("3:04 PM"))
	}
	return fmt.Sprintf("Havdalah: %s", t.Format("3:04 PM"))
}
