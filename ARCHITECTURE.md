# Architecture

System design and external-data contracts for `zman-didan`. For build/test
commands, CLI flags, make targets, language modes, and "where mappings live,"
see [`CLAUDE.md`](CLAUDE.md) — that file is the operational reference and is
kept current; this file documents how the pipeline fits together and the
non-obvious quirks of the upstream data sources. Where the two would duplicate
a table, the canonical copy lives in `CLAUDE.md`.

## Module

`github.com/toobuntu/zman-didan` (binary: `bin/didan`).

## Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│  CLI input (cobra)                                          │
│  --year 5786 --zip 17601 --lang ah --candles 25            │
└────────────────────────┬────────────────────────────────────┘
                         │  types.Config
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/hebcal  Client.FetchYear()        [HTTPCache]     │
│  GET hebcal.com/hebcal?cfg=json&...&lg=<mapped>            │
│  ah/ahn: second GET lg=he → replace ev.Hebrew with nikud   │
│  → []types.HebcalEvent, types.Location, fromCache          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/chabad  Client.FetchCandlesYear() [HTTPCache]     │
│  GET chabad.org/.../candlelighting.ics.asp?weeks=52         │
│  → map[string]CandleDay  (string key "YYYY-MM-DD")          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/chabad  Client.FetchZmanimInZone() × N [ZmanimCache]│
│  GET chabad.org/tools/rss/zmanim.xml?tdate=YYYY-MM-DD       │
│  cache hit → cached ZmanimDay; miss → fetch→parse→cache     │
│  → map[string]types.ZmanimDay  (keyed "YYYY-MM-DD")         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/patcher  PatchCandles()                           │
│  candles + havdalah events: replace Date from CandleDay     │
│  (havdalah adds the tosfos offset)                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/attacher  AttachZmanim()                          │
│  candles  → append shkiah, tzeis                            │
│  havdalah → append Shma range (misheyakir–latest shema)     │
│  Pesach night 1 → chatzos halaila; Tisha B'Av → chatzos     │
│  Chanuka nights → menora lighting note                      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/fastday Build() → internal/alarm Rebuild()        │
│  synthesise fast begin/end events; repopulate VALARM blocks │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/haftorah  Patch()                                 │
│  prefer ev.Leyning.HaftarahChabad (API haftarah_chabad);    │
│  fall back to embedded haftorah_chabad.json                 │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/transliterator Apply()  (a/ah/ahn only)           │
│  longest-match-first substitution on Title/Description/Memo │
│  hn/ahn/shn: strip nikud (U+05B0–U+05C7)                   │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/cleaner Clean() → internal/specialdates Merge()   │
│  strip Hebcal boilerplate; merge Chabad Yomei d'Pagra       │
│  (special dates via HTTPCache)                              │
└────────────────────────┬────────────────────────────────────┘
                         │  sort all events by Date
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  internal/icalwriter  Write()                               │
│  RFC 5545, CRLF, 75-octet folding, bilingual SUMMARY,       │
│  LOCATION; VTIMEZONE static for America/New_York (Phase 2   │
│  for other zones)                                           │
└─────────────────────────────────────────────────────────────┘
```

## Key data structures

### types.HebcalEvent
Normalised from Hebcal JSON, mutable throughout the pipeline. `AllDay bool`
distinguishes date-only events (`DTSTART;VALUE=DATE`) from timed ones. `Slug`
is extracted from the Hebcal link URL for haftorah-table lookup. `Alarms` is
populated by the alarm package and serialised by icalwriter. `Leyning` carries
`Haftarah` and `HaftarahChabad` (see Haftorah source priority).

### types.ZmanimDay
All `time.Time` fields are in the location's IANA timezone, sourced verbatim
from the chabad.org RSS feed; no times are computed locally. `ShaahZmanitMin`
is a float64 duration in decimal minutes (60:30 → 60.5).

`ChatzosHalaila` — halachic midnight of the night D→D+1 — is reported in the
feed for day D as an AM time but physically occurs on D+1; `parseLocalTime`
detects the AM hour for that field and constructs the value on D+1.

`Misheyakir`'s RSS label differs by day type, which the parser must tolerate:
weekday `"Earliest Tallit and Tefillin (Misheyakir)"` vs Shabbos/Yom-Tov
`"Earliest Tallit (Misheyakir)"` (tallis worn, tefillin not). Likewise the
nightfall that ends Shabbos/YT is labelled `"Shabbat Ends"`/`"Holiday Ends"`
rather than `"Nightfall (Tzeit Hakochavim)"`. Substring classification (below)
handles both; an exact-string map historically dropped the Shabbos/YT variants,
zeroing `Misheyakir` and `Tzeis` on every Shabbos and Yom Tov.

### cache.ZmanimCache
JSON file at `~/.cache/didan/zmanim.json`, an envelope `{version, entries}` with
map keys `"YYYY-MM-DD|locationID"`; `time.Time` serialised as RFC3339. The file
carries `zmanimCacheVersion`; a version mismatch on load discards the whole
cache and forces a clean re-fetch (bump it whenever parser logic or the field
set changes — e.g. the Misheyakir variant fix). Zmanim are deterministic per
date and location and never change, so the 30-day retention window exists only
to bound storage, not for freshness; `Prune()` runs once per run at startup and
writes only if entries changed. Single-threaded; no locking.

### cache.HTTPCache
Raw-response cache at `~/.cache/didan/http/`, keyed by `sha256(url)`, with a
7-day mtime TTL. Backs the Hebcal calendar fetch (including the lg=he nikud
enrichment second request), the chabad.org candle ICS, and the special-dates
ICS. `--refresh` bypasses all caches.

## Event identification

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

- Returns today's zmanim by default; `tdate` overrides to any date
  (`YYYY-MM-DD`; no slash-encoding issue on this endpoint).
- The `<guid>` encodes feed *generation* time — ignored entirely.

### RSS title → ZmanimDay field mapping

Classification is **substring**, not exact-string, matching: each `<item>`
title's label is normalised, split, then tested against `classifierRules`
(`internal/chabad/client.go`) **in order, first match wins**. Substring matching
is deliberate — it absorbs the weekday/Shabbos/YT label variants that an
exact-string map cannot. A rule may set a canonical field, mark the token as an
`Events`-only entry, or both.

| RSS `<title>` substring token(s) | Field | Notes |
|---|---|---|
| `Alot Hashachar` | `Alos` | |
| `Misheyakir` | `Misheyakir` | matches both weekday `Earliest Tallit and Tefillin (Misheyakir)` and Shabbos/YT `Earliest Tallit (Misheyakir)` |
| `Hanetz Hachamah` / `Sunrise` | `Sunrise` | |
| `Latest Shema` / `Latest Kriat Shema` | `LatestShema` | |
| `Chatzot Hayom` / `Midday` | `Chatzos` | |
| `Plag` | `PlagHamincha` | |
| `Shkiah` / `Sunset` | `Shkiah` | |
| `Tzeit Hakochavim` / `Nightfall` | `Tzeis` | plain nightfall |
| `Shabbat Ends` / `Shabbos Ends` / `Holiday Ends` / `Yom Tov Ends` / `Holiday/Fast Ends` / `Shabbat/Holiday Ends` / `Fast Ends` | `Tzeis` (+ `Events`) | contextual "ends"; also stored as an event so callers can distinguish it from plain nightfall |
| `Chatzot HaLailah` / `Midnight` | `ChatzosHalaila` | time constructed on D+1 |
| `Candle Lighting` | `Events` only | context-specific (e.g. Erev YK candles are before shkiah, not tzeis) — no canonical field |
| `Fast Begins`, `Chametz` | `Events` only | |
| `Shaah Zmanit (proportional hour)` | `ShaahZmanitMin` (float64) | parsed as a duration, not a clock time |

`normalizeLabel` repairs a known feed defect where two labels run together
without a delimiter (`"Sunset (Shkiah)Fast Begins"` → `"… | Fast Begins"`).
Unrecognised labels are preserved as `Events`. Items such as `Latest Shacharit`
and the Mincha times are parsed and discarded.

## Chabad Candle Lighting ICS

### ZIP code
URL: `https://www.chabad.org/calendar/candlelighting/candlelighting.ics.asp?z=ZIP&before=N&bdef=0&weeks=52&tdate=M/D/YYYY`

