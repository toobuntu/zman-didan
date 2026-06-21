<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

# didan

A CLI tool that generates a Chabad-minhag Jewish calendar as an iCalendar (.ics)
file. Fetches Hebrew calendar data from Hebcal, replaces all zmanim with
Chabad-authoritative times from chabad.org, applies Ashkenazi transliterations,
substitutes Chabad haftorah readings, adds Chabad Yomei d'Pagra, and enriches
event descriptions with additional zmanim.

> System design, the pipeline diagram, and the chabad.org / Hebcal data
> contracts live in [`docs/architecture.md`](docs/architecture.md). Planned work
> is in [`docs/roadmap.md`](docs/roadmap.md); known debt in
> [`docs/technical-debt.md`](docs/technical-debt.md). This file is the working
> reference for building in the repo.

## Module and binary

- **Module**: `github.com/toobuntu/zman-didan`
- **Binary**: `bin/didan` (built with `make build`)

## Project layout

```
didan/
├── cmd/didan/main.go              # cobra CLI — all flags, validation
├── internal/
│   ├── types/types.go             # shared structs (Config, HebcalEvent, ZmanimDay, …)
│   ├── hebcal/client.go           # Hebcal JSON API; HTTPCache; nikud enrichment
│   ├── chabad/client.go           # chabad.org RSS + candle ICS; normalize→split→classify
│   ├── cache/
│   │   ├── zmanim.go              # ~/.cache/didan/zmanim.json (30-day retention)
│   │   └── httpcache.go           # ~/.cache/didan/http/ (SHA-256 URL key, 7-day TTL)
│   ├── generator/generator.go     # pipeline orchestration + [source] progress output
│   ├── patcher/candle.go          # replace candle/havdalah times; apply tosfos offset
│   ├── attacher/zmanim.go         # attach zmanim to event descriptions
│   ├── fastday/builder.go         # synthesize fast-day begin/end events
│   ├── alarm/builder.go           # rebuild VALARM blocks per policy
│   ├── transliterator/transliterator.go  # Ashkenazi substitution (title_only + all scopes)
│   ├── haftorah/patcher.go        # Chabad haftorah: API field preferred, JSON fallback
│   ├── cleaner/description.go     # strip Hebcal boilerplate from descriptions
│   ├── specialdates/merge.go      # Chabad Yomei d'Pagra; description reformatter
│   ├── icalwriter/writer.go       # RFC 5545 output; buildSummary; LOCATION; eventEmoji
│   └── embeddata/
│       ├── embeddata.go           # go:embed declarations
│       └── files/
│           ├── haftorah_chabad.json     # parsha slug → Chabad haftorah (fallback; verify)
│           ├── yomei_dpagra.json        # Chabad special dates + Chitas summaries
│           ├── transliterations.json    # {title_only: [...], all: [...]} substitution pairs
│           └── rebbes.json              # unified rebbe data: names, honorifics, dates
├── .github/workflows/ci.yml
├── .githooks/pre-commit
├── docs/
│   ├── architecture.md
│   ├── roadmap.md
│   ├── technical-debt.md
│   ├── rebbes_schema.md
│   ├── zmanim_parser_design.md
│   └── zmanim_parser_data-driven_classifier.md
├── .vale.ini                      # en_US prose linter (Vale.Spelling)
├── AGENTS.md                      # this guide (CLAUDE.md symlinks to it)
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

## Building and running

```sh
make hooks          # register .githooks/pre-commit (once per clone)
make build          # go build -o bin/didan ./cmd/didan
./bin/didan generate --year 5786 --zip 17601
./bin/didan generate --year 5786 --zip 17601 --lang ah
./bin/didan generate --start 2026-03-16 --end 2026-04-19 --zip 17601 --lang ah
make test           # go test -count=1 ./...
make integration    # network tests (go test -tags integration ./internal/integration/)
make check          # style + scan + test (mirrors CI)
```

## Tool installation

```sh
brew install staticcheck govulncheck actionlint reuse
```

## Make targets

| Target | What it does |
|--------|-------------|
| `make build` | Compile `bin/didan` |
| `make install` | `go install ./cmd/didan` (to `$(go env GOBIN)` / `$GOPATH/bin`) |
| `make dev` | Install dev tooling (staticcheck, govulncheck, actionlint) |
| `make test` | `go test -count=1 ./...` (bypasses Go test result cache) |
| `make integration` | Network tests with build tag `integration` |
| `make check` | `style` + `scan` + `test` (full local suite) |
| `make fmt` | `gofmt -w` all Go files |
| `make style` | Format check + `go vet` + `staticcheck` |
| `make scan` | `govulncheck` |
| `make vale` | en_US prose lint of Markdown docs |
| `make actionlint` | Lint `.github/workflows/ci.yml` |
| `make reuse` | REUSE license compliance check |
| `make tidy` | `go mod tidy` |
| `make hooks` | Register `.githooks/pre-commit` |
| `make clean` | Remove `bin/didan` |

## CLI flags

```
--year N              Hebrew year (mutually exclusive with --start/--end)
--start YYYY-MM-DD / --end YYYY-MM-DD
--zip NNNNN
--lat N --lon N --tzid TZ --name NAME
--geoname N
--lang                h (default), hn, a, ah, ahn, s, sh, shn
--candles N           minutes before shkiah for candle lighting (default 25)
--tosfos N            minutes added to havdala for tosfos Shabbos (default 4)
--emojis              bool, default true — prefix SUMMARY with emoji
--no-clobber          refuse to overwrite existing output file
--output DIR
--refresh             bypass all caches; re-download everything
```

## Language modes

| `--lang` | Hebcal `lg=` | Post-processing |
|----------|-------------|-----------------|
| `h` (default) | `he` | none |
| `hn` | `he-x-NoNikud` | none |
| `a` | `a` | Ashkenazi table |
| `ah` | `ah` + `he` (nikud enrichment) | Ashkenazi table |
| `ahn` | `ah` + `he` (nikud enrichment) | Ashkenazi table + strip nikud client-side |
| `s` | `s` | none |
| `sh` | `sh` | none |
| `shn` | `sh` | strip nikud client-side |

For `ah`/`ahn` modes, the Hebcal `lg=ah` response does not include nikud in
the `hebrew` field. A second Hebcal request with `lg=he` is made to replace
`ev.Hebrew` with nikud-bearing text. Both requests are cached via HTTPCache.

Note: `lg=a` (Ashkenazi transliteration of event titles) is unrelated to the
`ashkenazi_standard` leyning field in the API response. The former controls
title text; the latter is an alternative haftorah string in the leyning object.

## Hebcal API leyning fields (items[].leyning)

| Field | Populated when | Notes |
|-------|---------------|-------|
| `haftarah` | always | Ashkenazi standard |
| `haftarah_chabad` | Chabad differs from Ashkenazi | Live as of 2026-03-30 |
| `haftarah_sephardic` | Sephardic differs from Ashkenazi | Closest to Chabad for most parshas, but not a valid Chabad fallback — when `haftarah_chabad` is null, Chabad follows `haftarah`, not `haftarah_sephardic` |
| `ashkenazi_standard` | same as `haftarah` | Redundant |
| `ashkenazi_litvish` | Lithuanian variant differs | Potentially closest to Chabad Yiddish pronunciation; unverified |

Go's default HTTP transport adds `Accept-Encoding: gzip` automatically and
decompresses transparently — no explicit header needed in the client code.

## Cache semantics

All caches live under `~/.cache/didan/`.

| Cache | File | Key | Eviction |
|-------|------|-----|----------|
| Zmanim | `zmanim.json` | `YYYY-MM-DD\|locationID` | entries > 30 days old |
| HTTP (Hebcal, candle ICS, special dates) | `http/<sha256(url)>` | full URL | file mtime > 7 days |

Zmanim data is deterministic for a given date and location — it never changes.
The 30-day retention window exists only to bound storage size, not for freshness.
`Prune()` is called once per run at startup and only writes if entries changed.
`Set()` does not call `Prune()` internally.

`zmanim.json` is an envelope `{version, entries}` carrying `zmanimCacheVersion`;
a version mismatch on load discards the cache and forces a re-fetch. Bump it
whenever parser logic or the stored field set changes (e.g. the Shabbos/Yom-Tov
Misheyakir label-variant fix).

## Where mappings live

| Mapping | Location |
|---------|----------|
| Transliterations (title_only scope) | `internal/embeddata/files/transliterations.json` — `title_only` array |
| Transliterations (all scope) | `internal/embeddata/files/transliterations.json` — `all` array |
| Rebbe name normalization + birth/histalkus dates | `internal/embeddata/files/rebbes.json` |
| Haftorah assignments (fallback) | `internal/embeddata/files/haftorah_chabad.json` |
| Yomei d'Pagra summaries | `internal/embeddata/files/yomei_dpagra.json` |
| RSS title → ZmanimDay field | `internal/chabad/client.go` — `classifierRules` slice |
| Alarm policy | `internal/alarm/builder.go` — switch statement |
| Emoji lookup | `internal/icalwriter/writer.go` — `eventEmoji()` |

## Data flow

```
Hebcal JSON (lg=X)
  [ah/ahn only] + Hebcal JSON (lg=he) → replace ev.Hebrew with nikud
