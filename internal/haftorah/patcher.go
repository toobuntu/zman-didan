// Package haftorah replaces Hebcal haftorah assignments with Chabad-authoritative
// readings.
//
// # Source priority
//
// Two sources provide Chabad haftorahs, applied in this order of preference:
//
//  1. ev.Leyning.HaftarahChabad — populated from the Hebcal API's haftarah_chabad
//     field (live on www.hebcal.com as of 2026-03-30, hebcal-rest-api v6.4.1+,
//     github.com/hebcal/hebcal-rest-api/issues/715). Non-null only when Chabad
//     custom differs from Ashkenazi standard. When null, ev.Leyning.Haftarah
//     (the Ashkenazi standard reading) is already correct for Chabad.
//
//  2. haftorah_chabad.json — embedded fallback table keyed by parsha slug.
//     Retained as insurance for edge cases and special Shabbatot not yet
//     verified against the live API. Compare a full year's API output against
//     this table, verify against chabad.org/library/article_cdo/aid/4158333,
//     then remove.
//
// Known discrepancy between embedded JSON and live API: Tzav is
// "Jeremiah 7:21-8:3, 9:22-23" in the embedded table but "Jeremiah 7:21-28,
// 9:22-23" from the API. The API value is authoritative.
package haftorah

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/toobuntu/zman-didan/internal/embeddata"
	"github.com/toobuntu/zman-didan/internal/types"
)

type entry struct {
	Book string `json:"book"`
	Ref  string `json:"ref"`
}

var haftoreLine = regexp.MustCompile(`(?i)Haftarah:[^\n]+`)

// Patch replaces Hebcal haftorah references with Chabad assignments.
// It prefers ev.Leyning.HaftarahChabad (from the Hebcal API) when present,
// and falls back to the embedded haftorah_chabad.json table otherwise.
func Patch(events []types.HebcalEvent) error {
	table, err := loadTable()
	if err != nil {
		return err
	}
	for i := range events {
		ev := &events[i]
		if ev.Category != "parashat" && ev.Subcat != "special-shabbat" {
			continue
		}

		var newRef string
		if ev.Leyning != nil && ev.Leyning.HaftarahChabad != "" {
			// API field present: Chabad reading differs from Ashkenazi standard.
			newRef = ev.Leyning.HaftarahChabad
		} else if ent, ok := table[ev.Slug]; ok {
			// Fallback: embedded table (insurance while API coverage is verified).
			newRef = fmt.Sprintf("%s %s", ent.Book, ent.Ref)
		} else {
			// null haftarah_chabad + no embedded entry: Chabad follows the
			// standard haftarah already present in ev.Leyning.Haftarah.
			continue
		}

		if ev.Leyning != nil {
			ev.Leyning.Haftarah = newRef
		}
		replacement := "Haftorah: " + newRef
		switch {
		case haftoreLine.MatchString(ev.Description):
			ev.Description = haftoreLine.ReplaceAllString(ev.Description, replacement)
		case ev.Description != "":
			ev.Description += "\n" + replacement
		default:
			ev.Description = replacement
		}
	}
	return nil
}

func loadTable() (map[string]entry, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(embeddata.HaftoraChabadJSON, &raw); err != nil {
		return nil, fmt.Errorf("parsing haftorah_chabad.json: %w", err)
	}
	table := make(map[string]entry, len(raw))
	for k, v := range raw {
		if strings.HasPrefix(k, "_") {
			continue
		}
		var e entry
		if err := json.Unmarshal(v, &e); err != nil {
			continue
		}
		if e.Book != "" {
			table[k] = e
		}
	}
	return table, nil
}
