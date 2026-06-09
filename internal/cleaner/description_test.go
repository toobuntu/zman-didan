/*
 * SPDX-FileCopyrightText: Copyright 2026 Todd Schulman
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package cleaner

import (
	"testing"

	"github.com/toobuntu/zman-didan/internal/types"
)

func TestClean_AlsoSpelled(t *testing.T) {
	events := []types.HebcalEvent{{
		Description: "A holiday.\nAlso spelled Chanukah in English.\nMore info.",
	}}
	Clean(events)
	if got := events[0].Description; got != "A holiday.\n\nMore info." {
		t.Errorf("Description = %q", got)
	}
}

func TestClean_AlsoKnownAs(t *testing.T) {
	events := []types.HebcalEvent{{
		Description: "Also known as Sukkos",
	}}
	Clean(events)
	if events[0].Description != "" {
		t.Errorf("Description = %q, want empty", events[0].Description)
	}
}

func TestClean_HebcalURL(t *testing.T) {
	events := []types.HebcalEvent{{
		Description: "https://hebcal.com/s/something\nMore text",
	}}
	Clean(events)
	if got := events[0].Description; got != "More text" {
		t.Errorf("Description = %q", got)
	}
}

func TestClean_ChabadURLPreserved(t *testing.T) {
	events := []types.HebcalEvent{{
		Description: "See https://www.chabad.org/holidays/article_cdo/aid/123",
	}}
	Clean(events)
	if got := events[0].Description; got != "See https://www.chabad.org/holidays/article_cdo/aid/123" {
		t.Errorf("chabad.org URL removed: %q", got)
	}
}

func TestClean_JPSAttribution(t *testing.T) {
	events := []types.HebcalEvent{{
		Description: "Reading.\nJPS Translation of the Tanakh\nMore.",
	}}
	Clean(events)
	if got := events[0].Description; got != "Reading.\n\nMore." {
		t.Errorf("Description = %q", got)
	}
}

func TestClean_SourceJPS(t *testing.T) {
	events := []types.HebcalEvent{{
		Description: "Text.\nSource: JPS Translation 1999",
	}}
	Clean(events)
	if got := events[0].Description; got != "Text." {
		t.Errorf("Description = %q", got)
	}
}

func TestClean_MultipleBlankLines(t *testing.T) {
	events := []types.HebcalEvent{{
		Description: "Line1\n\n\n\nLine2",
	}}
	Clean(events)
	if got := events[0].Description; got != "Line1\n\nLine2" {
		t.Errorf("Description = %q", got)
	}
}

func TestClean_MemoAlsoCleaned(t *testing.T) {
	events := []types.HebcalEvent{{
		Memo: "Also known as Festival of Lights\nhttps://hebcal.com/s/chanukah",
	}}
	Clean(events)
	if events[0].Memo != "" {
		t.Errorf("Memo = %q, want empty", events[0].Memo)
	}
}

func TestClean_NoChange(t *testing.T) {
	events := []types.HebcalEvent{{
		Description: "Clean description with no boilerplate.",
	}}
	Clean(events)
	if events[0].Description != "Clean description with no boilerplate." {
		t.Errorf("Description changed unexpectedly: %q", events[0].Description)
	}
}