→ normalize events
→ drop Hebcal timed fast events (replaced by fastday.Build)
→ Chabad candle ICS (year, HTTPCache)
→ Chabad zmanim RSS (per date, ZmanimCache)
→ PatchCandles: replace times + tosfos offset
→ AttachZmanim: enrich descriptions with zmanim
→ fastday.Build: synthesize fast-begin/end events
→ alarm.Rebuild: set VALARM blocks per policy
→ haftorah.Patch: prefer ev.Leyning.HaftarahChabad (API); fall back to JSON
→ transliterator.Apply: Ashkenazi substitutions (a/ah/ahn only)
→ cleaner.Clean: strip Hebcal boilerplate
→ specialdates.Merge: Yomei d'Pagra (HTTPCache)
→ sort by date
→ icalwriter.Write: RFC 5545 output
```

## Key design decisions

### Haftorah source priority

`haftorah.Patch` applies Chabad haftorahs in this order:
1. `ev.Leyning.HaftarahChabad` — from Hebcal API `haftarah_chabad` field, live
   as of 2026-03-30. Non-null only when Chabad differs from Ashkenazi standard.
   When null, `ev.Leyning.Haftarah` (Ashkenazi standard) is already correct.
2. `haftorah_chabad.json` — embedded fallback table. Retained until a full-year
   comparison confirms the API covers all cases. Known discrepancy: Tzav is
   `7:21-8:3, 9:22-23` in the JSON but `7:21-28, 9:22-23` from the API;
   the API value is authoritative.

### rebbes.json fields

| Field | Used for |
|-------|---------|
| `verbose_names` | String replacement in `normalizeNames`; lookup key in `findByVerboseName` |
| `honorific` | Output name in calendar events |
| `huledes_year` | "(N years ago)" in birthday descriptions |
| `huledes_gregorian` | Gregorian DOB shown in birthday descriptions |
| `histalkus_year` | Disambiguates shared verbose names; "(N years ago)" in histalkus descriptions |
| `histalkus_gregorian` | Gregorian DOD shown in histalkus descriptions |

### transliterations.json scopes

`title_only`: biblical book names (Genesis → Bereishis, etc.). Applied to
`Title` and `Memo` only — **not** `Description`. These words appear in
Description as English prose (e.g. "commemorates the Exodus from Egypt") where
the substitution would be wrong.

`all`: holiday and parsha names. Applied to `Title`, `Description`, and `Memo`.

### Operation order in specialdates

`reformatDescription` runs before `normalizeNames`, so `verbose_names` must
match raw Hebcal feed strings exactly. The feed uses U+2019 RIGHT SINGLE
QUOTATION MARK in SUMMARY strings; the regex uses `[\u2019']` to match both forms.

`birthdayRe` matches all "X occurs on..." boilerplate, not just birthdays.
`reformatDescription` checks `strings.HasPrefix(event, "Birthday of ")` before
applying birthday-specific formatting; non-birthday events return "EVENT on DAY HMONTH YEAR".

### Alarm policy

| Category | VALARM triggers |
|----------|----------------|
| `candles` | −2h, −15min |
| `havdalah` | −10min, PT0S |
| `fast-begin` | −2h, −30min |
| `fast-end` | −15min, PT0S |
| all-day / others | none |

VALARM DESCRIPTION is always `"Event reminder"`.

### Candle lighting title forms

| chabad.org label | `IsYomTov` | `AfterHavdala` | SUMMARY |
|-----------------|-----------|----------------|---------|
| Light Shabbat Candles / Light Candles | false | false | `Candle lighting: 6:52 PM` |
| Light Holiday Candles | true | false | `YT candles: 7:05 PM` |
| Light Holiday Candles after | true | true | `YT candles after: 8:12 PM` |

Havdala SUMMARY always starts with English — LTR time must remain readable.

### VTIMEZONE

Static block for `America/New_York` only. Phase 2: generate from tzdata.

## Data sources

| Source | Used for |
|--------|----------|
| `hebcal.com/hebcal?cfg=json` | Calendar structure, parsha, leyning, Hebrew text |
| `chabad.org/tools/rss/zmanim.xml?tdate=YYYY-MM-DD` | All zmanim per date (ZmanimCache) |
| `chabad.org/calendar/candlelighting/candlelighting.ics.asp` | Candle + havdalah (HTTPCache) |
| `download.hebcal.com/ical/chabad-special-dates.ics` | Yomei d'Pagra (HTTPCache) |
| Embedded JSON | Haftorahs (fallback), transliterations, rebbe data, Chitas summaries |

## Pending / Phase 2

- [ ] **Haftorah verification**: Run a full Hebrew year through the pipeline and
      compare `ev.Leyning.HaftarahChabad` API values against `haftorah_chabad.json`.
      Verify discrepancies against https://www.chabad.org/library/article_cdo/aid/4158333.
      Known discrepancy: Tzav — API returns `Jeremiah 7:21-28, 9:22-23`; embedded
      JSON has `7:21-8:3, 9:22-23`. Once verified, delete `haftorah_chabad.json`,
      `loadTable()`, and the embedded JSON fallback path in `haftorah.Patch`.

- [ ] **`ashkenazi_litvish` investigation**: The Hebcal API exposes `lg=ashkenazi_litvish`
      and a `leyning.ashkenazi_litvish` haftorah field. Lithuanian Ashkenazi has
      geographic overlap with Chabad's origins. Verify whether this `lg=` value
      produces event title transliterations closer to Chabad Yiddish than `lg=ah`.
      Requires comparing against authoritative Chabad pronunciation sources.

- [ ] **Additional Hebcal API features to evaluate**:
      - `o=on` — Days of the Omer (Sefirat HaOmer, highly relevant for Chabad)
      - `mvch=on` — Shabbat Mevarchim (Shabbos before Rosh Chodesh)
      - `molad=on` — Molad announcement
      - `D=on` / `d=on` — Hebrew date overlay on all events
      - `yzkr=on` — Yizkor dates

- [ ] **anash.org zmanim source** (resolved — no action needed):
      anash.org uses WordPress AJAX (`admin-ajax.php?action=get_zmanim`).
      PHP backend proxies to chabad.org server-side. No private API found.

- [ ] VTIMEZONE generation from tzdata for non-Eastern timezones
- [ ] German output (`--lang de`)
- [ ] Parsha summaries in descriptions
- [ ] SwiftUI GUI wrapper
- [ ] Contact chabad.org about RSS feed usage terms
- [ ] Add `make test` to pre-commit hook and CI
