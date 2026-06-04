package chabad

import (
	"testing"
	"time"
)

func TestParseCandleICS_KeyTypes(t *testing.T) {
	// Verifies that the candle map uses string keys ("YYYY-MM-DD"), not
	// time.Time keys. time.Time equality includes the *Location pointer;
	// different LoadLocation calls for the same TZID may return different
	// pointers, causing silent cache misses if time.Time were used as key.
	//
	// Construct a minimal ICS with one candle lighting event and one
	// Shabbat Ends event, then confirm the map is keyed by date string.
	const ics = "BEGIN:VCALENDAR\r\n" +
		"VERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20260320T225200Z\r\n" + // 6:52 PM ET (UTC-4)
		"SUMMARY:Light Shabbat Candles at 6:52 PM\r\n" +
		"END:VEVENT\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20260321T230300Z\r\n" + // 7:03 PM ET
		"SUMMARY:Shabbat Ends\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	result, err := parseCandleICS([]byte(ics), "America/New_York")
	if err != nil {
		t.Fatalf("parseCandleICS error: %v", err)
	}

	day, ok := result["2026-03-20"]
	if !ok {
		t.Fatalf("expected key %q in result, got keys: %v", "2026-03-20", mapKeys(result))
	}
	if day.Candles.IsZero() {
		t.Error("Candles time is zero")
	}
	if day.IsYomTov {
		t.Error("IsYomTov should be false for Light Shabbat Candles")
	}

	havdala, ok := result["2026-03-21"]
	if !ok {
		t.Fatalf("expected key %q in result", "2026-03-21")
	}
	if havdala.Havdalah.IsZero() {
		t.Error("Havdalah time is zero")
	}
}

func TestParseCandleICS_YomTovAfter(t *testing.T) {
	const ics = "BEGIN:VCALENDAR\r\n" +
		"VERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20260403T000200Z\r\n" + // 8:02 PM ET (UTC-4)
		"SUMMARY:Light Holiday Candles after Shabbat\r\n" +
		"END:VEVENT\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20260401T230500Z\r\n" +
		"SUMMARY:Light Holiday Candles at 7:05 PM\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	result, err := parseCandleICS([]byte(ics), "America/New_York")
	if err != nil {
		t.Fatalf("parseCandleICS error: %v", err)
	}

	// First-night YT candles (not after)
	erevPesach, ok := result["2026-04-01"]
	if !ok {
		t.Fatal("expected 2026-04-01 in result")
	}
	if !erevPesach.IsYomTov {
		t.Error("expected IsYomTov=true for Light Holiday Candles")
	}
	if erevPesach.AfterHavdala {
		t.Error("expected AfterHavdala=false for first-night candles")
	}

	// Second-night YT candles (after Havdalah)
	pesachI, ok := result["2026-04-02"]
	if !ok {
		t.Fatal("expected 2026-04-02 in result")
	}
	if !pesachI.IsYomTov {
		t.Error("expected IsYomTov=true for Light Holiday Candles after")
	}
	if !pesachI.AfterHavdala {
		t.Error("expected AfterHavdala=true for Light Holiday Candles after")
	}
}

func TestNormalizeLabel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		// Missing delimiter after closing paren
		{"Sunset (Shkiah)Fast Begins", "Sunset (Shkiah) | Fast Begins"},
		// Already correctly delimited
		{"Candle Lighting | Fast Begins", "Candle Lighting | Fast Begins"},
		// No issue
		{"Alot Hashachar", "Alot Hashachar"},
	}
	for _, tt := range tests {
		got := normalizeLabel(tt.in)
		if got != tt.want {
			t.Errorf("normalizeLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestClassifyLabel(t *testing.T) {
	tests := []struct {
		label   string
		field   string
		isEvent bool
	}{
		{"Alot Hashachar", "alos", false},
		{"Earliest Tallit and Tefillin (Misheyakir)", "misheyakir", false},
		{"Earliest Tallit (Misheyakir)", "misheyakir", false},
		{"Sunset (Shkiah)", "shkiah", false},
		{"Tzeit Hakochavim", "tzeis", false},
		{"Shabbat Ends", "tzeis", true},
		{"Holiday/Fast Ends", "tzeis", true},
		{"Chatzot HaLailah", "chatzos_halaila", false},
		{"Candle Lighting", "", true},
		{"Fast Begins", "", true},
		{"Finish Eating Chametz before 10:38 AM", "", true},
		{"SomethingUnknown", "", true},
	}
	for _, tt := range tests {
		field, isEvent := classifyLabel(tt.label)
		if field != tt.field || isEvent != tt.isEvent {
			t.Errorf("classifyLabel(%q)\n  got  field=%q isEvent=%v\n  want field=%q isEvent=%v",
				tt.label, field, isEvent, tt.field, tt.isEvent)
		}
	}
}

func TestParseLocalTime_ChatzosHalaila(t *testing.T) {
	// Chatzos HaLailah falls after midnight on the next calendar day.
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, loc) // seder night

	got, err := parseLocalTime("1:08 AM", date, true, loc)
	if err != nil {
		t.Fatalf("parseLocalTime error: %v", err)
	}
	if got.Day() != 2 {
		t.Errorf("expected chatzos halaila to fall on day 2, got day %d", got.Day())
	}
	if got.Hour() != 1 || got.Minute() != 8 {
		t.Errorf("expected 1:08, got %02d:%02d", got.Hour(), got.Minute())
	}

	// Non-chatzos: same date even if hour < 12
	got2, err := parseLocalTime("6:18 AM", date, false, loc)
	if err != nil {
		t.Fatalf("parseLocalTime error: %v", err)
	}
	if got2.Day() != 1 {
		t.Errorf("expected alos to stay on day 1, got day %d", got2.Day())
	}
}

func mapKeys(m map[string]CandleDay) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Ensure time.Time zero works as expected in map lookups — documents the
// reason we use string keys instead of time.Time keys.
func TestTimeMapKeyBehavior(t *testing.T) {
	loc1, _ := time.LoadLocation("America/New_York")
	loc2, _ := time.LoadLocation("America/New_York")

	t1 := time.Date(2026, 3, 20, 0, 0, 0, 0, loc1)
	t2 := time.Date(2026, 3, 20, 0, 0, 0, 0, loc2)

	// With time.Time keys: equality depends on *Location pointer.
	// In practice loc1 == loc2 because LoadLocation caches internally,
	// but that is an implementation detail we must not rely on.
	// String keys are unambiguous.
	if t1 != t2 {
		t.Log("Note: time.Time keys differ for separately loaded locations (pointer inequality)")
	}

	m := map[string]bool{t1.Format("2006-01-02"): true}
	if !m[t2.Format("2006-01-02")] {
		t.Error("string key lookup failed — should always succeed")
	}
}
