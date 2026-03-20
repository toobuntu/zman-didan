// Package transliterator applies Ashkenazi substitutions to event Title,
// Description, and Memo fields, and optionally strips Hebrew nikud.
//
// The substitution table is loaded from the embedded transliterations.json
// and applied longest-source-first to prevent partial replacements
// (e.g. "Shemini Atzeres" must replace before "Atzeres" alone).
package transliterator

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/toobuntu/zman-didan/internal/embeddata"
	"github.com/toobuntu/zman-didan/internal/types"
)

var (
	tableOnce sync.Once
	table     [][2]string
)

// loadTable parses transliterations.json and sorts longest-source-first.
// Called once; result is cached in the package-level table variable.
func loadTable() [][2]string {
	tableOnce.Do(func() {
		var pairs [][2]string
		if err := json.Unmarshal(embeddata.TransliterationsJSON, &pairs); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: loading transliterations.json: %v\n", err)
			return
		}
		sort.Slice(pairs, func(i, j int) bool {
			return len(pairs[i][0]) > len(pairs[j][0])
		})
		table = pairs
	})
	return table
}

// Apply runs the Ashkenazi substitution table over all event Title,
// Description, and Memo fields. If stripNikud is true, Hebrew nikud
// codepoints (U+05B0–U+05C7) are also removed.
func Apply(events []types.HebcalEvent, stripNikud bool) {
	t := loadTable()
	for i := range events {
		ev := &events[i]
		ev.Title = applyTable(ev.Title, t)
		ev.Description = applyTable(ev.Description, t)
		ev.Memo = applyTable(ev.Memo, t)
		if stripNikud {
			ev.Title = removeNikud(ev.Title)
			ev.Hebrew = removeNikud(ev.Hebrew)
			ev.Description = removeNikud(ev.Description)
		}
	}
}

func applyTable(s string, t [][2]string) string {
	for _, pair := range t {
		s = strings.ReplaceAll(s, pair[0], pair[1])
	}
	return s
}

// removeNikud strips Hebrew vowel points and related codepoints:
//
//	U+05B0–U+05BD  vowel points (nikud)
//	U+05BF         point rafe
//	U+05C1–U+05C2  shin/sin dot
//	U+05C4–U+05C5  upper/lower dot
//	U+05C7         qamats qatan
func removeNikud(s string) string {
	return strings.Map(func(r rune) rune {
		if isNikud(r) {
			return -1
		}
		return r
	}, s)
}

func isNikud(r rune) bool {
	return (r >= 0x05B0 && r <= 0x05BD) ||
		r == 0x05BF ||
		r == unicode.ReplacementChar ||
		(r >= 0x05C1 && r <= 0x05C2) ||
		(r >= 0x05C4 && r <= 0x05C5) ||
		r == 0x05C7
}
