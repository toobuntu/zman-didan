<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

You are taking over development of `zman-didan` (module: github.com/toobuntu/zman-didan),
a Go CLI tool that generates Chabad-minhag iCalendar (.ics) files. The project is at
/Users/todd/devel/claude/desktop/didan/. Start by reading CLAUDE.md in full, then
internal/types/types.go, internal/hebcal/client.go, and internal/haftorah/patcher.go
to orient yourself. The Filesystem MCP server is connected; use it for all file access.

## Current state (as of 2026-03-30)

The haftarah_chabad feature is now fully live. The Hebcal REST API (www.hebcal.com)
returns a `haftarah_chabad` field in items[].leyning when Chabad custom differs from
Ashkenazi standard — no URL parameter needed. didan already parses and prefers this
field (in Leyning.HaftarahChabad), with embedded haftorah_chabad.json as fallback.

## Immediate priorities

**1. Haftorah verification**
Run a full Hebrew year (e.g. 5786) and compare what the live API returns in
`haftarah_chabad` against the embedded `haftorah_chabad.json`. Known discrepancy:
Tzav — API says `Jeremiah 7:21-28, 9:22-23`; JSON has `7:21-8:3, 9:22-23`.
Cross-check both against https://www.chabad.org/library/article_cdo/aid/4158333.
Once verified, delete `loadTable()`, the embedded JSON fallback path in
`haftorah.Patch`, and the JSON file itself.

To generate a comparison: run `./bin/didan generate --year 5786 --zip 17601 --refresh`
then grep for DESCRIPTION lines containing "Haftorah:". Also add an integration test
that calls `FetchYear` for a known parsha and asserts `ev.Leyning.HaftarahChabad`.

**2. ashkenazi_litvish investigation**
2026-06-04 update: investigated and rejected as too niche. Chabad Yiddish does
sound this way, but the transliterations are non-standard and jarring at first
glance. Could perhaps revisit this at some later time. Abandoned for now.

The Hebcal API exposes `lg=ashkenazi_litvish` as a language parameter and
`leyning.ashkenazi_litvish` as a haftorah variant. Chabad has Lithuanian geographic
roots. Fetch a few weeks of calendar with `lg=ashkenazi_litvish` and compare event
title transliterations against `lg=ah` and against authoritative Chabad sources.
If it's a better match, add it as `--lang al` or similar.

Relevant API call for comparison:
  curl -fSsL "https://www.hebcal.com/hebcal?v=1&cfg=json&start=2026-04-01&end=2026-04-30&geo=zip&zip=17601&maj=on&ss=on&s=on&lg=ashkenazi_litvish"

**3. Additional Hebcal API features**
Evaluate whether to add these to the pipeline:
- `o=on` — Sefirat HaOmer (Days of the Omer), highly relevant for Chabad
- `mvch=on` — Shabbat Mevarchim (Shabbos before Rosh Chodesh)
- `molad=on` — Molad announcement at Shabbat Mevarchim
- `D=on` — Hebrew date on all events

**4. Test suite gaps**
These packages have no unit tests yet: alarm, cleaner, haftorah, hebcal.
The haftorah package is now the most critical since it has two code paths
(API field vs embedded JSON fallback) and a known discrepancy to verify.

## Key API facts discovered this session

- `haftarah_chabad` is non-null ONLY when Chabad differs from Ashkenazi standard.
  When null, `haftarah` (Ashkenazi standard) is already correct for Chabad.
  `haftarah_sephardic` is NOT a valid Chabad fallback.
- `ashkenazi_standard` in the leyning object = same as `haftarah` (redundant).
- Go's default HTTP transport adds Accept-Encoding: gzip and decompresses
  transparently — no code change needed to benefit from Hebcal's compression.
- Hebcal's Cache-Control/Expires headers are properly set; our HTTPCache (mtime-based,
  7-day TTL) is compatible but ignores those headers in favour of our own TTL.
- anash.org's zmanim widget uses WordPress AJAX to their own PHP backend, which
  proxies to chabad.org server-side. No private chabad.org API is involved.

## Coding conventions

- Go, internal packages under internal/, no exported types except from types/
- Filesystem write_file always replaces entire file — read before writing
- Lang flags: h|hn|a|ah|ahn|s|sh|shn  (mapped in hebcal/client.go langToAPIParam)
- Tests: go test -count=1 ./... (make test); integration: make integration
- Style: gofmt + go vet + staticcheck (make check)
