/*
 * SPDX-FileCopyrightText: Copyright 2026 Todd Schulman
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package icalwriter

import (
	"testing"

	"github.com/toobuntu/zman-didan/internal/types"
)

func TestBuildSummary(t *testing.T) {
	tests := []struct {
		ev     types.HebcalEvent
		lang   string
		emojis bool
		want   string
	}{
		// Candle lighting — always bilingual regardless of lang
		{
			ev:   types.HebcalEvent{Category: "candles", Title: "Candle lighting: 6:52 PM", Hebrew: "הדלקת נרות"},
			lang: "h", emojis: false,
			want: "Candle lighting: 6:52 PM / הדלקת נרות",
		},
		// Havdala — always bilingual
		{
			ev:   types.HebcalEvent{Category: "havdalah", Title: "Havdalah (+4): 8:03 PM", Hebrew: "הַבְדָּלָה"},
			lang: "h", emojis: false,
			want: "Havdalah (+4): 8:03 PM / הַבְדָּלָה",
		},
		// he mode — non-timed event: Title only (Hebrew from API already)
		{
			ev:   types.HebcalEvent{Category: "holiday", Title: "Shabbos", Hebrew: "שַׁבָּת"},
			lang: "h", emojis: false,
			want: "Shabbos",
		},
		// ah mode — non-timed event: bilingual
		{
			ev:   types.HebcalEvent{Category: "holiday", Title: "Shabbos", Hebrew: "שַׁבָּת"},
			lang: "ah", emojis: false,
			want: "Shabbos / שַׁבָּת",
		},
		// Emoji prefix
		{
			ev:   types.HebcalEvent{Category: "candles", Title: "Candle lighting: 6:52 PM", Hebrew: "הדלקת נרות"},
			lang: "ah", emojis: true,
			want: "🕯️ Candle lighting: 6:52 PM / הדלקת נרות",
		},
		// Hebrew == Title: no bilingual suffix added
		{
			ev:   types.HebcalEvent{Category: "holiday", Title: "Shabbos", Hebrew: "Shabbos"},
			lang: "ah", emojis: false,
			want: "Shabbos",
		},
	}
	for _, tt := range tests {
		got := buildSummary(tt.ev, tt.lang, tt.emojis)
		if got != tt.want {
			t.Errorf("buildSummary(%q, %q, %v)\n  got  %q\n  want %q",
				tt.ev.Title, tt.lang, tt.emojis, got, tt.want)
		}
	}
}

func TestTriggerDuration(t *testing.T) {
	tests := []struct {
		minutes int
		want    string
	}{
		{0, "PT0S"},
		{-15, "-PT15M"},
		{-120, "-PT2H"},
		{-90, "-PT1H30M"},
		{-10, "-PT10M"},
		{-30, "-PT30M"},
	}
	for _, tt := range tests {
		got := triggerDuration(tt.minutes)
		if got != tt.want {
			t.Errorf("triggerDuration(%d) = %q, want %q", tt.minutes, got, tt.want)
		}
	}
}

func TestEscapeText(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`back\slash`, `back\\slash`},
		{"semi;colon", `semi\;colon`},
		{"com,ma", `com\,ma`},
		{"line\nnewline", `line\nnewline`},
		{"Lancaster, PA 17601", `Lancaster\, PA 17601`},
	}
	for _, tt := range tests {
		got := escapeText(tt.in)
		if got != tt.want {
			t.Errorf("escapeText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
