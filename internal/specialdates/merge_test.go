package specialdates

import (
	"testing"
)

func TestShortenTitle(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		// Yahrzeits — feed uses U+2019 RIGHT SINGLE QUOTATION MARK
		{"the Rebbe Rashab\u2019s 106th Yahrzeit (2nd of Nisan)", "Rebbe Rashab's Histalkus"},
		{"the Rebbe\u2019s 32nd Yahrzeit (3rd of Tammuz)", "Rebbe's Histalkus"},
		// Without "the" prefix
		{"Tzemach Tzeddek\u2019s 160th Yahrzeit (13th of Nisan)", "Tzemach Tzeddek's Histalkus"},
		// Straight apostrophe fallback
		{"the Alter Rebbe's 213th Yahrzeit (24th of Tevet)", "Alter Rebbe's Histalkus"},
		// Birthdays
		{"Birthday of the Rebbe (11th of Nisan)", "Rebbe's Birthday"},
		{"Birthday of the Rebbe Rashab (20th of Cheshvan)", "Rebbe Rashab's Birthday"},
		{"Birthday of Tzemach Tzeddek (29th of Elul)", "Tzemach Tzeddek's Birthday"},
		// Non-matching — passed through unchanged
		{"Rosh Hashanah of Chassidism (19th of Kislev)", "Rosh Hashanah of Chassidism (19th of Kislev)"},
		{"Marriage of Rebbe and Rebbetzin (14th of Kislev)", "Marriage of Rebbe and Rebbetzin (14th of Kislev)"},
	}
	for _, tt := range tests {
		got := shortenTitle(tt.in)
		if got != tt.want {
			t.Errorf("shortenTitle(%q)\n  got  %q\n  want %q", tt.in, got, tt.want)
		}
	}
}

func TestReformatDescription_Yahrzeit(t *testing.T) {
	// Rebbe Rashab: histalkus 5680, huledes 5621, age 59, DOD 1920-03-21
	raw := "Hebcal joins you in remembering R. Sholom DovBer of Lubavitch, whose 106th Yahrzeit occurs on Thursday, March 21, 2026, corresponding to the 2nd of Nisan, 5786"
	got := reformatDescription(raw)
	want := "Histalkus of R. Sholom DovBer of Lubavitch at age 59 on 2 Nisan 5680 (21 Mar 1920) — 106 years ago"
	if got != want {
		t.Errorf("reformatDescription (yahrzeit)\n  got  %q\n  want %q", got, want)
	}
}

func TestReformatDescription_Birthday(t *testing.T) {
	// The Rebbe: huledes 5662, DOB 1902-04-18
	raw := "Birthday of Lubavitcher Rebbe R. Menachem M. Schneerson occurs on Saturday, April 4, 2026, corresponding to the 11th of Nisan, 5786"
	got := reformatDescription(raw)
	want := "Birthday of Lubavitcher Rebbe R. Menachem M. Schneerson on 11 Nisan 5662 (18 Apr 1902) — 124 years ago"
	if got != want {
		t.Errorf("reformatDescription (birthday)\n  got  %q\n  want %q", got, want)
	}
}

func TestReformatDescription_NonBirthdayEvent(t *testing.T) {
	// birthdayRe matches "X occurs on..." for all events, not just birthdays.
	// Non-birthday events must not be prefixed with "Birthday of".
	tests := []struct {
		raw  string
		want string
	}{
		{
			"Rosh Hashanah of Chassidism occurs on Tuesday, December 9, 2025, corresponding to the 19th of Kislev, 5786",
			"Rosh Hashanah of Chassidism on 19 Kislev 5786",
		},
		{
			"Marriage of Rebbe and Rebbetzin occurs on Thursday, December 4, 2025, corresponding to the 14th of Kislev, 5786",
			"Marriage of Rebbe and Rebbetzin on 14 Kislev 5786",
		},
	}
	for _, tt := range tests {
		got := reformatDescription(tt.raw)
		if got != tt.want {
			t.Errorf("reformatDescription (non-birthday)\n  got  %q\n  want %q", got, tt.want)
		}
	}
}

func TestFormatGregorian(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"1920-03-21", "21 Mar 1920"},
		{"1902-04-18", "18 Apr 1902"},
		{"1994-06-12", "12 Jun 1994"},
		{"bad-date", "bad-date"},
	}
	for _, tt := range tests {
		got := formatGregorian(tt.in)
		if got != tt.want {
			t.Errorf("formatGregorian(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
