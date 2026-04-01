// SPDX-FileCopyrightText: 2026 toobuntu
// SPDX-License-Identifier: GPL-3.0-or-later

package patcher

import (
	"testing"
	"time"

	"github.com/toobuntu/zman-didan/internal/chabad"
	"github.com/toobuntu/zman-didan/internal/types"
)

func TestRebuildCandle(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	t652 := time.Date(2026, 3, 20, 18, 52, 0, 0, loc)
	t705 := time.Date(2026, 4, 1, 19, 5, 0, 0, loc)
	t812 := time.Date(2026, 4, 2, 20, 12, 0, 0, loc)

	tests := []struct {
		hebrew       string
		t            time.Time
		isYomTov     bool
		afterHavdala bool
		wantTitle    string
		wantHebrew   string
	}{
		{"הדלקת נרות", t652, false, false, "Candle lighting: 6:52 PM", "הדלקת נרות"},
		{"הדלקת נרות", t705, true, false, "YT candles: 7:05 PM", "הדלקת נרות"},
		{"הדלקת נרות", t812, true, true, "YT candles after: 8:12 PM", "הדלקת נרות אחר"},
	}
	for _, tt := range tests {
		gotTitle, gotHebrew := rebuildCandle(tt.hebrew, tt.t, tt.isYomTov, tt.afterHavdala)
		if gotTitle != tt.wantTitle {
			t.Errorf("rebuildCandle title: got %q, want %q", gotTitle, tt.wantTitle)
		}
		if gotHebrew != tt.wantHebrew {
			t.Errorf("rebuildCandle hebrew: got %q, want %q", gotHebrew, tt.wantHebrew)
		}
	}
}

func TestRebuildHavdalaTitle(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	t803 := time.Date(2026, 3, 21, 20, 3, 0, 0, loc)

	tests := []struct {
		t      time.Time
		tosfos int
		want   string
	}{
		{t803, 4, "Havdalah (+4): 8:03 PM"},
		{t803, 0, "Havdalah: 8:03 PM"},
	}
	for _, tt := range tests {
		got := rebuildHavdalaTitle(tt.t, tt.tosfos)
		if got != tt.want {
			t.Errorf("rebuildHavdalaTitle(%d) = %q, want %q", tt.tosfos, got, tt.want)
		}
	}
}

func TestPatchCandles_StringKeyLookup(t *testing.T) {
	// Verifies that PatchCandles correctly matches events to candleTimes
	// entries. This is the regression test for the time.Time map key bug
	// where different *Location pointers caused silent lookup failures.
	loc, _ := time.LoadLocation("America/New_York")
	candleTime := time.Date(2026, 3, 20, 18, 52, 0, 0, loc)

	events := []types.HebcalEvent{
		{
			Date:     time.Date(2026, 3, 20, 18, 36, 0, 0, loc),
			AllDay:   false,
			Category: "candles",
			Title:    "Candle lighting: 6:36pm",
			Hebrew:   "הדלקת נרות",
		},
	}
	candleTimes := map[string]chabad.CandleDay{
		"2026-03-20": {Candles: candleTime, IsYomTov: false},
	}

	PatchCandles(events, candleTimes, "America/New_York", 4)

	if events[0].Date != candleTime {
		t.Errorf("PatchCandles did not update Date: got %v, want %v",
			events[0].Date, candleTime)
	}
	if events[0].Title != "Candle lighting: 6:52 PM" {
		t.Errorf("PatchCandles title: got %q, want %q",
			events[0].Title, "Candle lighting: 6:52 PM")
	}
}
