# Didan

A CLI tool (with thin SwiftUI macOS frontend) that generates a Chabad-minhag Jewish
calendar as an iCalendar (.ics) file. Fetches Hebrew calendar data from the Hebcal
JSON API, replaces all zmanim with Chabad-authoritative times from chabad.org,
applies Ashkenazi transliterations, substitutes Chabad haftorah readings, adds
Chabad Yomei d'Pagra, and enriches event descriptions with additional zmanim for
Shabbos, Yom Tov, and fast days.

## Tech Stack

- **Module**: `github.com/toobuntu/didan`
- **CLI core**: Go (>= 1.22), single static binary `didan`
- **GUI**: SwiftUI macOS app (`gui/`) — thin wrapper invoking the CLI binary (Phase 2)
- **Key dependencies**: `github.com/spf13/cobra` (CLI), `github.com/arran4/golang-ical` (iCal)
- **Static data**: JSON embedded via `//go:embed` in `internal/embeddata/` — curated
  during development, not fetched at runtime
- **Cache**: `~/.cache/didan/zmanim.json` — keyed by `YYYY-MM-DD|ZIP`, entries
  pruned when their date is earlier than today

## Project Layout

```
didan/
├── cmd/
│   └── didan/
│       └── main.go             # CLI entry point, cobra commands + flags
├── internal/
│   ├── types/
│   │   └── types.go            # shared structs: Location, HebcalEvent, ZmanimDay, Config
│   ├── hebcal/
│   │   └── client.go           # Hebcal JSON API fetch + normalisation
│   ├── chabad/
│   │   └── client.go           # chabad.org zmanim RSS + candle ICS fetch
│   ├── cache/
│   │   └── zmanim.go           # read/write/prune ~/.cache/didan/zmanim.json
│   ├── generator/
│   │   └── generator.go        # pipeline orchestration
│   ├── patcher/
│   │   └── candle.go           # replace hebcal candle/havdalah times
│   ├── attacher/
│   │   └── zmanim.go           # attach zmanim to event descriptions
│   ├── fastday/
│   │   └── builder.go          # synthesise fast-day begin/end events
│   ├── alarm/
│   │   └── builder.go          # strip and rebuild VALARM blocks
│   ├── transliterator/
│   │   └── transliterator.go   # Ashkenazi substitution + nikud strip
│   ├── haftorah/
│   │   └── patcher.go          # Chabad haftorah assignments
│   ├── cleaner/
│   │   └── description.go      # normalise event descriptions
│   ├── specialdates/
│   │   └── merge.go            # merge Chabad Yomei d'Pagra
│   ├── embeddata/
│   │   ├── embeddata.go        # go:embed declarations
│   │   └── files/
│   │       ├── haftorah_chabad.json
│   │       └── yomei_dpagra.json
│   └── icalwriter/
│       └── writer.go           # serialise to RFC 5545 .ics
├── gui/                        # SwiftUI macOS app (Phase 2)
│   └── Didan.xcodeproj
├── go.mod
├── go.sum
├── ARCHITECTURE.md
└── CLAUDE.md
```

## Building and Running

```sh
go mod tidy
go build -o didan ./cmd/didan
./didan generate --year 5786 --zip 17601
./didan generate --year 5786 --zip 17601 --lang ah --candles 25 --output ~/Desktop
go test ./...

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o didan-linux ./cmd/didan
```

## GitHub

Repository: `https://github.com/toobuntu/didan`

To create the remote from an existing local repo with branches already committed:

```sh
gh repo create toobuntu/didan --private --source=. --remote=origin --push
git push origin ruby
```

## Key Design Decisions

### Data flow

Hebcal JSON API → normalise events → fetch Chabad candle ICS (year) →
fetch Chabad zmanim (per date, cached) → patch candle/havdalah times →
attach zmanim to descriptions → build fast-day events → rebuild alarms →
patch haftorahs → apply transliteration → clean descriptions →
merge Yomei d'Pagra → sort → write iCal

### Hebcal as structural skeleton only

Hebcal supplies the calendar structure: which days have which events, parsha
assignments, categories, Hebrew text. All time values are authoritative from
chabad.org. Hebcal times are placeholders only.

