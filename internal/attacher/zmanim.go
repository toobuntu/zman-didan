// Package attacher appends zmanim lines to event descriptions for Shabbos,
// Yom Tov, fast days, Pesach seder night, Tisha B'Av, and Chanuka.
//
// Zero-valued zmanim are silently omitted. Some zmanim are conditionally
// present in the RSS feed: chabad.org includes Chatzos HaLailah only on
// relevant nights. Misheyakir is always present — the label differs between
// weekday ("Earliest Tallit and Tefillin") and Shabbos/YT ("Earliest Tallit")
// but the zman itself is real and relevant for Tallis on Shabbos.
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
			// Candle lighting: show shkia and tzeis as context.
			appendZmanim(ev, buildLines(
				zmanimLine("Shkia", z.Shkiah, tz),
				zmanimLine("Tzeis", z.Tzeis, tz),
			))

		case ev.Category == "havdalah" && hasZmanim:
			// Motzoei Shabbos/YT: show the Shma window as a range and shkia.
			// The event title already carries the tosfos offset, e.g. "Havdala (+4): 8:07 PM".
			// Misheyakir is earliest Shma; LatestShema is the deadline.
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

		if isPesachNight(ev) && hasZmanim && !z.ChatzosHalaila.IsZero() {
			appendZmanim(ev, fmt.Sprintf("Chatzos HaLailah: %s", fmtTime(z.ChatzosHalaila, tz)))
		}

		if isChanuka(ev) && hasZmanim {
			if note := chanukaMenoraNote(ev, events, z, tz); note != "" {
				appendZmanim(ev, note)
			}
		}
	}
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

func isPesachNight(ev *types.HebcalEvent) bool {
	return ev.Category == "holiday" &&
		containsFold(ev.Title, "pesach") &&
		containsFold(ev.HDate, "15 Nisan")
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
