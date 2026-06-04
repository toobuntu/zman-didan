// Package transliterator applies Ashkenazi substitutions to event fields,
// and optionally strips Hebrew nikud.
//
// transliterations.json has two scopes:
//
//   - "title_only": biblical book names (Genesis → Bereishis, etc.) applied
//     only to Title and Memo. NOT applied to Description, where these words
//     appear as English prose (e.g. "commemorates the Exodus from Egypt").
//
//   - "all": holiday and parsha names applied to Title, Description, and Memo.
//
// Both tables are sorted longest-source-first to prevent partial replacements
// (e.g. "Shemini Atzeres" must match before a hypothetical "Atzeres" entry).
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

type tables struct {
	TitleOnly [][2]string `json:"title_only"`
	All       [][2]string `json:"all"`
}

var (
	once       sync.Once
	titleTable [][2]string
	allTable   [][2]string
)

func loadTables() {
	once.Do(func() {
		var t tables
		if err := json.Unmarshal(embeddata.TransliterationsJSON, &t); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: loading transliterations.json: %v\n", err)
			return
		}
		sortLongestFirst := func(pairs [][2]string) {
			sort.Slice(pairs, func(i, j int) bool {
				return len(pairs[i][0]) > len(pairs[j][0])
			})
		}
		sortLongestFirst(t.TitleOnly)
		sortLongestFirst(t.All)
		titleTable = t.TitleOnly
		allTable = t.All
	})
}

// Apply runs the Ashkenazi substitution tables over event fields.
// title_only substitutions (biblical book names) apply to Title and Memo.
// all substitutions apply to Title, Description, and Memo.
// If stripNikud is true, Hebrew nikud codepoints are also removed.
func Apply(events []types.HebcalEvent, stripNikud bool) {
	loadTables()
	for i := range events {
		ev := &events[i]
		// Apply all-scope table first (longer matches first).
		ev.Title = applyTable(ev.Title, allTable)
		ev.Description = applyTable(ev.Description, allTable)
		ev.Memo = applyTable(ev.Memo, allTable)
		// Apply title-only table to Title and Memo (not Description).
		ev.Title = applyTable(ev.Title, titleTable)
		ev.Memo = applyTable(ev.Memo, titleTable)
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
