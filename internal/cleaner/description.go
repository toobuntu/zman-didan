// SPDX-FileCopyrightText: 2026 toobuntu
// SPDX-License-Identifier: GPL-3.0-or-later

// Package cleaner normalises event descriptions by removing Hebcal-specific
// boilerplate, redundant subtitles, and JPS attribution lines.
package cleaner

import (
	"regexp"
	"strings"

	"github.com/toobuntu/zman-didan/internal/types"
)

var redundantPatterns = []*regexp.Regexp{
	// "Also spelled X" disambiguation lines from Hebcal.
	regexp.MustCompile(`(?i)also spelled [^\n]+`),
	// "Also known as X" disambiguation lines.
	regexp.MustCompile(`(?i)also known as [^\n]+`),
	// Hebcal URL references — chabad.org links added by the pipeline are kept.
	regexp.MustCompile(`https?://hebcal\.com[^\n]*`),
	// JPS attribution.
	regexp.MustCompile(`(?i)(source:\s*)?jps\s+translation[^\n]*`),
}

var multiBlank = regexp.MustCompile(`\n{3,}`)

// Clean applies the normalisation pass to all events in place.
func Clean(events []types.HebcalEvent) {
	for i := range events {
		ev := &events[i]
		ev.Description = cleanText(ev.Description)
		ev.Memo = cleanText(ev.Memo)
	}
}

func cleanText(s string) string {
	for _, re := range redundantPatterns {
		s = re.ReplaceAllString(s, "")
	}
	s = multiBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
