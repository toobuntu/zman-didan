<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

# Roadmap

Status legend: **done** · **in progress** · **planned**

This roadmap is forward-looking. For the catalog of existing debt with
acceptance criteria, see [`technical-debt.md`](technical-debt.md). For the
system overview, see [`architecture.md`](architecture.md).

## 1. Build and version provenance — planned (high priority)

**Motivation.** During the Shma-range investigation it was impossible to
determine which build produced any given generated `.ics`. The bug ultimately
traced to a stale binary plus a poisoned cache; a version/build stamp would
have made that a one-glance diagnosis.

**Scope.** Adopt the blackoutd-style versioning approach:

- Inject version, commit SHA, and build timestamp at link time via
  `-ldflags "-X main.version=… -X main.commit=… -X main.buildDate=…"`,
  with a `make build` rule and a fallback to `runtime/debug.ReadBuildInfo()`
  for `go install` builds.
- Stamp them into the iCalendar output: extend `PRODID` to
  `-//didan//<version>+<shortsha> (<buildDate>)//EN` and add an
  `X-DIDAN-BUILD:` property to `VCALENDAR`.
- Add a `didan version` subcommand.

## 2. Cache schema versioning — done (zmanim); planned (HTTP)

The zmanim cache now carries `zmanimCacheVersion` (envelope `{version, entries}`);
a version mismatch discards the cache and forces a clean re-fetch. This auto-heals
poisoned caches whenever parser logic changes. **Planned:** extend the same
guard to the HTTP body cache so a parser/format change can invalidate cached
raw responses too.

## 3. Regression guard for the Shabbos/Yom-Tov zmanim bug — planned (high priority)

Add an integration test that generates a known motzei-Shabbos **and** a
motzei-Yom-Tov Havdala and asserts the DESCRIPTION `Shma:` line is a range
(contains an en dash), not a lone time. Add a parser unit test covering the
Shabbos variant `"Earliest Tallit (Misheyakir)"` and `"Holiday Ends"` → tzeis.

## 4. Comment and documentation hygiene — in progress

- **done:** restored the rationale comments stripped from `chabad/client.go`
  and `types.go` (RSS label variants, the `)Word` defect, the event-flag
  semantics, the RSS item grammar, the Erev-YK candle edge case).
- **done:** corrected `ARCHITECTURE.md` (RSS substring classifier + label
  variants, module path, `map[string]CandleDay`, cache versioning/retention,
  `lg=` codes, caching layer) and relocated it to `docs/architecture.md`.

## 5. Haftorah verification and fallback removal — planned

Per the handoff: run a full Hebrew year (5786) and diff the live API
`haftarah_chabad` against the embedded `haftorah_chabad.json` (known Tzav
discrepancy). Once verified against the authoritative Chabad source, delete
`loadTable()`, the embedded JSON, and the fallback path in `haftorah.Patch`.

## 6. Additional Hebcal features — planned

- Sefirat HaOmer (`o=on`) — relevant for Chabad.
- Optional: Hebrew date on all events (`D=on`).
- Shipped already: Shabbat Mevarchim (`mvch=on`), Molad (`molad=on`),
  Rosh Chodesh.
- `ashkenazi_litvish` — investigated and rejected as too niche (handoff,
  2026-06-04). May revisit.

## 7. Zmanim source: stay on RSS — decided

chabad.org also exposes a printable monthly grid
(`/calendar/zmanimgrid_cdo/...`). Decision: **stay on the RSS feed.** The grid
is an HTML table built for human display (presentation markup, column headers,
pagination, no stable contract), whereas the RSS `<item><title>` lines are a
flat, label-prefixed, machine-oriented format didan already parses robustly. The
grid would be strictly more brittle without gaining data didan lacks.

## 8. VTIMEZONE generalization — planned

`icalwriter` emits a hardcoded `VTIMEZONE` for `America/New_York` only and
silently omits it for other zones. Generate `VTIMEZONE` from the IANA tzdata
(DST transitions, offsets) so non-Eastern locations produce correct output.

## 9. Test coverage gaps — in progress

