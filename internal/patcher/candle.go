// Package patcher replaces hebcal's placeholder candle lighting and havdalah
// times with Chabad-authoritative times from the chabad.org ICS feed.
package patcher

import (
	"time"

	"github.com/toobuntu/zman-didan/internal/chabad"
	"github.com/toobuntu/zman-didan/internal/types"
)

// PatchCandles replaces DTSTART on candle lighting and havdalah events with
// the corresponding times from candleTimes, keyed by local midnight.
func PatchCandles(events []types.HebcalEvent, candleTimes map[time.Time]chabad.CandleDay, tzid string) {
	tz, err := time.LoadLocation(tzid)
	if err != nil {
		tz = time.UTC
	}
	for i := range events {
		ev := &events[i]
		switch ev.Category {
		case "candles":
			key := midnight(ev.Date, tz)
			if day, ok := candleTimes[key]; ok && !day.Candles.IsZero() {
				ev.Date = day.Candles
				ev.AllDay = false
			}
		case "havdalah":
			key := midnight(ev.Date, tz)
			if day, ok := candleTimes[key]; ok && !day.Havdalah.IsZero() {
				ev.Date = day.Havdalah
				ev.AllDay = false
			}
		}
	}
}

func midnight(t time.Time, tz *time.Location) time.Time {
	local := t.In(tz)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, tz)
}
