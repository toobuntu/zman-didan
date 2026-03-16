# Architecture

## Module

`github.com/toobuntu/didan`

## Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│  CLI input (cobra)                                          │
│  --year 5786 --zip 17601 --lang ah --candles 25             │
└────────────────────────┬────────────────────────────────────┘
                         │  types.Config
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/hebcal  Client.FetchYear()                        │
│  GET hebcal.com/hebcal?cfg=json&year=5786&yt=H&...          │
│  → []types.HebcalEvent, types.Location                      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/chabad  Client.FetchCandlesYear()                 │
│  GET chabad.org/.../candlelighting.ics.asp?weeks=52         │
│  → map[time.Time]CandleDay  (candles + havdalah by date)    │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/chabad  Client.FetchZmanimInZone()  × N dates     │
│  GET chabad.org/tools/rss/zmanim.xml?tdate=YYYY-MM-DD       │
│  cache hit → return cached ZmanimDay                        │
│  cache miss → fetch → parse RSS → cache → return            │
│  → map[string]types.ZmanimDay  (keyed by "YYYY-MM-DD")      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/patcher  PatchCandles()                           │
│  For each candles event: replace Date from CandleDay        │
│  For each havdalah event: replace Date from CandleDay       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/attacher  AttachZmanim()                          │
│  candles → append shkiah, tzeis to Description              │
│  havdalah → append misheyakir, latest shema                 │
│  Pesach night 1 → append chatzos halaila                    │
│  Tisha B'Av → append chatzos hayom                          │
│  Chanuka nights → append menora lighting note               │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/fastday  Build()                                  │
│  For each fast day: synthesise 2 new timed events           │
│    Fast Begins: alos (or prev-day shkiah for TB/YK)         │
│    Fast Ends:   tzeis                                        │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/alarm  Rebuild()                                  │
│  Clear all Alarms slices; re-populate per policy table      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/haftorah  Patch()                                 │
│  Load embedded haftorah_chabad.json                         │
│  For parashat events: replace haftorah line in Description  │
│  and update Leyning.Haftarah                                │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/transliterator  Apply()                           │
│  Longest-match-first substitution on Title + Description    │
│  If --lang ah-x-NoNikud: strip U+05B0–U+05C7               │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/cleaner  Clean()                                  │
│  Remove "Also spelled…", redundant subtitles, JPS lines,    │
│  collapse blank lines                                       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/specialdates  Merge()                             │
│  Fetch chabad-special-dates.ics                             │
│  Apply same transliteration + description passes            │
│  Add Chitas summaries from embedded yomei_dpagra.json       │
│  Return new events for append + sort                        │
└────────────────────────┬────────────────────────────────────┘
                         │  sort all events by Date
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/icalwriter  Write()                               │
│  RFC 5545 output with CRLF line endings, 75-octet folding   │
│  VTIMEZONE block (static for America/New_York; Phase 2 for  │
│  other zones)                                               │
│  Timed events: DTSTART;TZID=<tzid>:...  (not UTC)          │
│  Output: didan_5786_17601.ics                               │
└─────────────────────────────────────────────────────────────┘
```

## Key Data Structures

### types.HebcalEvent
Normalised from Hebcal JSON, mutable throughout the pipeline. `AllDay bool`
distinguishes date-only events (DTSTART;VALUE=DATE) from timed ones. `Slug`
is extracted from the Hebcal API link URL for haftorah table lookup.
`Alarms []Alarm` is populated by the alarm package and serialised by icalwriter.

### types.ZmanimDay
All `time.Time` fields are in the location's IANA timezone, sourced verbatim
from the chabad.org RSS feed. No times are computed locally.

`ShaahZmanitMin` is a float64 duration in decimal minutes (60:30 → 60.5).

`ChatzosHalaila` — the halachic midnight of the night D→D+1 — is reported in
the RSS feed for day D as an AM time, but physically occurs on day D+1.
`parseLocalTime` detects this (AM hour for the chatzos_halaila field) and
constructs the `time.Time` value on D+1.

### cache.ZmanimCache
JSON file at `~/.cache/didan/zmanim.json`. Map keys: `"YYYY-MM-DD|locationID"`.
Serialises `time.Time` as RFC3339. Prunes entries where date < today on every
write. Single-threaded; no locking.

## Event Identification

| Event | How identified |
|-------|----------------|
| Candle lighting | `Category == "candles"` |
| Havdalah | `Category == "havdalah"` |
| Fast days | `Subcat == "fast"` |
| Tisha B'Av / Yom Kippur | above + title contains "Tisha B'Av" / "Yom Kippur" |
| Pesach seder night | `Category == "holiday"`, `HDate` contains "15 Nisan" |
| Chanuka | `Category == "holiday"`, title contains "chanuk" (case-insensitive) |

## Chabad RSS

URL: `https://www.chabad.org/tools/rss/zmanim.xml?locationId=ZIP&locationType=2&bDef=0&before=N&tdate=YYYY-MM-DD`

