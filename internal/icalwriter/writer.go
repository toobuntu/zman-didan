// Package icalwriter serialises a slice of HebcalEvents to RFC 5545 iCalendar
// format. Output uses CRLF line endings and folds lines exceeding 75 octets.
// Timed events use TZID= parameters (not UTC) for consistency.
package icalwriter

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/toobuntu/zman-didan/internal/types"
)

const crlf = "\r\n"

// Write serialises events to w as a VCALENDAR document.
func Write(w io.Writer, events []types.HebcalEvent, loc types.Location, year int, lang string, emojis bool) error {
	ww := &writer{w: w}
	ww.line("BEGIN:VCALENDAR")
	ww.line("VERSION:2.0")
	ww.line(fmt.Sprintf("PRODID:-//didan//Didan Calendar %d//EN", year))
	ww.line("CALSCALE:GREGORIAN")
	ww.line("METHOD:PUBLISH")
	ww.fold(fmt.Sprintf("X-WR-CALNAME:Didan %s %d", loc.City, year))
	ww.line(fmt.Sprintf("X-WR-TIMEZONE:%s", loc.TZID))
	writeVTimezone(ww, loc.TZID)
	for _, ev := range events {
		if err := writeEvent(ww, ev, loc.TZID, lang, emojis); err != nil {
			fmt.Printf("Warning: skipping event %q: %v\n", ev.Title, err)
		}
	}
	ww.line("END:VCALENDAR")
	return ww.err
}

func writeVTimezone(ww *writer, tzid string) {
	if tzid != "America/New_York" {
		return // TODO Phase 2: generate VTIMEZONE from tzdata for other zones
	}
	ww.line("BEGIN:VTIMEZONE")
	ww.line("TZID:America/New_York")
	ww.line("BEGIN:DAYLIGHT")
	ww.line("TZNAME:EDT")
	ww.line("TZOFFSETFROM:-0500")
	ww.line("TZOFFSETTO:-0400")
	ww.line("DTSTART:19700308T020000")
	ww.line("RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=2SU")
	ww.line("END:DAYLIGHT")
	ww.line("BEGIN:STANDARD")
	ww.line("TZNAME:EST")
	ww.line("TZOFFSETFROM:-0400")
	ww.line("TZOFFSETTO:-0500")
	ww.line("DTSTART:19701101T020000")
	ww.line("RRULE:FREQ=YEARLY;BYMONTH=11;BYDAY=1SU")
	ww.line("END:STANDARD")
	ww.line("END:VTIMEZONE")
}

func writeEvent(ww *writer, ev types.HebcalEvent, tzid, lang string, emojis bool) error {
	tz, err := time.LoadLocation(tzid)
	if err != nil {
		return err
	}
	uid := ev.UID
	if uid == "" {
		uid = fmt.Sprintf("didan-%s-%s", ev.Date.Format("20060102"), sanitize(ev.Title))
	}
	ww.line("BEGIN:VEVENT")
	ww.fold(fmt.Sprintf("DTSTAMP:%s", time.Now().UTC().Format("20060102T150405Z")))
	ww.fold(fmt.Sprintf("UID:%s", uid))
	ww.fold("SUMMARY:" + escapeText(buildSummary(ev, lang, emojis)))

	if ev.AllDay {
		d := ev.Date.In(tz)
		ww.line("DTSTART;VALUE=DATE:" + d.Format("20060102"))
		ww.line("DTEND;VALUE=DATE:" + d.AddDate(0, 0, 1).Format("20060102"))
		ww.line("X-MICROSOFT-CDO-ALLDAYEVENT:TRUE")
		ww.line("TRANSP:TRANSPARENT")
		ww.line("X-MICROSOFT-CDO-BUSYSTATUS:FREE")
	} else {
		t := ev.Date.In(tz).Format("20060102T150405")
		ww.fold(fmt.Sprintf("DTSTART;TZID=%s:%s", tzid, t))
		ww.fold(fmt.Sprintf("DTEND;TZID=%s:%s", tzid, t))
		ww.line("TRANSP:TRANSPARENT")
		ww.line("X-MICROSOFT-CDO-BUSYSTATUS:FREE")
	}
	ww.line("CLASS:PUBLIC")

	// Use Description if populated, else Memo. ev.Link is NOT included —
	// hebcal.com links are not required by Hebcal's terms for iCal output,
	// and they add noise. chabad.org links added by haftorah.Patch are kept
	// as they are in ev.Description already.
	desc := ev.Description
	if desc == "" {
		desc = ev.Memo
	}
	if desc != "" {
		ww.fold("DESCRIPTION:" + escapeText(desc))
	}

	for _, a := range ev.Alarms {
		ww.line("BEGIN:VALARM")
		ww.line("ACTION:DISPLAY")
		ww.fold("DESCRIPTION:" + escapeText(a.Description))
		ww.line("TRIGGER:" + triggerDuration(a.TriggerMinutes))
		ww.line("END:VALARM")
	}
	ww.line("END:VEVENT")
	return ww.err
}

