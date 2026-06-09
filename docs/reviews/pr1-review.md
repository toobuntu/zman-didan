<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

# PR #1 review — mine then close

A file-by-file disposition of PR #1 (omnibus, ~5,510 lines, ~64 files) so its
useful pieces are carried forward before the PR is closed as superseded. The
diff is archived at `docs/reviews/pr1.diff`; the authoritative file list at
`docs/reviews/pr1-name-only.txt`.

> **Verified line-by-line on 2026-06-09** against `main` (`83297fd`) via
> per-file `git diff <pr-head efd9ecad> <main>`. Every path in
> `pr1-name-only.txt` (63) now has exactly one verdict, and the Go-source
> verdicts below were re-derived from real diffs rather than inferred. Two
> things the original draft got wrong, now corrected: (1) the deltas
> (`+134/-50`, etc.) were measured against the PR's old base (`4919713`) and
> overstated the divergence — against current `main` these files differ by
> 7–48 changed lines, almost entirely SPDX-header, gofmt-whitespace, and
> comment-stripping churn (only `cache/zmanim.go` would revert an actual
> feature); (2) the P1-2 test scaffolding does **not** have drifted signatures
> — all three files compile and pass against `main` unmodified (38 tests).
> Every verdict still holds; **nothing new was salvaged**.

**Why it can't merge as a unit.** The PR's base predates the AGENTS.md
migration, the `docs/architecture.md` relocation, live `haftarah_chabad`,
SHA-pinned actions, and Go 1.25, and the branch is `CONFLICTING`. Re-applying
its Go-source edits would revert the zmanim cache-versioning envelope
(`cache/zmanim.go`), re-introduce the old `2026 toobuntu` SPDX headers and
un-gofmt'd formatting tree-wide, re-add a dead `origin()` helper that
`staticcheck` would flag, and delete the rationale comments since restored
(P2-3) — even though the *logic* on most of these files is already byte-identical
to `main`. (Note: the classification fix is **in** the PR, not predated by it;
see the source table.)

## Legend

- **pick-now** — carried into the working tree this session.
- **pick-defer** — useful; recorded in `technical-debt.md`, not yet applied.
- **have** — `main` already has an equal or better version.
- **drop** — superseded, stale, or redundant; not carried.
- **decide** — optional; a tooling-preference choice left to you.

## Carried into the working tree this session

- `.github/workflows/codeql.yml` — adopted (see below).
- `.github/workflows/ci.yml` — added least-privilege `permissions`,
  `persist-credentials: false`, a `Test` step, and `go mod tidy -diff`; rewrote
  the `vale` job to install via Homebrew (ADR 0001). The `actionlint` and
  `zizmor` jobs were later moved into `.github/workflows/actionlint.yml`.
- `.github/workflows/actionlint.yml` + `.github/actionlint.yaml` +
  `.github/zizmor.yml` + `.github/actionlint-matcher.json` (+`.license`) —
  Homebrew-style actionlint + zizmor(SARIF), adopted from PR #1's intent (see
  below).

## CI / workflows / supply chain