- Returns today's zmanim by default; `tdate` overrides to any date.
- `tdate` uses `YYYY-MM-DD` format for the RSS endpoint (no slash-encoding issue).
- The `<guid>` field encodes feed *generation* time — ignored entirely.
- Field lookup uses prefix matching on `<title>` text; feed reordering is safe.

### RSS title → ZmanimDay field mapping

| RSS `<title>` prefix | Field |
|---|---|
| `Dawn (Alot Hashachar)` | `Alos` |
| `Earliest Tallit and Tefillin (Misheyakir)` | `Misheyakir` |
| `Sunrise (Hanetz Hachamah)` | `Sunrise` |
| `Latest Shema` | `LatestShema` |
| `Midday (Chatzot Hayom)` | `Chatzos` |
| `Plag Hamincha` | `PlagHamincha` |
| `Sunset (Shkiah)` | `Shkiah` |
| `Nightfall (Tzeit Hakochavim)` | `Tzeis` |
| `Midnight (Chatzot HaLailah)` | `ChatzosHalaila` (time on D+1) |
| `Shaah Zmanit (proportional hour)` | `ShaahZmanitMin` (float64) |

Items `Latest Shacharit`, `Earliest Mincha (Mincha Gedolah)`, and
`Mincha Ketanah` are parsed and discarded.

## Chabad Candle Lighting ICS

### ZIP code
URL: `https://www.chabad.org/calendar/candlelighting/candlelighting.ics.asp?z=ZIP&before=N&bdef=0&weeks=52&tdate=M/D/YYYY`

### Lat/lon
URL: `https://www.chabad.org/calendar/candlelighting/candlelighting.ics.asp?locationType=3&coords=LAT,LON&tzname=TZ&before=N&bdef=0&weeks=52&tdate=M/D/YYYY`

Single fetch for the full year (`weeks=52`). Maximum appears to be 104 weeks (~2y).
DTSTART values are UTC (`…Z`), converted to local time via `time.In(tz)`.

Summary patterns → type:
- `"Light Candles"`, `"Light Holiday Candles"`, `"Light Shabbat Candles"` → Candles
- `"Shabbat Ends"`, `"Holiday Ends"`, `"Yom Tov Ends"` → Havdalah

**tdate encoding**: the endpoint requires literal slashes in `M/D/YYYY`. Go's
`url.Values.Encode()` percent-encodes `/` → `%2F`, causing HTTP 403. The `tdate`
parameter is therefore appended raw after encoding all other parameters.

## Chabad Geolocation Parameters

Confirmed from chabad.org web UI URL patterns (undocumented API):

| Parameter | ZIP | Lat/Lon |
|-----------|-----|---------|
| `locationId` / `z` | ZIP code | — |
| `locationType` | `2` | `3` |
| `coords` | — | `LAT,LON` (decimal, comma-separated) |
| `tzname` | — | IANA timezone with `/` replaced by `*` |

Example: `America/New_York` → `America*New_York`, `Europe/Berlin` → `Europe*Berlin`.

