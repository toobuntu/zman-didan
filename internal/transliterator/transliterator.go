// Package transliterator applies Ashkenazi substitutions to event Title and
// Description fields, and optionally strips Hebrew nikud (vowel points).
//
// The substitution table is applied longest-match-first to prevent partial
// replacements (e.g. "Shemini Atzeres" must replace before "Atzeres" alone).
package transliterator

import (
	"sort"
	"strings"
	"unicode"

	"github.com/toobuntu/zman-didan/internal/types"
)

// substitutions maps Sephardic/modern/JPS forms to Ashkenazi forms.
// Order here does not matter — Apply sorts by length before substituting.
var substitutions = [][2]string{
	// Seforim names
	{"Genesis", "Bereishis"},
	{"Exodus", "Shemos"},
	{"Leviticus", "Vayikra"},
	{"Numbers", "Bamidbar"},
	{"Deuteronomy", "Devarim"},
	// Nevi'im names
	{"Jeremiah", "Yirmiyahu"},
	{"Ezekiel", "Yechezkel"},
	{"Isaiah", "Yeshayahu"},
	{"I Samuel", "I Shmuel"},
	{"II Samuel", "II Shmuel"},
	{"I Kings", "I Melachim"},
	{"II Kings", "II Melachim"},
	{"Zechariah", "Zecharya"},
	{"Malachi", "Malachi"},
	{"Hosea", "Hoshea"},
	{"Amos", "Amos"},
	{"Obadiah", "Ovadya"},
	{"Jonah", "Yonah"},
	{"Micah", "Micha"},
	{"Nahum", "Nachum"},
	{"Habakkuk", "Chavakuk"},
	{"Zephaniah", "Tzefanya"},
	{"Haggai", "Chagai"},
	// Holiday names
	{"Shemini Atzeret", "Shemini Atzeres"},
	{"Simchat Torah", "Simchas Torah"},
	{"Sukkot", "Succos"},
	{"Shavuot", "Shavuos"},
	{"Rosh HaShanah", "Rosh Hashana"},
	{"Rosh HaShana", "Rosh Hashana"},
	{"Rosh Hashana", "Rosh Hashana"},
	{"Hanukkah", "Chanuka"},
	{"Chanukah", "Chanuka"},
	// Parsha names (Sephardic → Ashkenazi)
	{"Vezot Haberakhah", "V'Zos Habracha"},
	{"Beha'alotcha", "Beha'alosecha"},
	{"Achrei Mot", "Acharei Mos"},
	{"Bechukotai", "Bechukosai"},
	{"Ki Teitzei", "Ki Seitzei"},
	{"Lech-Lecha", "Lech Lecha"},
	{"Vayetzei", "Vayeitzei"},
	{"Vayeshev", "Vayeishev"},
	{"Vaetchanan", "Va'eschanan"},
	{"Bereishit", "Bereishis"},
	{"Bereshit", "Bereishis"},
	{"Ha'Azinu", "Ha'azinu"},
	{"Miketz", "Mikeitz"},
	{"Ki Tavo", "Ki Savo"},
	{"Ki Tisa", "Ki Sisa"},
	{"Toldot", "Toldos"},
	{"Vayera", "Vayeira"},
	{"Shemot", "Shemos"},
	{"Vaera", "Va'eira"},
	{"Yitro", "Yisro"},
	{"Shmini", "Shemini"},
	{"Chukat", "Chukas"},
	{"Matot", "Matos"},
	{"Nasso", "Naso"},
	// Terms
	{"Haftarah", "Haftorah"},
	{"Haftara", "Haftorah"},
	{"Shabbat", "Shabbos"},
	{"Havdalah", "Havdala"},
}

// Apply runs the Ashkenazi substitution table over all event Title,
// Description, and Memo fields. If stripNikud is true, Hebrew nikud
// codepoints (U+05B0–U+05C7) are also removed.
func Apply(events []types.HebcalEvent, stripNikud bool) {
	table := buildTable()
	for i := range events {
		ev := &events[i]
		ev.Title = applyTable(ev.Title, table)
		ev.Description = applyTable(ev.Description, table)
		ev.Memo = applyTable(ev.Memo, table)
		if stripNikud {
			ev.Title = removeNikud(ev.Title)
			ev.Hebrew = removeNikud(ev.Hebrew)
			ev.Description = removeNikud(ev.Description)
		}
	}
}

func buildTable() [][2]string {
	table := make([][2]string, len(substitutions))
	copy(table, substitutions)
	sort.Slice(table, func(i, j int) bool {
		return len(table[i][0]) > len(table[j][0])
	})
	return table
}

func applyTable(s string, table [][2]string) string {
	for _, pair := range table {
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