// buildSummary constructs the SUMMARY value.
//
// For Ashkenazi+Hebrew modes (ah, ah-x-NoNikud) and transliteration-only
// modes (a, s, sh), we append " / Hebrew" when the Hebrew field is distinct.
// For Hebrew-only modes (he, he-x-NoNikud), Title already contains Hebrew.
// Emojis are prepended when cfg.Emojis is true, using a category lookup.
func buildSummary(ev types.HebcalEvent, lang string, emojis bool) string {
	title := ev.Title
	if ev.Hebrew != "" && ev.Hebrew != title {
		switch lang {
		case "ah", "ah-x-NoNikud", "a", "s", "sh":
			title = title + " / " + ev.Hebrew
		}
	}
	if emojis {
		if e := eventEmoji(ev); e != "" {
			title = e + " " + title
		}
	}
	return title
}

// eventEmoji returns the emoji prefix for an event, following Hebcal conventions.
// Returns "" for events with no standard emoji.
func eventEmoji(ev types.HebcalEvent) string {
	switch ev.Category {
	case "candles":
		return "🕯️"
	case "havdalah":
		return "✨"
	}
	t := strings.ToLower(ev.Title)
	switch {
	case strings.Contains(t, "rosh hashana"), strings.Contains(t, "erev rosh hashana"):
		return "🍏🍯"
	case strings.Contains(t, "yom kippur"), strings.Contains(t, "erev yom kippur"):
		return "⚖️"
	case strings.Contains(t, "succos"), strings.Contains(t, "sukkot"),
		strings.Contains(t, "shemini atzeres"), strings.Contains(t, "simchas torah"):
		return "🌿"
	case strings.Contains(t, "chanuka"):
		return "🕎"
	case strings.Contains(t, "tu bishvat"):
		return "🌱"
	case strings.Contains(t, "purim"), strings.Contains(t, "shushan purim"):
		return "🎭"
	case strings.Contains(t, "pesach"), strings.Contains(t, "erev pesach"):
		return "🫓"
	case strings.Contains(t, "shavuos"), strings.Contains(t, "shavuot"):
		return "🌸"
	case strings.Contains(t, "lag b'omer"), strings.Contains(t, "lag baomer"):
		return "🔥"
	case ev.Subcat == "fast",
		strings.Contains(t, "tzom"), strings.Contains(t, "tisha b'av"),
		strings.Contains(t, "fast begins"), strings.Contains(t, "fast ends"):
		return "✡️"
	case ev.Category == "roshchodesh":
		return "🌙"
	case strings.Contains(t, "parashat"), strings.Contains(t, "parshas"):
		return "📖"
	}
	return ""
}

func triggerDuration(minutes int) string {
	if minutes == 0 {
		return "PT0S"
	}
	neg, m := "", minutes
	if m < 0 {
		neg, m = "-", -m
	}
	switch {
	case m%60 == 0:
		return fmt.Sprintf("%sPT%dH", neg, m/60)
	case m < 60:
		return fmt.Sprintf("%sPT%dM", neg, m)
	default:
		return fmt.Sprintf("%sPT%dH%dM", neg, m/60, m%60)
	}
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

type writer struct {
	w   io.Writer
	err error
}

func (w *writer) line(s string) {
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintf(w.w, "%s%s", s, crlf)
}

func (w *writer) fold(s string) {
	if w.err != nil {
		return
	}
	const max = 75
	if len(s) <= max {
		w.line(s)
		return
	}
	var sb strings.Builder
	first := true
	for len(s) > 0 {
		width := max
		if !first {
			width = max - 1
		}
		if len(s) <= width {
			if !first {
				sb.WriteByte(' ')
			}
			sb.WriteString(s)
			sb.WriteString(crlf)
			break
		}
		cut := width
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		if !first {
			sb.WriteByte(' ')
		}
		sb.WriteString(s[:cut])
		sb.WriteString(crlf)
		s = s[cut:]
		first = false
	}
	_, w.err = fmt.Fprint(w.w, sb.String())
}
