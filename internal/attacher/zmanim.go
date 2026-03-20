// Package attacher appends zmanim lines to event descriptions for Shabbos,
// Yom Tov, fast days, Pesach seder nights, Tisha B'Av, and Chanuka.
//
// Zero-valued zmanim are silently omitted. Misheyakir is always present in
// the RSS feed — the label differs between weekday and Shabbos/YT but the
// zman itself is real and relevant for Tallis on Shabbos.
package attacher

import (
	"fmt"
	"strings"
	"time"

	"github.com/toobuntu/zman-didan/internal/types"
)

// AttachZmanim enriches event descriptions with relevant zmanim.
// zmanimMap is keyed by "YYYY-MM-DD".
func AttachZmanim(events []types.HebcalEvent, zmanimMap map[string]types.ZmanimDay, tzid string) {
	tz, _ := time.LoadLocation(tzid)
	for i := range events {
		ev := &events[i]
		key := ev.Date.In(tz).Format("2006-01-02")
		z, hasZmanim := zmanimMap[key]

		switch {
		case ev.Category == "candles" && hasZmanim:
			appendZmanim(ev, buildLines(
				zmanimLine("Shkia", z.Shkiah, tz),
				zmanimLine("Tzeis", z.Tzeis, tz),
			))
			// Append Chatzos HaLailah on seder nights (Pesach I and II).
			// The zmanim RSS includes this only when it's relevant, but we
			// further gate on isSederNight to avoid showing it on regular
			// Shabbos or non-seder YT candle events.
			if !z.ChatzosHalaila.IsZero() && isSederNight(ev, events, tz) {
				appendZmanim(ev, fmt.Sprintf("Chatzos HaLailah: %s", fmtTime(z.ChatzosHalaila, tz)))
			}

		case ev.Category == "havdalah" && hasZmanim:
			if parsha := findParshaTitle(events, key, tz); parsha != "" {
				if ev.Description == "" {
					ev.Description = parsha
				} else {
					ev.Description = parsha + "\n\n" + ev.Description
				}
			}
			appendZmanim(ev, buildLines(
				zmanimRange("Shma", z.Misheyakir, z.LatestShema, tz),
				zmanimLine("Shkia", z.Shkiah, tz),
			))

		case ev.Category == "holiday" && ev.Subcat == "fast" && hasZmanim:
			if containsFold(ev.Title, "tisha b'av") {
				appendZmanim(ev, buildLines(
					zmanimLine("Chatzos Hayom", z.Chatzos, tz),
				))
			}
		}

		if isChanuka(ev) && hasZmanim {
			if note := chanukaMenoraNote(ev, events, z, tz); note != "" {
				appendZmanim(ev, note)
			}
		}
	}
}

// isSederNight reports whether a candle lighting event falls on a Pesach seder
// night. Seder nights are:
//   - The night of Erev Pesach (any co-event on the same date has "Erev Pesach"
//     in its title).
//   - The second seder night, indicated by "after" in the candle event title
//     (rebuildCandle uses "YT candles after" for this case) when a Pesach
//     holiday event is present on the same date.
//
// We do not show Chatzos HaLailah on Shabbos Chol HaMoed Pesach candles or
// the Pesach VI/VII candle events because those are not seder nights.
func isSederNight(ev *types.HebcalEvent, allEvents []types.HebcalEvent, tz *time.Location) bool {
	dateKey := ev.Date.In(tz).Format("2006-01-02")
	for _, other := range allEvents {
		if other.Date.In(tz).Format("2006-01-02") != dateKey || !other.AllDay {
			continue
		}
		t := strings.ToLower(other.Title)
		if strings.Contains(t, "erev pesach") {
			return true // first seder night
		}
		// Second seder: candle event title contains "after" (YT candles after)
		// and there's a Pesach I all-day event on the same date.
		if strings.Contains(t, "pesach i") && !strings.Contains(t, "ch''m") &&
			containsFold(ev.Title, "after") {
			return true
		}
	}
	return false
}