| File | Verdict | Notes |
| --- | --- | --- |
| `.github/workflows/codeql.yml` | **pick-now** | Genuinely good. Adopted with three tweaks: `go-version-file: go.mod` (was `go-version: '1.22'`), checkout SHA aligned to `v6.0.3`, and `pull_request` scoped to `main`. Matrix scans `go` (`autobuild` — `build-mode: none` is not supported for Go) and `actions` (the workflows themselves). Keeps `permissions: {}` + per-job least privilege, `concurrency`, and the `repository_owner == 'toobuntu'` fork guard. **Pinned:** `codeql-action` is at v4.36.2 (`pinact`-bumped since); Dependabot keeps it fresh. |
| `.github/workflows/ci.yml` | drop (reference) | Built on the old base. Its real value — least-privilege `permissions`, `persist-credentials: false`, and a unit-test step — was applied directly to current `ci.yml` instead. |
| `.github/dependabot.yml` | have (commit yours) | Your untracked local file already covers `gomod` + `github-actions`, grouped weekly — equal to the PR's. Two follow-ups: (1) commit it; (2) add the SPDX header (the PR's had one; yours doesn't, so `reuse lint` will flag it once tracked). The `github-actions` ecosystem is what keeps the action SHA pins fresh. |
| `.github/zizmor.yml` | **pick-now** | Adopted (Homebrew lineage): `unpinned-uses.config.policies: {Homebrew/actions/*: ref-pin}`. Now genuinely required, because `actionlint.yml` pins `Homebrew/actions/*@main` by ref and zizmor's `unpinned-uses` rule would otherwise fail. |
| `.github/workflows/actionlint.yml` | **pick-now** | Adopted Homebrew-style (modeled on `blackoutd`, the non-tap analog): plain `ubuntu-latest` + `setup-homebrew` + `cache-homebrew-prefix` installing `actionlint shellcheck zizmor`, `zizmor --format sarif` uploaded to the Security tab, then `actionlint` with a vendored matcher. The inline `actionlint` and `zizmor` jobs were removed from `ci.yml` in favor of this. SHAs aligned to didan's pins (checkout v6.0.3, `codeql-action` v4.36.2); reconfirm with `pinact`. The PR's `pip install zizmor` (fragile under PEP 668) is avoided. |
| `.github/actionlint-matcher.json` (+`.license`) | **pick-now** | Vendored verbatim (the standard actionlint problem-matcher) so the workflow's `::add-matcher::` does not depend on `brew --repository` internals. |
| `.github/workflows/copilot-setup-steps.yml` | decide | Bootstrap for the GitHub Copilot coding agent. Useful only if you use that agent. Documents the toolchain (`pinact reuse zizmor shellcheck shfmt gh`). Uses `Homebrew/actions/*@main` (unpinned — would draw a zizmor `unpinned-uses` finding). |
| `.github/copilot-instructions.md` | decide | Copilot custom instructions. Adopt only if using Copilot; otherwise AGENTS.md/CLAUDE.md already cover this. |

## REUSE / licensing

Current convention: **inline** SPDX headers for files that can carry a comment
(Markdown, Go, `go.mod`), `.license` sidecars only for files that can't (JSON,
`go.sum`), and the copyright string `Copyright 2026 Todd Schulman`. PR #1 used
the older "sidecar for everything + `2026 toobuntu`" convention.

| File | Verdict | Notes |
| --- | --- | --- |
| `LICENSES/GPL-3.0-or-later.txt` | have | Already in tree. |
| `go.sum.license` | have | `go.sum` can't carry a comment; sidecar already present. |
| `internal/embeddata/files/{haftorah_chabad,rebbes,transliterations,yomei_dpagra}.json.license` | have | JSON sidecars already present. |
| `go.mod.license`, `README.md.license`, `CLAUDE.md.license`, `ARCHITECTURE.md.license`, `docs/{rebbes_schema,zmanim_parser_data-driven_classifier,zmanim_parser_design}.md.license`, `docs/zmanim_parser_design.pdf.license`, `.gitignore.license` | drop | Old sidecar approach + `2026 toobuntu`. Current files use inline headers. `ARCHITECTURE.md.license` also targets the pre-relocation root path. (If `reuse lint` flags `.gitignore`, prefer an inline `# SPDX` header over a sidecar, to match the current convention.) |
| `internal/embeddata/files/transliterations.json` | drop (reference) | Data churn vs the old base; current file is authoritative. Diff only if a transliteration looks lost. |

## AI-assistant / editor config

