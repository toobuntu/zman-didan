# didan

Generates a Chabad-minhag iCalendar (.ics) file for a given Hebrew year or
date range and location. Replaces Hebcal's placeholder zmanim with
Chabad-authoritative times from chabad.org and applies Ashkenazi
transliterations throughout.

## What it does

- Fetches the Hebrew calendar structure from the [Hebcal API](https://www.hebcal.com/home/developer-apis)
- Replaces candle lighting and havdalah times with Chabad times (Alter Rebbe zmanim)
- Applies tosfos Shabbos to havdalah (configurable offset, default +4 min)
- Appends zmanim to event descriptions: shkiah/tzeis for candle lighting, misheyakir/latest shema for havdalah, chatzos halaila for Pesach seder night, chatzos hayom for Tisha B'Av, menora lighting windows for Chanuka
- Synthesises timed fast-begin/end events with alarms
- Applies Ashkenazi transliterations (Succos, Shavuos, Bereishis, Yirmiyahu, etc.)
- Substitutes Chabad haftorah readings
- Merges Chabad Yomei d'Pagra with concise reformatted descriptions and "(N years ago)" annotations

## Install

```sh
git clone https://github.com/toobuntu/didan
cd didan
make hooks          # register pre-commit hook (requires staticcheck)
make build          # produces bin/didan
```

Requires Go 1.22 or later. Optional tools: `staticcheck`, `govulncheck`.

```sh
# Install optional lint/security tools
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## Usage

```sh
# Full Hebrew year by ZIP code
bin/didan generate --year 5786 --zip 17601

# Full Hebrew year by coordinates
bin/didan generate --year 5786 --lat 40.0732 --lon -76.3209 --tzid America/New_York --name Lancaster

# Full Hebrew year by GeoNames ID
bin/didan generate --year 5786 --geoname 5197079

# Date range (e.g. while traveling)
bin/didan generate --start 2025-11-01 --end 2025-11-30 --zip 17601

# Ashkenazi transliteration + Hebrew, emojis on (the default), custom output
bin/didan generate --year 5786 --zip 17601 --lang ah --output ~/Desktop

# Disable emojis; add extra tosfos Shabbos
bin/didan generate --year 5786 --zip 17601 --emojis=false --tosfos 5
```

### Flags

**Date range** — one required, mutually exclusive:

| Flag | Default | Description |
|------|---------|-------------|
| `--year N` | — | Hebrew year (e.g. `5786`) |
| `--start YYYY-MM-DD` | — | Start of date range; requires `--end` |
| `--end YYYY-MM-DD` | — | End of date range; requires `--start` |

**Location** — one required, mutually exclusive:

| Flag | Description |
|------|-------------|
| `--zip NNNNN` | US ZIP code |
| `--lat N --lon N --tzid TZ --name NAME` | Decimal lat/lon + [IANA timezone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) + display name |
| `--geoname N` | [GeoNames.org](https://www.geonames.org/) numeric ID |

**Options:**

| Flag | Default | Description |
|------|---------|-------------|
| `--lang` | `he` | Language/transliteration mode (see below) |
| `--candles` | `25` | Minutes before shkiah for candle lighting |
| `--tosfos` | `4` | Minutes added to havdalah (tosfos Shabbos/Yom Tov); `0` to disable |
| `--emojis` | `true` | Prefix SUMMARY with emoji (🕯️, 🍏🍯, etc.) |
| `--output` | `.` | Output directory |
| `--refresh` | `false` | Bypass zmanim cache; re-fetch everything from the network |
| `--no-clobber` | `false` | Refuse to overwrite an existing output file |

### Language modes

| Value | Meaning |
|-------|---------|
| `he` | Hebrew with nikud (default) |
| `he-x-NoNikud` | Hebrew without nikud |
| `a` | Ashkenazi transliteration only |
| `ah` | Ashkenazi transliteration + Hebrew with nikud |
| `ah-x-NoNikud` | Ashkenazi transliteration + Hebrew without nikud |
| `s` | Sefardi transliteration only |
| `sh` | Sefardi transliteration + Hebrew |

### IANA timezone identifiers

For `--tzid`, use standard IANA timezone names: `America/New_York`,
`Europe/Berlin`, `Asia/Jerusalem`, `Australia/Sydney`, etc.
Full list: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones

### Output filenames

| Mode | Filename |
|------|----------|
| Year + ZIP | `didan_5786_17601.ics` |
| Year + lat/lon | `didan_5786_40.0732_-76.3209.ics` |
| Range + ZIP | `didan_20251101_20251130_17601.ics` |

## Development

```sh
make vet        # go vet ./...
make lint       # staticcheck ./...
make vuln       # govulncheck ./...
make fmt        # gofmt -w ./...
make tidy       # go mod tidy
make clean      # remove bin/didan
```

**CI** — GitHub Actions runs vet, build, staticcheck, and govulncheck on push.
Setup (once per clone):

```sh
mkdir -p .github/workflows
cp docs/ci.yml .github/workflows/ci.yml
git add .github
```

## Zmanim

All times follow the Alter Rebbe's calculations as published by chabad.org.
Candle lighting defaults to 25 minutes before shkiah per the Rebbe's practice.
Havdalah is offset by `--tosfos` minutes (default 4) for tosfos Shabbos.

## Cache

Zmanim are cached at `~/.cache/didan/zmanim.json`, keyed by date and location
ID. Past-date entries are pruned on each run. Use `--refresh` to force a full
network re-fetch. Candle lighting times and Yomei d'Pagra are always fetched
fresh.

## Static data files

All in `internal/embeddata/files/` — edit and rebuild to update:

| File | Contents |
|------|----------|
| `haftorah_chabad.json` | Chabad haftorah by parsha slug — **needs halachic verification** |
| `yomei_dpagra.json` | Chabad special dates with Chitas summaries |
| `transliterations.json` | Sephardic/modern → Ashkenazi substitution pairs |
| `rebbe_names.json` | Verbose rebbe name forms → standard honorifics |
| `birthday_years.json` | Birth Hebrew years for Chabad figures (enables "Nth Birthday" descriptions) |

## Notes

- `haftorah_chabad.json` requires verification against
  https://www.chabad.org/library/article_cdo/aid/4158333 before distributing.
- Uses the chabad.org zmanim RSS feed. Per their
  [RSS terms](https://www.chabad.org/library/article_cdo/aid/298447), contact
  them before building a distributed application.
- Geolocation via `--lat/--lon` uses undocumented chabad.org API parameters
  inferred from web UI URL patterns.

## License

Personal use. See chabad.org data notes above.
