// SPDX-FileCopyrightText: 2026 toobuntu
// SPDX-License-Identifier: GPL-3.0-or-later

package fastday

import (
	"testing"
	"time"

	"github.com/toobuntu/zman-didan/internal/types"
)

func TestFastBaseName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Tzom Gedaliah", "Tzom Gedaliah"},
		{"Ta'anis Bechoros", "Ta'anis Bechoros"},
		// Hebrew after "/" is stripped
		{"Tzom Gedaliah / צוֹם גְּדַלְיָה", "Tzom Gedaliah"},
		{"Ta'anis Bechoros / תענית בכורות", "Ta'anis Bechoros"},
		// Leading emoji is stripped
		{"✡️ Tzom Gedaliah / צוֹם גְּדַלְיָה", "Tzom Gedaliah"},
		// Tisha B'Av
		{"Tisha B'Av / תִּשְׁעָה בְּאָב", "Tisha B'Av"},
	}
	for _, tt := range tests {
		got := fastBaseName(tt.in)
		if got != tt.want {
			t.Errorf("fastBaseName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsEmoji(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"✡️", true},
		{"🕯️", true},
		{"A", false},
		{"", false},
		{"Tzom", false},
	}
	for _, tt := range tests {
		got := isEmoji(tt.s)
		if got != tt.want {
			t.Errorf("isEmoji(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestFastTimes_RegularFast(t *testing.T) {
	// Regular fast (not Tisha B'Av or Yom Kippur): begin = alos, end = tzeis.
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)

	z := types.ZmanimDay{
		Date:  date,
		Alos:  time.Date(2026, 4, 1, 5, 22, 0, 0, loc),
		Tzeis: time.Date(2026, 4, 1, 19, 58, 0, 0, loc),
	}
	ev := types.HebcalEvent{Title: "Ta'anis Bechoros", Subcat: "fast"}

	begin, end := fastTimes(ev, z, types.ZmanimDay{}, false)

	if !begin.Equal(z.Alos) {
		t.Errorf("begin: got %v, want %v (alos)", begin, z.Alos)
	}
	if !end.Equal(z.Tzeis) {
		t.Errorf("end: got %v, want %v (tzeis)", end, z.Tzeis)
	}
}

func TestFastTimes_TishaBav(t *testing.T) {
	// Tisha B'Av: begin = previous day's shkiah.
	loc, _ := time.LoadLocation("America/New_York")
	fastDate := time.Date(2026, 7, 23, 0, 0, 0, 0, loc) // Tisha B'Av 5786

	z := types.ZmanimDay{
		Date:  fastDate,
		Alos:  time.Date(2026, 7, 23, 5, 10, 0, 0, loc),
		Tzeis: time.Date(2026, 7, 23, 20, 30, 0, 0, loc),
	}
	prevZ := types.ZmanimDay{
		Date:   time.Date(2026, 7, 22, 0, 0, 0, 0, loc),
		Shkiah: time.Date(2026, 7, 22, 20, 15, 0, 0, loc),
	}
	ev := types.HebcalEvent{Title: "Tisha B'Av", Subcat: "fast"}

	begin, end := fastTimes(ev, z, prevZ, true)

	if !begin.Equal(prevZ.Shkiah) {
		t.Errorf("Tisha B'Av begin: got %v, want prev shkiah %v", begin, prevZ.Shkiah)
	}
	if !end.Equal(z.Tzeis) {
		t.Errorf("end: got %v, want %v", end, z.Tzeis)
	}
}

func TestFastTimes_TishaBavNoPrevZ(t *testing.T) {
	// If previous day's zmanim aren't available, fall back to alos.
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2026, 7, 23, 0, 0, 0, 0, loc)
	z := types.ZmanimDay{
		Date: date,
		Alos: time.Date(2026, 7, 23, 5, 10, 0, 0, loc),
	}
	ev := types.HebcalEvent{Title: "Tisha B'Av", Subcat: "fast"}

	begin, _ := fastTimes(ev, z, types.ZmanimDay{}, false) // hasPrev=false

	if !begin.Equal(z.Alos) {
		t.Errorf("Tisha B'Av without prev: got %v, want alos %v", begin, z.Alos)
	}
}

func TestBuild_SkipsTimedEvents(t *testing.T) {
	// Build only processes AllDay=true events with Subcat="fast".
	// A timed fast event (AllDay=false) should be ignored.
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)

	events := []types.HebcalEvent{
		{Date: date, AllDay: false, Subcat: "fast", Title: "Fast begins"},
		{Date: date, AllDay: true, Subcat: "fast", Title: "Ta'anis Bechoros"},
	}
	zmanimMap := map[string]types.ZmanimDay{
		"2026-04-01": {
			Date:  date,
			Alos:  time.Date(2026, 4, 1, 5, 22, 0, 0, loc),
			Tzeis: time.Date(2026, 4, 1, 19, 58, 0, 0, loc),
		},
	}

	result := Build(events, zmanimMap, "17601", "America/New_York")

	// Should produce exactly 2 synthesised events (begin + end) from the
	// all-day event only.
	if len(result) != 2 {
		t.Fatalf("expected 2 synthesised events, got %d", len(result))
	}

	categories := map[string]bool{}
	for _, ev := range result {
		categories[ev.Category] = true
	}
	if !categories["fast-begin"] || !categories["fast-end"] {
		t.Errorf("expected fast-begin and fast-end categories, got %v", categories)
	}
}

func TestBuild_EventFields(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)
	alos := time.Date(2026, 4, 1, 5, 22, 0, 0, loc)
	tzeis := time.Date(2026, 4, 1, 19, 58, 0, 0, loc)

	events := []types.HebcalEvent{
		{Date: date, AllDay: true, Subcat: "fast", Title: "Ta'anis Bechoros"},
	}
	zmanimMap := map[string]types.ZmanimDay{
		"2026-04-01": {Date: date, Alos: alos, Tzeis: tzeis},
	}

	result := Build(events, zmanimMap, "17601", "America/New_York")
	if len(result) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result))
	}

	var begin, end types.HebcalEvent
	for _, ev := range result {
		switch ev.Category {
		case "fast-begin":
			begin = ev
		case "fast-end":
			end = ev
		}
	}

	if begin.Title != "Fast begins" {
		t.Errorf("begin title: %q", begin.Title)
	}
	if begin.Hebrew != "תחילת הצום" {
		t.Errorf("begin Hebrew: %q", begin.Hebrew)
	}
	if begin.Description != "Ta'anis Bechoros" {
		t.Errorf("begin description: %q", begin.Description)
	}
	if !begin.Date.Equal(alos) {
		t.Errorf("begin time: got %v, want %v", begin.Date, alos)
	}
	if len(begin.Alarms) != 2 {
		t.Errorf("begin alarms: got %d, want 2", len(begin.Alarms))
	}

	if end.Title != "Fast ends" {
		t.Errorf("end title: %q", end.Title)
	}
	if end.Hebrew != "סיום הצום" {
		t.Errorf("end Hebrew: %q", end.Hebrew)
	}
	if !end.Date.Equal(tzeis) {
		t.Errorf("end time: got %v, want %v", end.Date, tzeis)
	}
	if len(end.Alarms) != 2 {
		t.Errorf("end alarms: got %d, want 2", len(end.Alarms))
	}
}
