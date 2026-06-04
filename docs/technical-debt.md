# Technical Debt

Prioritized debt register for `zman-didan`. Each item has a problem statement,
acceptance criteria, and the files to add or update. Forward-looking feature
work lives in [`roadmap.md`](roadmap.md); several items here are the debt
underpinning a roadmap entry.

Priority: **P0** (actively misleading or correctness-risking) ·
**P1** (impedes reliability or onboarding) · **P2** (hygiene).

---

## P0-1 · No build/version provenance in generated artifacts

**Problem.** Nothing identifies which build produced a given `.ics`. The
Shma-range investigation could not tell whether output came from current
source, a stale `bin/didan`, or a poisoned cache — the root cause was exactly
that ambiguity (a March-21 binary plus pre-789265a cache entries).

**Acceptance criteria.**
- Version, commit SHA, and build date are injected at link time and also
  recoverable from `runtime/debug.ReadBuildInfo()` for `go install` builds.
- Generated calendars carry the stamp: `PRODID` includes version+shortsha+date,
  and a `X-DIDAN-BUILD:` property is present on `VCALENDAR`.
- `didan version` prints the same triple.
- A test asserts `PRODID`/`X-DIDAN-BUILD` are populated and well-formed.

**Files.** Add `cmd/didan` version wiring + `version` subcommand; update
`internal/icalwriter/writer.go` (`Write` signature/PRODID), `Makefile`
(`-ldflags`), `internal/icalwriter/writer_test.go`.

---

## P0-2 · `ARCHITECTURE.md` documents the pre-fix exact-match map

**Problem.** The "RSS title → ZmanimDay field mapping" table lists only the
exact weekday strings (e.g. `Earliest Tallit and Tefillin (Misheyakir)`) and
says lookup is "prefix matching." That is the pre-789265a behavior that *caused*
the Shabbos/Yom-Tov zero-Misheyakir bug. A contributor trusting the doc would
reintroduce it. The same file also shows the stale module path
(`toobuntu/didan` vs `toobuntu/zman-didan`), `map[time.Time]CandleDay` (pre
string-key fix), "prunes where date < today on every write" (pre-retention),
old `lg=` codes, and no caching/enrichment layer.

**Acceptance criteria.**
- RSS section documents substring matching and lists both label variants
  (weekday vs Shabbos/YT) and the `Shabbat Ends`/`Holiday Ends` → tzeis rule.
- Module path, candle map key type, cache retention + versioning, `--lang`
  codes (`h/hn/a/ah/ahn/s/sh/shn`), HTTP caching, and nikud enrichment are
  accurate.
- Decide and note the location convention (see P2-1).

**Files.** `ARCHITECTURE.md` (or `docs/architecture.md` per P2-1).

---

## P0-3 · No regression test for the Shabbos/Yom-Tov zmanim classification

**Problem.** The bug shipped silently because no test exercised the Shabbos/YT
RSS label variants. `zmanimRange` correctly collapses a zero start, so the
defect was invisible end-to-end.

**Acceptance criteria.**
- Parser unit test: `classifyLabel("Earliest Tallit (Misheyakir)") == ("misheyakir", false)`
  and `classifyLabel("Holiday Ends") == ("tzeis", true)`.
- Integration test: a motzei-Shabbos and a motzei-Yom-Tov Havdala DESCRIPTION
  `Shma:` line contains an en dash (range), not a lone time.

**Files.** `internal/chabad/client_test.go`,
`internal/integration/` (build-tagged end-to-end fixture).

---

## P1-1 · Developer tooling unpinned; `make check` fails out of the box

**Problem.** `make check` runs `style` (`staticcheck`) and `scan`
(`govulncheck`); neither is installed by default, so `check` errors before the
test phase. There is no documented bootstrap.

**Acceptance criteria.**
- A documented install path (README/CONTRIBUTING or a `make tools` target):
  `go install honnef.co/go/tools/cmd/staticcheck@<pinned>` and
  `go install golang.org/x/vuln/cmd/govulncheck@<pinned>`.
- `make check` either runs clean after bootstrap or skips a missing tool with
  a clear warning rather than a hard error.
- Tool versions pinned (CI parity).

**Files.** `Makefile`, `README.md` (or new `CONTRIBUTING.md`), CI workflow
under `.github/`.

---

## P1-2 · Unit-test coverage gaps

**Problem.** `alarm`, `cleaner`, `haftorah`, `hebcal`, and `generator` have no
unit tests. `haftorah` is highest-risk: two code paths (API field vs embedded
JSON) and a known data discrepancy.

