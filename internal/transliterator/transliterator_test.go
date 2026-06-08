package transliterator

import (
	"testing"

	"github.com/toobuntu/zman-didan/internal/types"
)

func applyToOne(title, desc, memo string, stripNikud bool) types.HebcalEvent {
	evs := []types.HebcalEvent{{Title: title, Description: desc, Memo: memo}}
	Apply(evs, stripNikud)
	return evs[0] // Apply mutates the slice element, not the passed value
}

func TestApply_AllScope(t *testing.T) {
	// Holiday and parsha names are replaced in Title, Description, and Memo.
	tests := []struct {
		field string
		in    string
		want  string
	}{
		{"title", "Shabbat Shalom", "Shabbos Shalom"},
		{"desc", "This Shabbat we read", "This Shabbos we read"},
		{"memo", "Erev Shabbat", "Erev Shabbos"},
		{"title", "Shavuot", "Shavuos"},
		{"title", "Sukkot", "Succos"},
	}
	for _, tt := range tests {
		var out types.HebcalEvent
		switch tt.field {
		case "title":
			out = applyToOne(tt.in, "", "", false)
		case "desc":
			out = applyToOne("", tt.in, "", false)
		case "memo":
			out = applyToOne("", "", tt.in, false)
		}
		var got string
		switch tt.field {
		case "title":
			got = out.Title
		case "desc":
			got = out.Description
		case "memo":
			got = out.Memo
		}
		if got != tt.want {
			t.Errorf("Apply %s(%q) = %q, want %q", tt.field, tt.in, got, tt.want)
		}
	}
}

func TestApply_TitleOnlyScope(t *testing.T) {
	// Biblical book names: replaced in Title and Memo, NOT in Description.
	events := []types.HebcalEvent{
		{
			Title:       "Parshas Shemos",                     // book name in title → replaced
			Description: "commemorates the Exodus from Egypt", // prose → NOT replaced
			Memo:        "Reading: Exodus 1:1",                // book name in memo → replaced
		},
	}
	Apply(events, false)
	ev := events[0]

	if ev.Title != "Parshas Shemos" {
		// "Shemos" is already Ashkenazi; the source is "Exodus" → "Shemos"
		// and "Shemot" → "Shemos". "Shemos" itself is not a source.
		// So Title should be unchanged here.
		t.Errorf("unexpected Title change: %q", ev.Title)
	}

	// The key test: "Exodus" in Description must NOT be replaced.
	if ev.Description != "commemorates the Exodus from Egypt" {
		t.Errorf("Description was modified (Exodus scope leak): %q", ev.Description)
	}

	// "Exodus" in Memo SHOULD be replaced.
	if ev.Memo != "Reading: Shemos 1:1" {
		t.Errorf("Memo not transliterated: got %q, want %q", ev.Memo, "Reading: Shemos 1:1")
	}
}

func TestApply_TitleOnly_BookNames(t *testing.T) {
	// Verify several book name substitutions hit Title but not Description.
	books := [][2]string{
		{"Genesis", "Bereishis"},
		{"Leviticus", "Vayikra"},
		{"Numbers", "Bamidbar"},
		{"Deuteronomy", "Devarim"},
		{"Isaiah", "Yeshayahu"},
		{"Jeremiah", "Yirmiyahu"},
		{"Ezekiel", "Yechezkel"},
	}
	for _, b := range books {
		src, dst := b[0], b[1]
		events := []types.HebcalEvent{
			{
				Title:       "Reading from " + src,
				Description: "The book of " + src + " tells us",
			},
		}
		Apply(events, false)
		ev := events[0]

		if ev.Title != "Reading from "+dst {
			t.Errorf("book %q: Title not replaced: got %q", src, ev.Title)
		}
		if ev.Description != "The book of "+src+" tells us" {
			t.Errorf("book %q: Description was modified: got %q", src, ev.Description)
		}
	}
}

func TestApply_HavdalahTransliteration(t *testing.T) {
	events := []types.HebcalEvent{{Title: "Havdalah: 8:03 PM"}}
	Apply(events, false)
	// "Havdalah" → "Havdala" is in the all-scope table
	if events[0].Title != "Havdala: 8:03 PM" {
		t.Errorf("Havdalah not transliterated: got %q", events[0].Title)
	}
}

func TestApply_StripNikud(t *testing.T) {
	hebrew := "הַבְדָּלָה" // with nikud
	stripped := "הבדלה"    // without nikud

	events := []types.HebcalEvent{{Hebrew: hebrew, Title: "x"}}
	Apply(events, true)
	if events[0].Hebrew != stripped {
		t.Errorf("nikud not stripped: got %q, want %q", events[0].Hebrew, stripped)
	}
}

func TestApply_StripNikudFalse(t *testing.T) {
	hebrew := "הַבְדָּלָה"
	events := []types.HebcalEvent{{Hebrew: hebrew}}
	Apply(events, false)
	if events[0].Hebrew != hebrew {
		t.Errorf("Hebrew was modified when stripNikud=false: got %q", events[0].Hebrew)
	}
}

func TestApply_LongestMatchFirst(t *testing.T) {
	// "Shemini Atzeret" must map to "Shemini Atzeres", not "Shemini Atzer..."
	// This verifies longest-source-first ordering in the all-scope table.
	events := []types.HebcalEvent{{Title: "Shemini Atzeret"}}
	Apply(events, false)
	if events[0].Title != "Shemini Atzeres" {
		t.Errorf("got %q, want %q", events[0].Title, "Shemini Atzeres")
	}
}