### Lat/lon
URL: `https://www.chabad.org/calendar/candlelighting/candlelighting.ics.asp?locationType=3&coords=LAT,LON&tzname=TZ&before=N&bdef=0&weeks=52&tdate=M/D/YYYY`

Single fetch for the full year (`weeks=52`; max ~104). `DTSTART` values are UTC
(`…Z`), converted to local via `time.In(tz)`. The returned map uses a **string**
key (`"YYYY-MM-DD"`), not `time.Time`: `time.Time` equality includes the
`*Location` pointer, and repeated `time.LoadLocation` calls for one TZID can
return different pointers, causing silent map-lookup misses.

Summary patterns → type:
- `Light Candles` / `Light Holiday Candles` / `Light Shabbat Candles` → Candles
- `Shabbat Ends` / `Holiday Ends` / `Yom Tov Ends` → Havdalah

**tdate encoding**: the endpoint requires literal slashes in `M/D/YYYY`. Go's
`url.Values.Encode()` percent-encodes `/` → `%2F`, causing HTTP 403, so `tdate`
is appended raw after the other parameters are encoded.

## Chabad geolocation parameters

Confirmed from chabad.org web UI URL patterns (undocumented API):

| Parameter | ZIP | Lat/Lon |
|-----------|-----|---------|
| `locationId` / `z` | ZIP code | — |
| `locationType` | `2` | `3` |
| `coords` | — | `LAT,LON` (decimal, comma-separated) |
| `tzname` | — | IANA timezone with `/` replaced by `*` |

Example: `America/New_York` → `America*New_York`.

## Hebcal API

