# didan

Generates a Chabad-minhag iCalendar (`.ics`) file for a Hebrew year or date
range and a location. It takes the Hebrew calendar structure from the Hebcal
API, replaces the placeholder zmanim with Chabad-authoritative times from
chabad.org, and enriches the result with Chabad readings, special dates, fast
times, and (optionally) Ashkenazi transliterations.

## What it does

- Fetches the calendar structure from the [Hebcal API](https://www.hebcal.com/home/developer-apis).
- Replaces candle lighting and havdalah with Chabad (Alter Rebbe) times; applies
  a configurable tosfos-Shabbos offset to havdalah (default +4 min).
- Enriches descriptions with zmanim: shkiah/tzeis for candle lighting, the Shma
  window (misheyakir–latest shema) for havdalah, chatzos halaila for the Pesach
  seder, chatzos hayom for Tisha B'Av, and menora-lighting windows for Chanuka.
- Synthesises timed fast begin/end events with alarms.
- Prefers the Hebcal API's Chabad haftarah, falling back to an embedded table.
- Merges Chabad Yomei d'Pagra with reformatted descriptions.
- Optionally applies Ashkenazi transliterations (Succos, Shavuos, Bereishis, …).

## Install

```sh
git clone https://github.com/toobuntu/zman-didan
cd zman-didan
make hooks    # register the pre-commit hook (once per clone)
make build    # produces bin/didan
make install  # optional: install to $(go env GOBIN) / $GOPATH/bin
```

Requires a recent Go toolchain (see `go.mod` for the minimum version).
For the lint/security tooling used by `make check`, run `make dev` (or install
`staticcheck` and `govulncheck` via Homebrew).

## Usage

```sh
# Full Hebrew year by ZIP
bin/didan generate --year 5786 --zip 17601

# By coordinates / by GeoNames ID
bin/didan generate --year 5786 --lat 40.0732 --lon -76.3209 --tzid America/New_York --name Lancaster
bin/didan generate --year 5786 --geoname 5197079

# Date range, Ashkenazi + Hebrew, custom output
bin/didan generate --start 2026-03-16 --end 2026-04-19 --zip 17601 --lang ah --output ~/Desktop

# No emojis; extra tosfos
bin/didan generate --year 5786 --zip 17601 --emojis=false --tosfos 5
```

### Flags

One date selector is required (mutually exclusive): `--year N` **or**
`--start YYYY-MM-DD` + `--end YYYY-MM-DD`.

One location is required (mutually exclusive): `--zip NNNNN`,
`--lat N --lon N --tzid TZ --name NAME`, or `--geoname N`.

| Flag | Default | Description |
|------|---------|-------------|
| `--lang` | `h` | Language/transliteration mode (see below) |
| `--candles` | `25` | Minutes before shkiah for candle lighting |
| `--tosfos` | `4` | Minutes added to havdalah; `0` to disable |
| `--emojis` | `true` | Prefix SUMMARY with emoji |
| `--output` | `.` | Output directory |
| `--refresh` | `false` | Bypass all caches; re-download everything |
| `--no-clobber` | `false` | Refuse to overwrite an existing output file |

### Language modes

| `--lang` | Meaning |
|----------|---------|
| `h` (default) | Hebrew with nikud |
| `hn` | Hebrew, no nikud |
| `a` | Ashkenazi transliteration only |
| `ah` | Ashkenazi + Hebrew (with nikud) |
| `ahn` | Ashkenazi + Hebrew, no nikud |
| `s` | Sefardi |
| `sh` | Sefardi + Hebrew |
| `shn` | Sefardi + Hebrew, no nikud |

The mapping to Hebcal's `lg=` parameter and the nikud-enrichment behavior for
`ah`/`ahn` are documented in [`CLAUDE.md`](CLAUDE.md#language-modes).

### Output filenames

| Mode | Filename |
|------|----------|
| Year + ZIP | `didan_5786_17601.ics` |
| Year + lat/lon | `didan_5786_40.0732_-76.3209.ics` |
| Range + ZIP | `didan_20260316_20260419_17601.ics` |

### IANA timezone identifiers

For `--tzid`, use standard IANA names: `America/New_York`, `Europe/Berlin`,
`Asia/Jerusalem`, `Australia/Sydney`, etc. Full list:
<https://en.wikipedia.org/wiki/List_of_tz_database_time_zones>

## Documentation

| Doc | Contents |
|-----|----------|
| [`CLAUDE.md`](CLAUDE.md) | Working in the repo: layout, build/test/make targets, CLI, language modes, cache semantics, where each mapping lives |
| [`docs/architecture.md`](docs/architecture.md) | System design, pipeline, and the chabad.org / Hebcal data contracts |
| [`docs/roadmap.md`](docs/roadmap.md) | Planned work (versioning, VTIMEZONE, haftorah verification, …) |
| [`docs/technical-debt.md`](docs/technical-debt.md) | Prioritized debt register |

## Caching

Responses are cached under `~/.cache/didan/`: per-date zmanim in `zmanim.json`
(deterministic, 30-day retention, schema-versioned) and raw HTTP bodies in
`http/` (7-day TTL). `--refresh` bypasses everything. See
[`CLAUDE.md`](CLAUDE.md#cache-semantics) for details.

## Data and licensing notes

- All zmanim follow the Alter Rebbe's calculations as published by chabad.org;
  candle lighting defaults to 25 minutes before shkiah.
- The embedded `haftorah_chabad.json` fallback still needs halachic verification
  against <https://www.chabad.org/library/article_cdo/aid/4158333>.
- The zmanim RSS feed is used per chabad.org's
  [RSS terms](https://www.chabad.org/library/article_cdo/aid/298447); contact
  them before building a distributed application.
- `--lat/--lon` uses undocumented chabad.org API parameters inferred from web UI
  URL patterns.

## License

Personal use. See the data notes above.