| File | Verdict | Notes |
| --- | --- | --- |
| `.claude/settings.json` (+`.license`) | decide | Repo-level Claude Code settings. Adopt only if you want shared, committed config. |
| `.mcp.json` (+`.license`), `.vscode/mcp.json` (+`.license`) | decide | MCP server definitions. Adopt only if you want them committed; they can encode local paths. |
| `CLAUDE.md` | drop (superseded) | Predates the AGENTS.md migration; current CLAUDE.md points at AGENTS.md. |

## Build

| File | Verdict | Notes |
| --- | --- | --- |
| `Makefile` | have | Adds `test`/`integration` targets and `check: style scan test`. Current Makefile is a superset (also `dev`, `vale`, …). |
| `scripts/annotate.sh` | have | POSIX `reuse lint --json | jq | reuse annotate` helper; already in tree. Confirm parity if desired (PR's copyright string is `toobuntu`). |

## Go source (modified)

Verified line-by-line on 2026-06-09 via `git diff <pr-head> <main> -- <path>`.
Δ is **vs current `main`** (not the PR's old base). Every "drop" verdict holds
and **nothing is salvageable**: the differences reduce to three recurring causes
— the old `2026 toobuntu` SPDX header, gofmt whitespace, and comment-stripping
(the P2-3 churn) — plus one genuine feature-revert (`cache/zmanim.go`) and one
dead helper (`generator.go`). The uniform `+2/−5` signature is a pure header
swap (5-line block comment → 2-line `//` form), i.e. the code is byte-identical.

| File | Δ vs `main` | Verdict | Notes |
| --- | --- | --- | --- |
| `cmd/didan/main.go` | +2/−5 | have (verified) | Header-only vs `main`; the `--lang` rename (`h/hn/a/ah/ahn/s/sh/shn`) already landed. No `version` subcommand — **P0-1 is not minable here.** |
| `internal/cache/zmanim.go` | +5/−43 | **drop (verified)** | **The only file that would revert a feature.** PR lacks the cache-versioning envelope entirely — no `zmanimCacheVersion`, no `cacheFile{Version,Entries}`; `load`/`save` operate on a bare map. Adopting it reverts the versioning fix. |
| `internal/chabad/client.go` | +7/−26 | **drop (verified)** | Classifier **code byte-identical** (same `classifierRules`/`classifyLabel`/`normalizeLabel`). Diff = old SPDX + mis-aligned `const` block + PR **deletes** the rationale comments `main` keeps (incl. "reverting to exact matching silently drops every Shabbos/Yom-Tov zman"). Correction: the PR *contains* the classification fix; it is not "superseded by" it. |
| `internal/types/types.go` | +9/−16 | **drop (verified)** | SPDX + mis-gofmt field alignment (ignores `HaftarahChabad` width) + PR **deletes** the Misheyakir-variant rationale comment `main` keeps. |
| `internal/generator/generator.go` | +13/−9 | **drop (verified)** | SPDX + PR adds a **dead** `origin()` helper (never called; all 5 sites use its identical twin `hostTag` — `staticcheck` would flag it) + verbose error prefixes `main` shortened + PR drops a `calendarDateRange` comment `main` keeps. |
| `internal/hebcal/client.go` | +2/−5 | **drop (verified)** | **SPDX header only; code byte-identical.** Correction: not "see P1-6" — P1-6 is integration-test drift, unrelated to this blob's content. |
| `internal/haftorah/patcher.go` | +2/−5 | **drop (verified)** | **SPDX header only; code byte-identical.** Live `haftarah_chabad` priority already in both. |
| `internal/specialdates/merge.go` | +2/−5 | **drop (verified)** | **SPDX header only; code byte-identical.** (The roadmap §12 redesign is unrelated to this blob.) |
| `internal/transliterator/transliterator.go` | +2/−5 | **drop (verified)** | **SPDX header only; code byte-identical.** |
| `internal/cache/httpcache.go` | +2/−5 | have (verified) | Header-only vs `main`; PR body == `main` (both pre-versioning — P1-3 tracks that). The `+83` in the prior draft was vs the old base, where the file was new. |
| `internal/{alarm/builder,attacher/zmanim,cleaner/description,embeddata/embeddata,icalwriter/writer,patcher/candle}.go` | +2/−5 each | have (verified) | SPDX header only; code byte-identical to `main`. (Prior `+3/−0` was vs the old base, where the PR *added* the header.) |
| `internal/fastday/builder.go` | +10/−13 | have (verified) | **Not** header-only (prior draft grouped it as such): also gofmt struct-literal re-alignment in `beginEvent`/`endEvent`. Code identical; `main` is the gofmt-correct version. |

## Go tests (new in PR)

| File | Δ | Verdict | Notes |
| --- | --- | --- | --- |
| `internal/{attacher,cache,chabad,fastday,icalwriter,patcher,specialdates,transliterator}/*_test.go`, `internal/integration/integration_test.go` | new | have (verified) | Equivalents confirmed present on `main`. |
| `internal/alarm/builder_test.go` | +117 | **pick-now (adopted)** | No `alarm` test on `main`. **Verified: compiles and passes against `main` as-is** (8 tests). Encodes the current alarm policy (candles −120/−15, havdalah −10/0, fast-begin −120/−30, fast-end −15/0, all-day/unknown → nil). Drop-in. |
| `internal/cleaner/description_test.go` | +100 | **pick-now (adopted)** | No `cleaner` test on `main`. **Verified: compiles and passes as-is** (9 tests). Covers "Also spelled/known as", Hebcal-URL strip (chabad.org preserved), JPS attribution, blank-line collapse, `Memo`. Drop-in. |
| `internal/haftorah/patcher_test.go` | +442 | **pick-now (adopted)** | No `haftorah` test on `main`. **Verified: all 21 pass against `main` today** (`loadTable` still present). **Keep** the ~13 API-path / source-independent cases (`APIFieldPreferred`, `NullChabadNoEmbedded_NoChange`, `SkipsNonParashat`, the three `Rule1_*`, the seven `Rule2_*`). When **P1-4** removes `loadTable`/the embedded JSON, **drop** `FallbackToEmbeddedTable`, `SpecialShabbat`, `NilLeyning`, `LoadTable_ValidEntries`, `LoadTable_SkipsMetaKeys`, and **re-point** the `DescriptionReplace`/`Append`/`Empty` trio (they source their value from embedded `noach`) to a `HaftarahChabad` fixture. `APIFieldPreferred` ≡ `Rule1_Tzav` (duplicate). |

PR #1 covers 3 of the 5 P1-2 gaps (`alarm`, `cleaner`, `haftorah`); `hebcal`
and `generator` have **no test on `main`** (confirmed) and must be written
fresh. **Correction** to the prior draft (and to P1-2's reference note in
`technical-debt.md`): the signatures did **not** drift — all three files
compile and pass against `main` (`83297fd`) unmodified, 38 tests total, run
2026-06-09. They are drop-in, not merely "reference scaffolding."

## Closing message (for the PR)

> Superseded by current `main` (AGENTS.md migration, `docs/architecture.md`
> relocation, live `haftarah_chabad`, SHA-pinned actions, Go 1.25), and the
> branch now conflicts. A line-by-line pass confirmed the Go-source edits are
> byte-identical to `main` apart from the old SPDX header, gofmt formatting,
> and stripped rationale comments — only `cache/zmanim.go` would revert an
> actual feature (the cache-versioning envelope), so re-merging the source
> would be a net regression rather than a gain.
>
> Useful pieces were carried forward on `main`: the CodeQL workflow (`go` +
> `actions`), zizmor, least-privilege permissions + `persist-credentials`, and
> a CI unit-test step. The `alarm`, `cleaner`, and `haftorah` unit tests from
> this PR are adopted as-is (verified drop-in — 38 tests pass against `main`).
> Full disposition: `docs/reviews/pr1-review.md`.
