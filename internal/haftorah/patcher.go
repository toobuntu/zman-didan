// Package haftorah replaces Hebcal haftorah assignments with Chabad-authoritative
// readings from the embedded haftorah_chabad.json table.
//
// NOTE: haftorah_chabad.json requires verification against
// https://www.chabad.org/library/article_cdo/aid/4158333 before distribution.
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
		ent, ok := table[ev.Slug]
		if !ok {
			continue
		}
		newRef := fmt.Sprintf("%s %s", ent.Book, ent.Ref)
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
