// Package attacher appends zmanim lines to event descriptions for Shabbos,
// Yom Tov, fast days, Pesach seder night, Tisha B'Av, and Chanuka.
//
// Zero-valued zmanim (time.Time{}) are silently omitted. chabad.org omits
// certain zmanim on days they are halachically irrelevant — for example,
// Misheyakir (Earliest Tallit and Tefillin) is not included on Shabbos
// since Tefillin is not worn. Displaying "—" for such times is unhelpful.
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
				zmanimLine("Shkiah", z.Shkiah, tz),
				zmanimLine("Tzeis HaKochavim", z.Tzeis, tz),
			))

		case ev.Category == "havdalah" && hasZmanim:
			appendZmanim(ev, buildLines(
				zmanimLine("Misheyakir", z.Misheyakir, tz),
				zmanimLine("Latest Shema (Alter Rebbe)", z.LatestShema, tz),
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

// zmanimLine returns "Label: HH:MM AM" or "" if t is zero.
func zmanimLine(label string, t time.Time, tz *time.Location) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s: %s", label, fmtTime(t, tz))
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
		fmt.Fprintf(&sb, "  L'chatchila: after shkiah, before tzeis (%s–%s).\n", fmtTime(z.Shkiah, tz), fmtTime(z.Tzeis, tz))
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

func fmtTime(t time.Time, tz *time.Location) string {
	if t.IsZero() {
		return ""
	}
	return t.In(tz).Format("3:04 PM")
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