### internal/ packages

Go's `internal/` directory restricts imports to code within the parent tree.
Each pipeline stage is its own package; none import from each other — only
from `internal/types`. This keeps the dependency graph acyclic.

### Static JSON data

Haftorah assignments and Yomei d'Pagra descriptions are curated once during
development and embedded in the binary at compile time via `//go:embed`.
No runtime network access for these.

### Language / transliteration modes

| `--lang`       | Hebcal `lg=` | Post-processing                       |
|----------------|--------------|---------------------------------------|
| `he` (default) | `he`         | none                                  |
| `he-x-NoNikud` | `he-x-NoNikud` | none                                |
| `a`            | `a`          | Ashkenazi substitution table          |
| `ah`           | `ah`         | Ashkenazi substitution table          |
| `ah-x-NoNikud` | `ah`         | Ashkenazi table + strip U+05B0–U+05C7 |

`ah-x-NoNikud` is not a native Hebcal parameter; nikud is stripped post-fetch.

### Candle lighting offset

`--candles N` (default 25). Passed as `b=N` to Hebcal and `before=N` to the
Chabad candle ICS endpoint. The Rebbe's practice was 23–25 minutes before shkiah.

### Alarm policy

| Event type      | VALARM triggers                       |
|-----------------|---------------------------------------|
| Candle lighting | −2h, −15min                           |
| Havdalah        | −10min, PT0S (at event time)          |
| Fast begin      | −2h, −30min                           |
| Fast end        | −15min, PT0S (at event time)          |
| All-day events  | none                                  |

Apple Calendar honours up to 2 VALARM components per VEVENT; we emit exactly 2.

### Cache invalidation

Entries keyed `YYYY-MM-DD|ZIP`. Pruned on every write when entry date < today.
`--refresh` bypasses the cache for the entire run.

### Output filename

`didan_HHHH_ZZZZZ.ics` — Hebrew year + ZIP. German: `didan_HHHH_ZZZZZ_de.ics`.

## Data Sources

| Source | Used for |
|--------|----------|
| `https://www.hebcal.com/hebcal?v=1&cfg=json&...` | Calendar structure, parsha/leyning |
| `https://www.chabad.org/tools/rss/zmanim.xml?...&tdate=YYYY-MM-DD` | All zmanim per date |
| `https://www.chabad.org/calendar/candlelighting/candlelighting.ics.asp?...` | Candle + havdalah year |
| `https://download.hebcal.com/ical/chabad-special-dates.ics` | Yomei d'Pagra base list |
| `internal/embeddata/files/haftorah_chabad.json` | Chabad haftorah assignments (static, verify before distribution) |
| `internal/embeddata/files/yomei_dpagra.json` | Yomei d'Pagra summaries (static) |

## Phases

### Phase 1 (current — implemented)
Candle lighting + havdalah replacement, all zmanim, fast-day events, alarm
rebuild, haftorah replacement, transliteration, Yomei d'Pagra merge, description
normalisation, iCal output.

### Phase 2
German output, parsha summaries, SwiftUI GUI wrapper.

### Phase 3
Sefaria integration; additional language support.

## Halachic Notes

- Candle lighting: configurable minutes before shkiah (default 25).
- Havdalah: tzeis hakochavim per chabad.org (~36 min / 8.5° after shkiah for
  regular Shabbos; Motzoei Yom Tov may differ — use chabad.org value always).
- Misheyakir: earliest tallis and tefillin per Alter Rebbe calculation.
- Latest shema: 3 proportional hours after hanetz amiti (true sunrise).
  Appended to Shabbos/YT havdalah event description.
- Tisha B'Av and Yom Kippur fast begin: previous day's shkiah (not alos).
- All other fasts: alos hashachar as fast begin time.
- Chatzos halaila: added to Pesach seder night event description.
- Chatzos hayom: added to Tisha B'Av event description.
- Chanuka menora: prioritized lighting note in description (no alarm):
    l'chatchila window (shkiah–tzeis); b'dieved up to tzeis+30min;
    from Plag Hamincha b'dieved / Erev Shabbos; Motzoei Shabbos after Havdalah.
- All zmanim: chabad.org is authoritative in all cases.
