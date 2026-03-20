# didan

A CLI tool that generates a Chabad-minhag Jewish calendar as an iCalendar (.ics)
file. Fetches Hebrew calendar data from Hebcal, replaces all zmanim with
Chabad-authoritative times from chabad.org, applies Ashkenazi transliterations,
substitutes Chabad haftorah readings, adds Chabad Yomei d'Pagra, and enriches
event descriptions with additional zmanim.

## Module and binary

- **Module**: `github.com/toobuntu/zman-didan`
- **Binary**: `bin/didan` (built with `make build`)

## Project layout

```
didan/
├── cmd/didan/main.go              # cobra CLI — all flags, validation
├── internal/
│   ├── types/types.go             # shared structs (Config, HebcalEvent, ZmanimDay, …)
│   ├── hebcal/client.go           # Hebcal JSON API
│   ├── chabad/client.go           # chabad.org RSS + candle ICS; normalize→split→classify parser
│   ├── cache/zmanim.go            # ~/.cache/didan/zmanim.json
│   ├── generator/generator.go     # pipeline orchestration + progress output
│   ├── patcher/candle.go          # replace candle/havdalah times; apply tosfos offset
│   ├── attacher/zmanim.go         # attach zmanim to event descriptions
│   ├── fastday/builder.go         # synthesise fast-day begin/end events
│   ├── alarm/builder.go           # rebuild VALARM blocks per policy
│   ├── transliterator/transliterator.go  # Ashkenazi substitution (reads transliterations.json)
│   ├── haftorah/patcher.go        # Chabad haftorah from embedded JSON
│   ├── cleaner/description.go     # strip Hebcal boilerplate from descriptions
│   ├── specialdates/merge.go      # Chabad Yomei d'Pagra; description reformatter
│   ├── icalwriter/writer.go       # RFC 5545 output; buildSummary; eventEmoji
│   └── embeddata/
│       ├── embeddata.go           # go:embed declarations
│       └── files/
│           ├── haftorah_chabad.json     # parsha slug → Chabad haftorah (NEEDS VERIFICATION)
│           ├── yomei_dpagra.json        # Chabad special dates + Chitas summaries
│           ├── transliterations.json    # [[source, target], …] substitution pairs
│           └── rebbes.json              # unified rebbe data: names, honorifics, dates
├── .github/workflows/ci.yml      # GitHub Actions: vet, build, staticcheck, govulncheck,
│                                  #   actionlint, REUSE
├── .githooks/pre-commit           # fmt+restage, vet, staticcheck, actionlint, REUSE
├── docs/
│   ├── rebbes_schema.md           # rebbes.json schema + full DOB/DOD reference tables
│   ├── zmanim_parser_design.md
│   └── zmanim_parser_data-driven_classifier.md
├── Makefile                       # build, fmt, vet, lint, vuln, actionlint, reuse,
│                                  #   check, tidy, hooks, clean
├── go.mod                         # module github.com/toobuntu/zman-didan
├── go.sum                         # dependency lock file — commit this
├── ARCHITECTURE.md
└── README.md
```

## Building and running

```sh
make hooks          # register .githooks/pre-commit (once per clone)
make build          # go build -o bin/didan ./cmd/didan
./bin/didan generate --year 5786 --zip 17601
./bin/didan generate --year 5786 --zip 17601 --lang ah
./bin/didan generate --start 2026-03-16 --end 2026-04-19 --zip 17601 --lang ah
go test ./...       # no tests yet
make check          # fmt + vet + lint + vuln (mirrors CI locally)
```

## CLI flags (current)

```
--year N          Hebrew year (mutually exclusive with --start/--end)
--start YYYY-MM-DD / --end YYYY-MM-DD
--zip NNNNN
--lat N --lon N --tzid TZ --name NAME
--geoname N
--lang            he (default), he-x-NoNikud, a, ah, ah-x-NoNikud, s, sh
--candles N       minutes before shkiah (default 25)
--tosfos N        minutes added to havdala for tosfos Shabbos (default 4)
--emojis          bool, default true — prefix SUMMARY with emoji
--no-clobber      refuse to overwrite existing output file
--output DIR
--refresh         bypass zmanim cache; re-fetch all from chabad.org
```

## Where mappings live

| Mapping | Location |
|---------|----------|
| Transliteration (Sephardic/modern → Ashkenazi) | `internal/embeddata/files/transliterations.json` |
| Rebbe name normalization + birth/death dates | `internal/embeddata/files/rebbes.json` |
| Haftorah assignments | `internal/embeddata/files/haftorah_chabad.json` |
| Yomei d'Pagra summaries | `internal/embeddata/files/yomei_dpagra.json` |
| RSS title → ZmanimDay field | `internal/chabad/client.go` — `classifierRules` slice |
| Alarm policy | `internal/alarm/builder.go` — switch statement |
| Emoji lookup | `internal/icalwriter/writer.go` — `eventEmoji()` |

