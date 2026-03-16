// Package embeddata exposes static JSON data files embedded at compile time.
//
// Files are embedded from the files/ subdirectory alongside this package.
// To update any data file, edit the JSON and rebuild — no code change needed.
//
// All haftorah assignments in haftorah_chabad.json must be verified against
// https://www.chabad.org/library/article_cdo/aid/4158333 before distribution.
package embeddata

import _ "embed"

// HaftoraChabadJSON is the Chabad haftorah assignment table.
// Keys are Hebcal parsha slugs (Sephardic spelling, e.g. "bereshit").
// Values contain the book name and verse range.
//
//go:embed files/haftorah_chabad.json
var HaftoraChabadJSON []byte

// YomeiDpagraJSON is the Chabad special dates description table.
// Keys are Hebrew date strings matching the chabad-special-dates.ics summaries.
//
//go:embed files/yomei_dpagra.json
var YomeiDpagraJSON []byte