`alarm`, `cleaner`, and `haftorah` now have table-driven unit tests (adopted
from PR #1, verified drop-in — 38 tests). Still without unit tests: `hebcal`
and `generator` (write fresh). The `internal/integration` harness (build-tagged)
is in place; populate it with end-to-end fixtures.

## 10. Developer tooling — planned

`make check` currently fails on missing `staticcheck` and `govulncheck`.
Document/pin the install (or make the targets skip gracefully when absent).
See the debt list for acceptance criteria.

## 11. Correlate Hebcal Baal HaTanya zmanim — planned

Hebcal is adding Alter Rebbe (Baal HaTanya) zmanim
([request](https://hebcal.userecho.com/communities/1/topics/1552-please-add-zmanim-according-to-the-baal-hatanya),
[hebcal-es6 commit](https://github.com/hebcal/hebcal-es6/commit/8b4e93d83a887a979df1d492f1f98494be45433e),
[issue #679](https://github.com/hebcal/hebcal-es6/issues/679)). chabad.org stays
authoritative, but to de-risk a future migration: fetch Hebcal's Baal HaTanya
zmanim alongside chabad.org's, cache them, and emit an internal per-zman offset
report (chabad.org − Hebcal) to quantify drift. A debug flag
(e.g. `--zmanim-offset-report`) writes the diff table; it is not part of normal
output. Parity is not yet 1:1, so this is correlation-only for now.

## 12. Self-owned Yomei d'Pagra via the Yahrzeit API — planned (high priority)

Today's special dates come from Hebcal's pre-built `chabad-special-dates.ics`,
whose SUMMARY strings drifted (e.g. "R. Menachem M. Schneerson", not
"...Menachem Mendel..."), breaking `rebbes.json` verbose-name matching and
dropping rebbe events (technical-debt P1-6). Replace the dependency on the
pre-built feed with a self-owned curated list:

- Maintain `yomei_dpagra.json` as the source of truth: per event, the Hebrew
  date (day + month), base Hebrew year (for the "Nth" count), type
  (Birthday/Passing/Anniversary), and didan's own DESCRIPTION text + emoji.
- Resolve Gregorian dates via the Yahrzeit + Anniversary API
  (`POST https://www.hebcal.com/yahrzeit`, `cfg=json&v=yahrzeit`,
  `n#/t#/hd#/hm#/hy#`, `hebdate=on`, `years=N`); the response yields `date`,
  `hdate`, and `anniversary` (the Nth count) per occurrence.
- Replace the API `.memo` with didan's text; add VALARMs one day before and on the
  day of each yoma d'pagra.
- Respect feed limits (Yahrzeit/Anniversary feeds cap at 1,200 events; at ~23
  events/year that bounds a *subscription* feed to ~52 years). didan generates
  per-run for a bounded span, so the cap does not bind one-shot output; note it
  for any future subscription endpoint.

This removes the name-drift failure mode entirely and gives full control over
wording, emoji, and alarms.

## 13. Evaluate the chabad-org-zmanim JSON web service — planned

`toolsforshlichus/chabad-org-zmanim` (a zero-dependency TypeScript client,
cloned for study) fetches chabad.org zmanim from a **different, richer endpoint
than didan's RSS**: `https://www.chabad.org/webservices/zmanim/zmanim/Get_Zmanim`
returns JSON, not an RSS feed.

Why it matters:

- **Typed `ZmanType` enum, no label parsing.** Each time carries a stable
  `ZmanType` (AlosHashachar, EarliestTefillin, NetzHachamah, LatestShema,
  LatestTefillah, LastEatingChametzTime, BurnChametzTime, Chatzos,
  MinchahGedolah, MinchahKetanah, PlagHaminchah, CandleLighting, Shkiah, Tzeis,
  ShabbosEnds, ChatzosNight, ShaahZmanit). This sidesteps the brittle RSS
  substring classifier behind the Shabbos/Yom-Tov Misheyakir regression — the
  single strongest argument for switching.
- **Date ranges in one request.** `startdate`/`enddate` (M/D/YYYY) fetch a span;
  didan currently issues one RSS request per date (~380 for a Hebrew year).
- **Variants resolved natively.** A `Default` boolean marks the canonical time
  when several variants exist, and `Footnotes` (keyed by `FootnoteType`) carry
  context didan now infers from label strings.
- **Access recipe.** `locationid` + `locationtype` (1 = Chabad city ID,
  2 = ZIP), `tdate` (M-D-YYYY), `aid` (affiliate, default 143790); headers
  include `x-user-agent: … co_ajax/2.0`, `sec-ch-ua`, and a `referer` to
  `/calendar/zmanim_cdo/...`. Times arrive as ASP.NET `"/Date(ms)/"` strings.

Tiers (decision pending — adopt, port ideas, or mine for data):

- **Mine now (no rewrite):** record the `ZmanType` enum, the param/header
  recipe, and the `Default`-flag + `Footnotes` semantics in `architecture.md`;
  note the ASP.NET date and HTML-entity handling.
- **Evaluate:** reimplement `internal/chabad` in Go against `Get_Zmanim`
  (range-capable, typed), retiring the RSS substring classifier. Prototype one
  known date and diff against the current RSS path before committing.
- **Avoid:** a runtime dependency on the TypeScript library — didan is pure Go.
  Reimplement from observed behavior; check the repo's LICENSE before copying
  any code, though API parameters and field names are facts, not expression.

Caveats: the endpoint is internal/undocumented (same status as didan's
`--lat/--lon` params) with no stability contract, and chabad.org's RSS terms
(contact before distribution) apply equally.
