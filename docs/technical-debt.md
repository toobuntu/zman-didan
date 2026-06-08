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

## P0-2 · `ARCHITECTURE.md` documents the pre-fix exact-match map — resolved

**Resolved.** Rewritten to document substring classification with both
Misheyakir label variants and the `Shabbat Ends → Tzeis` rule, plus the correct
module path, `map[string]CandleDay`, cache versioning/retention, `lg=` codes,
and the caching layer; relocated to `docs/architecture.md`.

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

## P1-6 · Integration tests failing on upstream drift

**Problem.** Three live-data integration tests fail against current
hebcal.com / chabad.org responses:

- `TestHebcalFetchYear_ParashaHasSlug` — parashat events have an empty `Slug`;
  Hebcal returns no `link` field for parashat items (confirmed: `jq` over the
  live response yields nothing), so `extractSlug` has nothing to extract. Low
  impact — haftorah now prefers the API `haftarah_chabad` field, which needs no
  slug.
- `TestHebcalFetchYear_AshkenaziNikud` — no nikud in `ev.Hebrew` for `lang=ah`.
  `enrichHebrew` replaces Hebrew by **slice index**, assuming the `lg=ah` and
  `lg=he` responses share ordering and count; when they diverge, alignment
  breaks.
- `TestSpecialDatesICS_VerboseNames` — the feed renamed the Rebbe to
  "R. Menachem M. Schneerson" (confirmed via `X-WR-CALDESC`), so `rebbes.json`
  verbose-name matching fails and the rebbe Yomei d'Pagra events go missing
  from generated calendars. **Superseded** by the self-owned Yomei d'Pagra
  redesign (roadmap §12).

**Acceptance criteria.**
- Slug: re-derive from an available field, or relax the test to "`Slug` OR
  `HaftarahChabad` present."
- Nikud: key `enrichHebrew` on a stable field (date + category), not index; a
  test asserts alignment survives a differing item set.
- Special dates: resolved by roadmap §12 (interim: update `rebbes.json`
  verbose_names to the current feed strings).

**Files.** `internal/hebcal/client.go`,
`internal/integration/integration_test.go`,
`internal/embeddata/files/rebbes.json`, `internal/specialdates/merge.go`.

---

## P1-7 · HTTP cache filenames are opaque hashes

**Problem.** Entries under `~/.cache/didan/http/` are named `<sha256(url)>` with
no indication of source or contents, which has repeatedly slowed cache debugging
(stale zmanim, poisoned cache, drifted special dates — each required grepping
every file to find the relevant one).

**Acceptance criteria.** Prefix each cache filename with a short source label so
files are self-identifying while staying unique, e.g. `http/hebcal-<sha256>`,
`http/candle-<sha256>`, `http/specialdates-<sha256>`, and
`http/yomadpagra-<sha256>` for the new package. Pass an explicit label argument
to the `HTTPCache` get/set API (do not infer it from the URL host). A label
slug must be filename-safe (lowercase, `[a-z0-9-]`). Old hash-only files are
harmlessly re-fetched once.

**Files.** `internal/cache/httpcache.go` and its callers in `internal/hebcal`,
`internal/chabad`, `internal/specialdates` (and future `internal/yomadpagra`).

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