## Key design decisions

### Data flow

Hebcal JSON → normalise events → fetch Chabad candle ICS (year, always fresh) →
fetch Chabad zmanim (per date, disk-cached) → patch candle/havdalah + tosfos →
attach zmanim to descriptions → build fast-day events → rebuild alarms →
patch haftorahs → apply transliteration → clean descriptions →
fetch + filter + merge Yomei d'Pagra → sort → write iCal

### Cache semantics

- Zmanim: disk-cached at `~/.cache/didan/zmanim.json`, keyed `YYYY-MM-DD|locationID`
- Candle ICS: not cached — fetched on every run
- Yomei d'Pagra ICS: not cached — fetched on every run
- `--refresh` bypasses the zmanim cache entirely

### rebbes.json

Single source of truth for rebbe data. See `docs/rebbes_schema.md` for the full
schema and DOB/DOD reference tables (Hebrew and Gregorian). Fields used at runtime:

- `verbose_names` — matched against raw Hebcal feed strings for name normalization
- `honorific` — replacement name written to calendar output
- `birth_year` — Hebrew year integer; used for "(N years ago)" in birthday descriptions
- `death_year` — Hebrew year integer; used to disambiguate when two figures share a
  verbose name (cross-checked against observance_year − N from Hebcal yahrzeit text)

Fields preserved for reference but not read at runtime:
`birth_date_hebrew`, `birth_date_gregorian`, `death_date_hebrew`, `death_date_gregorian`

### Operation order in specialdates

`reformatDescription` runs inside `convertEvent` — before `normalizeNames`. This
means `rebbes.json` `verbose_names` must match the raw Hebcal feed strings, not the
honorific forms. After reformatting, `normalizeNames` replaces verbose names with
honorifics in both Title and Description.

### RSS parser

chabad.org zmanim titles are not a stable API. The parser uses a
Normalize → Split → Classify pipeline:

1. `normalizeLabel`: insert `|` before uppercase letter after `)` (fixes `")Fast"` etc.)
2. `splitLabel`: split on `" | "`
3. `classifyLabel`: match against `classifierRules` slice (data-driven, first match wins)
   - canonical field → update ZmanimDay struct field
   - event flag → store in `ZmanimDay.Events` map
   - unknown labels default to event-only storage

### Tosfos Shabbos

`patcher.PatchCandles` adds `cfg.Tosfos` minutes to the chabad.org havdalah
time. The event Title is rebuilt to reflect both the corrected time and the
offset: `"Havdala (+4): 8:07 PM"`.

### Language / transliteration modes

| `--lang` | Hebcal `lg=` | Post-processing |
|----------|-------------|-----------------|
| `he` (default) | `he` | none |
| `he-x-NoNikud` | `he-x-NoNikud` | none |
| `a` | `a` | Ashkenazi table |
| `ah` | `ah` | Ashkenazi table |
| `ah-x-NoNikud` | `ah` | Ashkenazi table + strip U+05B0–U+05C7 |
| `s` | `s` | none (Hebcal provides Sefardi) |
| `sh` | `sh` | none (Hebcal provides Sefardi + Hebrew) |

### Alarm policy

| Category | VALARM triggers |
|----------|----------------|
| `candles` | −2h, −15min |
| `havdalah` | −10min, PT0S |
| `fast-begin` | −2h, −30min |
| `fast-end` | −15min, PT0S |
| all-day / others | none |

All VALARM DESCRIPTION values use `"Event reminder"` — Apple Calendar does not
display VALARM DESCRIPTION in its UI.

### VTIMEZONE

Static block for `America/New_York` only. Phase 2: generate from tzdata for
non-Eastern locations.

## Data sources

| Source | Used for |
|--------|----------|
| `hebcal.com/hebcal?cfg=json` | Calendar structure, parsha, leyning |
| `chabad.org/tools/rss/zmanim.xml?tdate=YYYY-MM-DD` | All zmanim per date (cached) |
| `chabad.org/calendar/candlelighting/candlelighting.ics.asp` | Candle + havdalah (always fetched) |
| `download.hebcal.com/ical/chabad-special-dates.ics` | Yomei d'Pagra (always fetched) |
| Embedded JSON | Haftorahs, transliterations, rebbe data, Chitas summaries |

## Pending / Phase 2

- [ ] Verify `haftorah_chabad.json` against chabad.org/library/article_cdo/aid/4158333
- [ ] VTIMEZONE generation from tzdata for non-Eastern timezones
- [ ] Cache candle ICS and special dates to reduce network calls on repeat runs
- [ ] German output (`--lang de`)
- [ ] Parsha summaries in descriptions
- [ ] SwiftUI GUI wrapper
- [ ] Contact chabad.org about RSS feed usage terms
