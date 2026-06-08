/*
 * SPDX-FileCopyrightText: Copyright 2026 Todd Schulman
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package attacher

import (
	"testing"
	"time"
)

func TestZmanimRange(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	am6_18 := time.Date(2026, 3, 20, 6, 18, 0, 0, loc)
	am10_07 := time.Date(2026, 3, 20, 10, 7, 0, 0, loc)
	pm7_30 := time.Date(2026, 3, 20, 19, 30, 0, 0, loc)
	pm8_00 := time.Date(2026, 3, 20, 20, 0, 0, 0, loc)
	zero := time.Time{}

	tests := []struct {
		label string
		start time.Time
		end   time.Time
		want  string
	}{
		// Both AM — suffix collapsed
		{"Shma", am6_18, am10_07, "Shma: 6:18–10:07 AM"},
		// Both PM — suffix collapsed
		{"Window", pm7_30, pm8_00, "Window: 7:30–8:00 PM"},
		// Mixed AM/PM — both shown
		{"Mixed", am10_07, pm7_30, "Mixed: 10:07 AM–7:30 PM"},
		// Zero start — falls back to single value
		{"Label", zero, am10_07, "Label: 10:07 AM"},
		// Zero end — falls back to single value
		{"Label", am6_18, zero, "Label: 6:18 AM"},
		// Both zero — empty
		{"Label", zero, zero, ""},
	}
	for _, tt := range tests {
		got := zmanimRange(tt.label, tt.start, tt.end, loc)
		if got != tt.want {
			t.Errorf("zmanimRange(%q, %v, %v)\n  got  %q\n  want %q",
				tt.label, tt.start, tt.end, got, tt.want)
		}
	}
}

func TestFmtTime(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	tests := []struct {
		t    time.Time
		want string
	}{
		{time.Date(2026, 3, 20, 6, 52, 0, 0, loc), "6:52 AM"},
		{time.Date(2026, 3, 20, 19, 17, 0, 0, loc), "7:17 PM"},
		{time.Date(2026, 3, 20, 12, 0, 0, 0, loc), "12:00 PM"},
		{time.Time{}, ""},
	}
	for _, tt := range tests {
		got := fmtTime(tt.t, loc)
		if got != tt.want {
			t.Errorf("fmtTime(%v) = %q, want %q", tt.t, got, tt.want)
		}
	}
}
