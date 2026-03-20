// Package embeddata exposes static JSON data files embedded at compile time.
//
// Files are embedded from the files/ subdirectory alongside this package.
// Edit any JSON file and rebuild — no Go code change required.
//
// Data files:
//
//   - haftorah_chabad.json   Chabad haftorah by parsha slug (needs verification
//     against https://www.chabad.org/library/article_cdo/aid/4158333)
//   - yomei_dpagra.json      Chabad special dates with Chitas summaries
//   - transliterations.json  Sephardic/modern → Ashkenazi substitution table
//   - rebbes.json            Unified rebbe biographical data: honorific, verbose
//     name variants, and Hebrew birth/death years. Source of truth for name
//     normalization, birthday "(N years ago)" annotation, and yahrzeit
//     death-year disambiguation.
package embeddata

import _ "embed"

//go:embed files/haftorah_chabad.json
var HaftoraChabadJSON []byte

//go:embed files/yomei_dpagra.json
var YomeiDpagraJSON []byte

// TransliterationsJSON is the Ashkenazi substitution table.
// Format: JSON array of [source, target] string pairs, applied
// longest-source-first to prevent partial replacements.
//
//go:embed files/transliterations.json
var TransliterationsJSON []byte

// RebbesJSON is the unified rebbe biographical data table.
// Format: JSON array of objects with fields:
//
//	honorific     string   standard Chabad name, unique across all entries
//	verbose_names []string all name forms observed in Hebcal feed data
//	birth_year    int      Hebrew calendar year of birth
//	death_year    int      Hebrew calendar year of death
//
// death_year enables disambiguation when two figures share a verbose name:
// the year computed from a Hebcal yahrzeit description (observance_year − N)
// must match death_year.
//
//go:embed files/rebbes.json
var RebbesJSON []byte
