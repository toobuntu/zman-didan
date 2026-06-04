# Roadmap

Status legend: **done** · **in progress** · **planned**

This roadmap is forward-looking. For the catalogue of existing debt with
acceptance criteria, see [`technical-debt.md`](technical-debt.md). For the
system overview, see [`../ARCHITECTURE.md`](../ARCHITECTURE.md) (note: that
document currently has drift flagged in the debt list).

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
- **planned:** correct `ARCHITECTURE.md` — its RSS title→field table documents
  the pre-789265a exact-match map and would reintroduce the bug if trusted;
  the module path, `map[time.Time]CandleDay`, prune-on-every-write description,
  and `lg=` codes are also stale, and the caching layer is undocumented.

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
flat, label-prefixed, machine-oriented format we already parse robustly. The
grid would be strictly more brittle for no data we lack.

## 8. VTIMEZONE generalization — planned

`icalwriter` emits a hardcoded `VTIMEZONE` for `America/New_York` only and
silently omits it for other zones. Generate `VTIMEZONE` from the IANA tzdata
(DST transitions, offsets) so non-Eastern locations produce correct output.

## 9. Test coverage gaps — planned

Packages still without unit tests: `alarm`, `cleaner`, `haftorah`, `hebcal`,
`generator`. The `internal/integration` harness (build-tagged) is in place;
populate it with end-to-end fixtures.

## 10. Developer tooling — planned

`make check` currently fails on missing `staticcheck` and `govulncheck`.
Document/pin the install (or make the targets skip gracefully when absent).
See the debt list for acceptance criteria.