The `n=` parameter (display name) is accepted but not required.

## Fast Day Event Synthesis

For each `Subcat == "fast"` event two new events are appended:

| Event | DTSTART | VALARM |
|-------|---------|--------|
| "[Name] Begins" | `Alos` (or prev-day `Shkiah` for TB/YK) | −2h, −30min |
| "[Name] Ends" | `Tzeis` | −15min, PT0S |

UIDs: `didan-{YYYY-MM-DD}-fast-begin-{locationID}` / `…-fast-end-{locationID}`

## Alarm Policy

| Category | VALARM triggers |
|----------|----------------|
| `candles` | −2h, −15min |
| `havdalah` | −10min, PT0S |
| `fast-begin` | −2h, −30min |
| `fast-end` | −15min, PT0S |
| all-day / others | none |

## Chanuka Menora Note (appended to Description)

**Regular night:** l'chatchila window (shkiah–tzeis); b'dieved until tzeis+30min;
if necessary from Plag HaMincha b'dieved.

**Erev Shabbos** (candles event on same date): from Plag HaMincha before Shabbos candles.

**Motzoei Shabbos** (havdalah event on same date): after Havdalah; earliest is tzeis.

## Hebcal API Parameters

| Param | Value | Notes |
|-------|-------|-------|
| `v` | `1` | required |
| `cfg` | `json` | |
| `year` | e.g. `5786` | Hebrew year; mutually exclusive with start/end |
| `yt` | `H` | interpret as Hebrew year |
| `start`/`end` | `YYYY-MM-DD` | date range; mutually exclusive with year/yt |
| `maj/min/mf/ss/nx/s` | `on` | holidays, fasts, special Shabbatot, parsha |
| `mod` | `off` | skip modern Israeli holidays |
| `c` | `on` | candle lighting placeholder |
| `M` | `on` | havdalah at tzeis placeholder |
| `b` | `25` | candle offset (configurable) |
| `geo` | `zip`/`pos`/`geoname` | location type |
| `zip` | e.g. `17601` | for `geo=zip` |
| `latitude`/`longitude`/`tzid` | decimals + IANA | for `geo=pos` |
| `geonameid` | numeric | for `geo=geoname` |
| `i` | `off` | diaspora schedule |
| `lg` | `he`/`a`/`ah`/`he-x-NoNikud` | language |
| `leyning` | `on` | full kriyah detail |

## Leyning JSON Shape

The Hebcal `leyning` field is a flat JSON object mixing types:
- String keys: `"torah"`, `"haftarah"`, `"maftir"`, `"1"`–`"7"`
- Non-string keys skipped: `"triennial"` (nested object, Conservative cycle —
  not used), `"haftaraNumV"` (integer), others

Decoded via `map[string]json.RawMessage`, selectively unmarshalling strings.

## Go Constructs Reference

- **`func (c *Client) Method()`** — method on a pointer receiver. Pointer
  receiver means the method can modify the struct; avoids copying large structs.
- **`(value, error)`** — Go's canonical way to return a result with a possible
  error. Caller must check error before using value. No exceptions.
- **`defer f()`** — schedules `f` to run when the enclosing function returns.
  Used for cleanup: `defer resp.Body.Close()` immediately after opening.
- **`if err != nil { return ..., err }`** — explicit error propagation.
- **`map[string]struct{}`** — idiomatic set; `struct{}` occupies zero bytes.
- **`//go:embed path`** — compiler directive embedding file bytes into the
  binary at build time. Used for JSON in `internal/embeddata`.
- **`time.Parse("2006-01-02", s)`** — Go uses the specific reference time
  `Mon Jan 2 15:04:05 MST 2006` as its format template. `2006` = year,
  `01` = month, `02` = day, `15` = 24h hour, `04` = minute, `05` = second,
  `3` = 12h hour, `PM` = AM/PM indicator.
- **`sort.Slice(s, func(i, j int) bool { ... })`** — in-place sort with a
  comparator closure. `i` and `j` are indices; return true if `s[i] < s[j]`.