Base: `https://www.hebcal.com/hebcal?v=1&cfg=json`. Key parameters: `year`+`yt=H`
**or** `start`/`end`; `maj/min/mf/ss/nx/s=on`; `mod=off`; `c=on` (candle
placeholder); `M=on` (havdalah at tzeis placeholder); `b=N` (candle offset);
`geo=zip|pos|geoname` with the matching location params; `i=off` (diaspora);
`leyning=on`; and `lg=<value>`.

`--lang` → Hebcal `lg=` mapping (see `CLAUDE.md` for the full table): `h→he`,
`hn→he-x-NoNikud`, `a→a`, `ah→ah`, `ahn→ah`, `s→s`, `sh→sh`, `shn→sh`. For
`ah`/`ahn` the `lg=ah` response omits nikud in `hebrew`, so a second `lg=he`
request supplies nikud and `ev.Hebrew` is replaced by index position (both
requests share all other parameters, so item ordering matches). `hn`/`ahn`/`shn`
additionally strip nikud client-side.

### Leyning JSON shape
The `leyning` field is a flat object mixing types. String-valued keys consumed:
`torah`, `haftarah`, `haftarah_chabad`, `maftir`, `1`–`7`. Non-string keys
(`triennial` object, `haftaraNumV` integer) are skipped. Decoded via
`map[string]json.RawMessage`, selectively unmarshalling strings.

## Haftorah source priority

`haftorah.Patch` prefers `ev.Leyning.HaftarahChabad` (the API `haftarah_chabad`
field, live as of 2026-03-30; non-null only when Chabad differs from Ashkenazi
standard). When it is null, `ev.Leyning.Haftarah` is already correct. The
embedded `haftorah_chabad.json` is a fallback retained until a full-year
comparison confirms API coverage (known Tzav discrepancy — the API value is
authoritative).

## Fast day event synthesis

For each `Subcat == "fast"` all-day event, two timed events are appended:

| Event | DTSTART | VALARM |
|-------|---------|--------|
| "[Name] Begins" | `Alos` (or prev-day `Shkiah` for Tisha B'Av / Yom Kippur) | −2h, −30min |
| "[Name] Ends" | `Tzeis` | −15min, PT0S |

UIDs: `didan-{YYYY-MM-DD}-fast-begin-{locationID}` / `…-fast-end-{locationID}`.

## Alarm policy

| Category | VALARM triggers |
|----------|----------------|
| `candles` | −2h, −15min |
| `havdalah` | −10min, PT0S |
| `fast-begin` | −2h, −30min |
| `fast-end` | −15min, PT0S |
| all-day / others | none |

VALARM DESCRIPTION is always `"Event reminder"`.

## Chanuka menora note (appended to Description)

- **Regular night:** l'chatchila window shkiah–tzeis; b'dieved until tzeis+30min;
  if necessary from Plag HaMincha b'dieved.
- **Erev Shabbos** (candles event same date): from Plag HaMincha before Shabbos candles.
- **Motzoei Shabbos** (havdalah event same date): after Havdalah; earliest is tzeis.

## Candle lighting title forms

| chabad.org label | `IsYomTov` | `AfterHavdala` | SUMMARY |
|-----------------|-----------|----------------|---------|
| Light Shabbat Candles / Light Candles | false | false | `Candle lighting: 6:52 PM` |
| Light Holiday Candles | true | false | `YT candles: 7:05 PM` |
| Light Holiday Candles after | true | true | `YT candles after: 8:12 PM` |

The Havdala SUMMARY always begins with the English LTR time so it stays readable
in bidirectional clients.

## Data sources

| Source | Used for | Cache |
|--------|----------|-------|
| `hebcal.com/hebcal?cfg=json` | calendar structure, parsha, leyning, Hebrew text | HTTPCache |
| `chabad.org/tools/rss/zmanim.xml` | all zmanim per date | ZmanimCache |
| `chabad.org/.../candlelighting.ics.asp` | candle + havdalah times | HTTPCache |
| `download.hebcal.com/ical/chabad-special-dates.ics` | Yomei d'Pagra | HTTPCache |
| Embedded JSON | haftorahs (fallback), transliterations, rebbe data, Chitas | (compiled in) |

## Go constructs reference

- **`func (c *Client) Method()`** — pointer receiver; method can mutate the
  struct and avoids copying it.
- **`(value, error)`** — canonical result+error return; check the error before
  using the value.
- **`defer f()`** — runs `f` when the enclosing function returns; used for
  cleanup (`defer resp.Body.Close()`).
- **`map[string]struct{}`** — idiomatic set; `struct{}` is zero-width.
- **`//go:embed path`** — embeds file bytes into the binary at build time
  (`internal/embeddata`).
- **`time.Parse("2006-01-02", s)`** — Go's reference-time layout
  (`Mon Jan 2 15:04:05 MST 2006`): `2006` year, `01` month, `02` day,
  `15` 24h hour, `04` minute, `05` second, `3` 12h hour, `PM` meridiem.
- **`sort.Slice(s, less)`** — in-place sort with a `less(i, j) bool` closure.