**Acceptance criteria.**
- Each package has table-driven tests for its public surface.
- `haftorah` tests cover both the API-`haftarah_chabad` path and the embedded
  fallback, including the Tzav discrepancy.
- `make test` exercises them; coverage does not regress.

**Files.** New `*_test.go` in each package; fixtures where network-free.

---

## P1-3 · HTTP body cache lacks schema versioning

**Problem.** The zmanim cache now self-invalidates on a version bump, but the
HTTP body cache (`internal/cache/httpcache.go`) keys only on URL+TTL. A change
to parsing or to the request parameters can leave stale raw responses being
re-parsed.

**Acceptance criteria.**
- HTTP cache entries are namespaced or versioned so a bump invalidates them.
- A test verifies a version bump causes a miss on a previously cached URL.

**Files.** `internal/cache/httpcache.go`, `internal/cache/*_test.go`.

---

## P1-4 · Embedded `haftorah_chabad.json` fallback is unverified dead weight

**Problem.** `haftorah.Patch` prefers the API `haftarah_chabad` field but still
carries an embedded JSON fallback with a known discrepancy (Tzav). The dual
path is untested and the JSON may be wrong.

**Acceptance criteria.**
- Full 5786 diff of live API vs embedded JSON, reconciled against the
  authoritative Chabad source.
- Once verified, remove `loadTable()`, the embedded JSON, and the fallback
  branch — or document why the fallback must remain.

**Files.** `internal/haftorah/patcher.go`,
`internal/embeddata/files/haftorah_chabad.json`,
`internal/embeddata/embeddata.go`, `internal/haftorah/patcher_test.go`.

---

## P1-5 · VTIMEZONE hardcoded to `America/New_York`

**Problem.** `icalwriter.writeVTimezone` emits a static block for Eastern only
and returns silently for any other zone, producing calendars without a
`VTIMEZONE` for non-Eastern locations.

**Acceptance criteria.**
- `VTIMEZONE` is generated from IANA tzdata (offsets + DST transitions) for any
  configured zone.
- Tests cover at least one non-US zone and a southern-hemisphere DST case.

**Files.** `internal/icalwriter/writer.go`, `internal/icalwriter/writer_test.go`.

---

## P2-1 · Documentation layout and root-vs-`docs/` convention

**Problem.** `docs/` now holds design docs, but `ARCHITECTURE.md` sits at the
repo root. The split is undecided, and comments duplicate rationale that could
live in one canonical doc.

**Acceptance criteria.**
- A decision recorded: keep `ARCHITECTURE.md` at root (common GitHub
  convention) **or** move to `docs/architecture.md` with a README pointer.
  Keep `CLAUDE.md` and `README.md` at root regardless.
- Triplicated rationale (e.g. the `time.Time`-key explanation) has one
  canonical home, with short in-code pointers — but only after the target doc
  is accurate (see P0-2).

**Files.** `ARCHITECTURE.md`/`docs/architecture.md`, `README.md`, `CLAUDE.md`.

---

## P2-2 · Archived `.del` files and debug artifacts in the tree

**Problem.** Three `*.go.del` archives (`normalizeLeyning*`, `birthdaybranch`,
`reformat`) and the `docs/debug/` investigation copies (including reproducible
`zmanim*.json` snapshots) are uncommitted clutter.

**Acceptance criteria.**
- `.del` files removed (`git rm`) in their owning feature commits; history is
  the archive.
- `docs/debug/` either gitignored or limited to provenance worth keeping
  (`client.log`, `client_zmanim.diff`); reproducible cache snapshots excluded.

**Files.** Delete `internal/hebcal/normalizeLeyning*.go.del`,
`internal/specialdates/{birthdaybranch,reformat}.go.del`; update `.gitignore`.

---

## P2-3 · Large mixed working tree pending decomposition

**Problem.** ~674/−408 across 12 tracked files plus untracked tests span several
unrelated features (caching, lang codes, nikud enrichment, haftarah_chabad,
transliteration, special dates). Committing as-is loses logical history and
risks bundling the comment-deletion churn.

**Acceptance criteria.**
- Changes land as the decomposed commits (see the chat decomposition); files
  touching multiple themes (`hebcal/client.go`, `specialdates/merge.go`) are
  split with `git add -p`.
- No commit contains the rationale-comment deletions (now restored).

**Files.** Whole working tree.
