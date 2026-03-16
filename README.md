# Didan

Generates a Chabad-minhag iCalendar (.ics) file for a given Hebrew year and
location, replacing Hebcal's placeholder zmanim with Chabad-authoritative times
from chabad.org and applying Ashkenazi transliterations throughout.

## What it does

- Fetches the Hebrew calendar structure from the [Hebcal API](https://www.hebcal.com/home/developer-apis)
- Replaces candle lighting and havdalah times with times from chabad.org (Alter Rebbe zmanim, configurable offset before shkiah)
- Appends additional zmanim to event descriptions: shkiah and tzeis for Shabbos/Yom Tov, misheyakir and latest shema for havdalah events, chatzos halaila for Pesach seder night, chatzos hayom for Tisha B'Av, and menora lighting windows for Chanuka
- Synthesises standalone timed events for fast-day start and end times with alarms
- Applies Ashkenazi transliterations (Succos, Shavuos, Bereishis, Yirmiyahu, etc.)
- Substitutes Chabad haftorah readings where they differ from standard Ashkenazi/Sephardic assignments
- Merges Chabad Yomei d'Pagra (special Chassidic dates) from the Hebcal special dates feed

## Install

```sh
git clone https://github.com/toobuntu/didan
cd didan
go build -o didan ./cmd/didan
```

Requires Go 1.22 or later.

## Usage

```sh
# Full Hebrew year by ZIP code
didan generate --year 5786 --zip 17601

# Full Hebrew year by coordinates
didan generate --year 5786 --lat 40.0732 --lon -76.3209 --tzid America/New_York --name Lancaster

# Full Hebrew year by GeoNames ID
didan generate --year 5786 --geoname 5197079

# Date range (e.g. while traveling)
didan generate --start 2025-11-01 --end 2025-11-30 --zip 17601

# With options
didan generate --year 5786 --zip 17601 --lang ah --candles 25 --output ~/Desktop
```

### Flags

**Date range** — one required, mutually exclusive:

| Flag | Description |
|------|-------------|
| `--year N` | Hebrew year (e.g. `5786`) |
| `--start YYYY-MM-DD` | Start of explicit date range; requires `--end` |
| `--end YYYY-MM-DD` | End of explicit date range; requires `--start` |

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
| `--output` | `.` | Output directory |
| `--refresh` | false | Bypass zmanim cache and re-fetch |

### Language modes

| Value | Meaning |
|-------|---------|
| `he` | Hebrew with nikud (default) |
| `he-x-NoNikud` | Hebrew without nikud |
| `a` | Ashkenazi transliteration only |
| `ah` | Ashkenazi transliteration + Hebrew with nikud |
| `ah-x-NoNikud` | Ashkenazi transliteration + Hebrew without nikud |

### IANA timezone identifiers

For `--tzid`, use IANA timezone names such as `America/New_York`, `Europe/Berlin`,
`Asia/Jerusalem`, `Australia/Sydney`. A full list is at
https://en.wikipedia.org/wiki/List_of_tz_database_time_zones.

### Output filenames

| Mode | Filename |
|------|----------|
| Year + ZIP | `didan_5786_17601.ics` |
| Year + lat/lon | `didan_5786_40.0732_-76.3209.ics` |
| Range + ZIP | `didan_20251101_20251130_17601.ics` |

## Zmanim

All times come from chabad.org and follow the Alter Rebbe's (Baal HaTanya's)
calculations. Candle lighting defaults to 25 minutes before shkiah, consistent
with the Rebbe's practice.

## Cache

Zmanim are cached at `~/.cache/didan/zmanim.json` keyed by date and location ID.
Past-date entries are pruned automatically on each run. Use `--refresh` to
force a full re-fetch.

## Notes

- `internal/embeddata/files/haftorah_chabad.json` contains Chabad haftorah
  assignments that require verification against
  https://www.chabad.org/library/article_cdo/aid/4158333 before distributing
  generated calendars to others.
- This tool uses the chabad.org zmanim RSS feed. Per their
  [RSS terms](https://www.chabad.org/library/article_cdo/aid/298447), contact
  them before incorporating the feed into a distributed application.
- Geolocation via `--lat/--lon` uses undocumented chabad.org API parameters
  inferred from web UI URL patterns.

## License

Personal use. See notes above regarding chabad.org data.