// findParshaTitle returns the Title of the parashat event on dateKey, or "".
func findParshaTitle(events []types.HebcalEvent, dateKey string, tz *time.Location) string {
	for _, ev := range events {
		if ev.Category == "parashat" && ev.Date.In(tz).Format("2006-01-02") == dateKey {
			return ev.Title
		}
	}
	return ""
}

// zmanimLine returns "Label: TIME" or "" if t is zero.
func zmanimLine(label string, t time.Time, tz *time.Location) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s: %s", label, fmtTime(t, tz))
}

// zmanimRange returns "Label: START–END", collapsing the AM/PM suffix when
// both times share the same period: "Shma: 6:18–10:07 AM".
// Falls back to individual times if only one is non-zero.
func zmanimRange(label string, start, end time.Time, tz *time.Location) string {
	switch {
	case start.IsZero() && end.IsZero():
		return ""
	case start.IsZero():
		return zmanimLine(label, end, tz)
	case end.IsZero():
		return zmanimLine(label, start, tz)
	}
	s, e := start.In(tz), end.In(tz)
	if (s.Hour() < 12) == (e.Hour() < 12) {
		period := "AM"
		if s.Hour() >= 12 {
			period = "PM"
		}
		return fmt.Sprintf("%s: %s–%s %s", label, s.Format("3:04"), e.Format("3:04"), period)
	}
	return fmt.Sprintf("%s: %s–%s", label, fmtTime(start, tz), fmtTime(end, tz))
}

// buildLines joins non-empty lines with newlines.
func buildLines(lines ...string) string {
	var out []string
	for _, l := range lines {
		if l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// appendZmanim appends zmanim text to an event description, separated by a
// blank line from any existing description content.
func appendZmanim(ev *types.HebcalEvent, lines string) {
	if lines == "" {
		return
	}
	if ev.Description == "" {
		ev.Description = lines
	} else {
		ev.Description = ev.Description + "\n\n" + lines
	}
}

func chanukaMenoraNote(ev *types.HebcalEvent, allEvents []types.HebcalEvent, z types.ZmanimDay, tz *time.Location) string {
	dateKey := ev.Date.In(tz).Format("2006-01-02")
	hasCandles, hasHavdalah := false, false
	for _, other := range allEvents {
		if other.Date.In(tz).Format("2006-01-02") != dateKey {
			continue
		}
		switch other.Category {
		case "candles":
			hasCandles = true
		case "havdalah":
			hasHavdalah = true
		}
	}
	if z.Shkiah.IsZero() && z.Tzeis.IsZero() && z.PlagHamincha.IsZero() {
		return ""
	}
	latest := z.Tzeis.Add(30 * time.Minute)
	var sb strings.Builder
	sb.WriteString("Menora lighting:\n")
	switch {
	case hasCandles:
		fmt.Fprintf(&sb, "  From Plag HaMincha (%s) before Shabbos candles.", fmtTime(z.PlagHamincha, tz))
	case hasHavdalah:
		fmt.Fprintf(&sb, "  After Havdalah. Earliest: %s (tzeis).", fmtTime(z.Tzeis, tz))
	default:
		fmt.Fprintf(&sb, "  L'chatchila: after shkia, before tzeis (%s–%s).\n", fmtTime(z.Shkiah, tz), fmtTime(z.Tzeis, tz))
		fmt.Fprintf(&sb, "  B'dieved: until %s (tzeis+30 min).\n", fmtTime(latest, tz))
		fmt.Fprintf(&sb, "  If necessary: from Plag HaMincha (%s) b'dieved.", fmtTime(z.PlagHamincha, tz))
	}
	return sb.String()
}

func isChanuka(ev *types.HebcalEvent) bool {
	return ev.Category == "holiday" && containsFold(ev.Title, "chanuk")
}

// fmtTime formats t in 12-hour format with uppercase AM/PM.
func fmtTime(t time.Time, tz *time.Location) string {
	if t.IsZero() {
		return ""
	}
	return t.In(tz).Format("3:04 PM")
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
