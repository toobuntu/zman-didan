// SPDX-FileCopyrightText: 2026 toobuntu
// SPDX-License-Identifier: GPL-3.0-or-later

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
//   - rebbes.json            Unified rebbe biographical data. Fields used at
//     runtime: honorific, verbose_names, huledes_year, dob_gregorian,
//     histalkus_year, histalkus_gregorian. The Hebrew date strings
//     (dob_hebrew, histalkus_hebrew) are preserved for reference only.
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
// Format: JSON array of objects. See docs/rebbes_schema.md for the full
// schema and a human-readable reference table with both calendar systems.
//
//go:embed files/rebbes.json
var RebbesJSON []byte
