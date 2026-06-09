/*
 * SPDX-FileCopyrightText: Copyright 2026 Todd Schulman
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package haftorah

import (
	"testing"

	"github.com/toobuntu/zman-didan/internal/types"
)

// --- Source priority tests ---

func TestPatch_APIFieldPreferred(t *testing.T) {
	// When HaftarahChabad is set, it takes precedence over the embedded table.
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "tzav",
		Leyning: &types.Leyning{
			Haftarah:       "Jeremiah 7:21-8:3, 9:22-23",
			HaftarahChabad: "Jeremiah 7:21-28, 9:22-23",
		},
		Description: "Haftarah: Jeremiah 7:21-8:3, 9:22-23",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	want := "Jeremiah 7:21-28, 9:22-23"
	if events[0].Leyning.Haftarah != want {
		t.Errorf("Leyning.Haftarah = %q, want %q", events[0].Leyning.Haftarah, want)
	}
	if got := events[0].Description; got != "Haftorah: "+want {
		t.Errorf("Description = %q, want %q", got, "Haftorah: "+want)
	}
}

func TestPatch_FallbackToEmbeddedTable(t *testing.T) {
	// When HaftarahChabad is empty, fall back to the embedded JSON table.
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "bereshit",
		Leyning: &types.Leyning{
			Haftarah: "Isaiah 42:5-21",
		},
		Description: "Haftarah: Isaiah 42:5-21",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	want := "Isaiah 42:5-43:10"
	if events[0].Leyning.Haftarah != want {
		t.Errorf("Leyning.Haftarah = %q, want %q", events[0].Leyning.Haftarah, want)
	}
}

func TestPatch_NullChabadNoEmbedded_NoChange(t *testing.T) {
	// When HaftarahChabad is empty and slug has no embedded entry,
	// the standard haftarah should remain unchanged.
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "nonexistent-parsha",
		Leyning: &types.Leyning{
			Haftarah: "Some Book 1:1-10",
		},
		Description: "Haftarah: Some Book 1:1-10",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "Some Book 1:1-10" {
		t.Errorf("Haftarah changed unexpectedly: %q", events[0].Leyning.Haftarah)
	}
	if events[0].Description != "Haftarah: Some Book 1:1-10" {
		t.Errorf("Description changed unexpectedly: %q", events[0].Description)
	}
}

func TestPatch_SkipsNonParashat(t *testing.T) {
	events := []types.HebcalEvent{{
		Category:    "holiday",
		Slug:        "bereshit",
		Description: "Haftarah: Isaiah 42:5-21",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Description != "Haftarah: Isaiah 42:5-21" {
		t.Errorf("non-parashat event was modified: %q", events[0].Description)
	}
}

func TestPatch_SpecialShabbat(t *testing.T) {
	// Events with Subcat "special-shabbat" should also be patched.
	events := []types.HebcalEvent{{
		Category: "holiday",
		Subcat:   "special-shabbat",
		Slug:     "shabbat-zachor",
		Leyning: &types.Leyning{
			Haftarah: "I Samuel 15:1-34",
		},
		Description: "Haftarah: I Samuel 15:1-34",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	want := "I Samuel 15:2-34"
	if events[0].Leyning.Haftarah != want {
		t.Errorf("Leyning.Haftarah = %q, want %q", events[0].Leyning.Haftarah, want)
	}
}

// --- Description manipulation tests ---

func TestPatch_DescriptionReplace(t *testing.T) {
	events := []types.HebcalEvent{{
		Category:    "parashat",
		Slug:        "noach",
		Leyning:     &types.Leyning{Haftarah: "Isaiah 54:1-10"},
		Description: "Torah: Genesis 6:9-11:32\nHaftarah: Isaiah 54:1-10\nMaftir: special",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	want := "Torah: Genesis 6:9-11:32\nHaftorah: Isaiah 54:1-55:5\nMaftir: special"
	if events[0].Description != want {
		t.Errorf("Description = %q, want %q", events[0].Description, want)
	}
}

func TestPatch_DescriptionAppend(t *testing.T) {
	events := []types.HebcalEvent{{
		Category:    "parashat",
		Slug:        "noach",
		Leyning:     &types.Leyning{Haftarah: "Isaiah 54:1-10"},
		Description: "Some description without haftarah line",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	want := "Some description without haftarah line\nHaftorah: Isaiah 54:1-55:5"
	if events[0].Description != want {
		t.Errorf("Description = %q, want %q", events[0].Description, want)
	}
}

func TestPatch_EmptyDescription(t *testing.T) {
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "noach",
		Leyning:  &types.Leyning{Haftarah: "Isaiah 54:1-10"},
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	want := "Haftorah: Isaiah 54:1-55:5"
	if events[0].Description != want {
		t.Errorf("Description = %q, want %q", events[0].Description, want)
	}
}

func TestPatch_NilLeyning(t *testing.T) {
	// If Leyning is nil but slug matches embedded table, Description is still set.
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "noach",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	want := "Haftorah: Isaiah 54:1-55:5"
	if events[0].Description != want {
		t.Errorf("Description = %q, want %q", events[0].Description, want)
	}
}

// --- hebcal-leyning#732 edge cases ---
//
// The analysis in https://github.com/hebcal/hebcal-leyning/issues/732
// identifies three categories of Chabad haftorah divergence:
//
// Rule 1: When A(shkenazi) and S(ephardic) agree, AH (Chabad) reads shorter.
// Rule 2: When A and S differ, AH sometimes follows A, sometimes S, sometimes unique.

// Rule 1 cases: A and S agree but AH truncates.

func TestPatch_Rule1_Tzav_AHTruncates(t *testing.T) {
	// A & S: Jeremiah 7:21-8:3, 9:22-23
	// AH:    Jeremiah 7:21-28, 9:22-23 (stops at end of chapter 7)
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "tzav",
		Leyning: &types.Leyning{
			Haftarah:       "Jeremiah 7:21-8:3, 9:22-23",
			HaftarahChabad: "Jeremiah 7:21-28, 9:22-23",
		},
		Description: "Haftarah: Jeremiah 7:21-8:3, 9:22-23",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "Jeremiah 7:21-28, 9:22-23" {
		t.Errorf("Tzav: got %q", events[0].Leyning.Haftarah)
	}
}

func TestPatch_Rule1_Behar_AHTruncates(t *testing.T) {
	// A & S: Jeremiah 32:6-27
	// AH:    Jeremiah 32:6-22 (five verses shorter)
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "behar",
		Leyning: &types.Leyning{
			Haftarah:       "Jeremiah 32:6-27",
			HaftarahChabad: "Jeremiah 32:6-22",
		},
		Description: "Haftarah: Jeremiah 32:6-27",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "Jeremiah 32:6-22" {
		t.Errorf("Behar: got %q", events[0].Leyning.Haftarah)
	}
}

func TestPatch_Rule1_ShabbatCholHamoedPesach_AHTruncates(t *testing.T) {
	// A & S: Ezekiel 37:1-17 (if present in Hebcal)
	// AH:    Ezekiel 37:1-14 (three verses shorter)
	events := []types.HebcalEvent{{
		Category: "parashat",
		Subcat:   "special-shabbat",
		Slug:     "shabbat-chol-hamoed-pesach",
		Leyning: &types.Leyning{
			Haftarah:       "Ezekiel 37:1-17",
			HaftarahChabad: "Ezekiel 37:1-14",
		},
		Description: "Haftarah: Ezekiel 37:1-17",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "Ezekiel 37:1-14" {
		t.Errorf("Shabbat Chol HaMoed Pesach: got %q", events[0].Leyning.Haftarah)
	}
}

// Rule 2 cases: A and S differ; AH follows A.

func TestPatch_Rule2_Vayeira_AHFollowsA(t *testing.T) {
	// A:  II Kings 4:1-37
	// S:  II Kings 4:1-23
	// AH: II Kings 4:1-37 (follows A, not S)
	// Note: our embedded table has 4:1-23 (S). When API provides the correct
	// value, it should override.
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "vayera",
		Leyning: &types.Leyning{
			Haftarah:       "II Kings 4:1-37",
			HaftarahChabad: "II Kings 4:1-37",
		},
		Description: "Haftarah: II Kings 4:1-37",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "II Kings 4:1-37" {
		t.Errorf("Vayeira: got %q", events[0].Leyning.Haftarah)
	}
}

func TestPatch_Rule2_Beshalach_AHFollowsA(t *testing.T) {
	// A:  Judges 4:4-5:31
	// S:  Judges 5:1-5:31
	// AH: Judges 4:4-5:31 (follows A)
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "beshalach",
		Leyning: &types.Leyning{
			Haftarah:       "Judges 4:4-5:31",
			HaftarahChabad: "Judges 4:4-5:31",
		},
		Description: "Haftarah: Judges 4:4-5:31",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "Judges 4:4-5:31" {
		t.Errorf("Beshalach: got %q", events[0].Leyning.Haftarah)
	}
}

func TestPatch_Rule2_Pekudei_AHFollowsA(t *testing.T) {
	// A:  I Kings 7:51-8:21
	// S:  I Kings 7:40-50
	// AH: I Kings 7:51-8:21 (follows A)
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "pekudei",
		Leyning: &types.Leyning{
			Haftarah:       "I Kings 7:51-8:21",
			HaftarahChabad: "I Kings 7:51-8:21",
		},
		Description: "Haftarah: I Kings 7:51-8:21",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "I Kings 7:51-8:21" {
		t.Errorf("Pekudei: got %q", events[0].Leyning.Haftarah)
	}
}

func TestPatch_Rule2_AchareiMot_AHFollowsA(t *testing.T) {
	// A:  Amos 9:7-15 (used when combined with Kedoshim too)
	// S:  Ezekiel 22:1-16 (standalone) or 20:2-20 (combined)
	// AH: Amos 9:7-15 (follows A)
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "achrei-mot",
		Leyning: &types.Leyning{
			Haftarah:       "Ezekiel 22:1-19",
			HaftarahChabad: "Amos 9:7-15",
		},
		Description: "Haftarah: Ezekiel 22:1-19",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "Amos 9:7-15" {
		t.Errorf("Acharei Mot: got %q", events[0].Leyning.Haftarah)
	}
}

func TestPatch_Rule2_VezotHaberachah_AHFollowsA(t *testing.T) {
	// A:  Joshua 1:1-18
	// S:  Joshua 1:1-9
	// AH: Joshua 1:1-18 (follows A)
	events := []types.HebcalEvent{{
		Category: "parashat",
		Slug:     "vezot-haberakhah",
		Leyning: &types.Leyning{
			Haftarah:       "Joshua 1:1-18",
			HaftarahChabad: "Joshua 1:1-18",
		},
		Description: "Haftarah: Joshua 1:1-18",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "Joshua 1:1-18" {
		t.Errorf("Vezot Haberachah: got %q", events[0].Leyning.Haftarah)
	}
}

func TestPatch_Rule2_ShabbatZachor_AHFollowsA(t *testing.T) {
	// A:  I Samuel 15:2-34
	// S:  I Samuel 15:1-34
	// AH: I Samuel 15:2-34 (follows A)
	events := []types.HebcalEvent{{
		Category: "holiday",
		Subcat:   "special-shabbat",
		Slug:     "shabbat-zachor",
		Leyning: &types.Leyning{
			Haftarah:       "I Samuel 15:1-34",
			HaftarahChabad: "I Samuel 15:2-34",
		},
		Description: "Haftarah: I Samuel 15:1-34",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "I Samuel 15:2-34" {
		t.Errorf("Shabbat Zachor: got %q", events[0].Leyning.Haftarah)
	}
}

// Rule 2 special case: AH is unique, matching neither A nor S exactly.

func TestPatch_Rule2_ShabbatHaChodesh_AHUnique(t *testing.T) {
	// A:  Ezekiel 45:16-46:18
	// S:  Ezekiel 45:18-46:15
	// AH: Ezekiel 45:18-46:16 (starts like S, ends one verse later than S)
	// Note: embedded table has A range (45:16-46:18). API should provide correct unique range.
	events := []types.HebcalEvent{{
		Category: "holiday",
		Subcat:   "special-shabbat",
		Slug:     "shabbat-hachodesh",
		Leyning: &types.Leyning{
			Haftarah:       "Ezekiel 45:16-46:18",
			HaftarahChabad: "Ezekiel 45:18-46:16",
		},
		Description: "Haftarah: Ezekiel 45:16-46:18",
	}}
	if err := Patch(events); err != nil {
		t.Fatal(err)
	}
	if events[0].Leyning.Haftarah != "Ezekiel 45:18-46:16" {
		t.Errorf("Shabbat HaChodesh: got %q", events[0].Leyning.Haftarah)
	}
}

// --- loadTable tests ---

func TestLoadTable_ValidEntries(t *testing.T) {
	table, err := loadTable()
	if err != nil {
		t.Fatal(err)
	}
	// Spot-check a few entries.
	cases := []struct {
		slug string
		book string
	}{
		{"bereshit", "Isaiah"},
		{"noach", "Isaiah"},
		{"shabbat-zachor", "I Samuel"},
		{"shabbat-hagadol", "Malachi"},
	}
	for _, tc := range cases {
		e, ok := table[tc.slug]
		if !ok {
			t.Errorf("missing slug %q", tc.slug)
			continue
		}
		if e.Book != tc.book {
			t.Errorf("slug %q: Book = %q, want %q", tc.slug, e.Book, tc.book)
		}
	}
}

func TestLoadTable_SkipsMetaKeys(t *testing.T) {
	table, err := loadTable()
	if err != nil {
		t.Fatal(err)
	}
	for k := range table {
		if k[0] == '_' {
			t.Errorf("meta key %q should have been skipped", k)
		}
	}
}
