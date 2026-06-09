<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

# PR #1 review — mine then close

A file-by-file disposition of PR #1 (omnibus, ~5,510 lines, ~64 files) so its
useful pieces are carried forward before the PR is closed as superseded. The
diff is archived at `docs/reviews/pr1.diff`; the authoritative file list at
`docs/reviews/pr1-name-only.txt`.

**Why it can't merge as a unit.** The PR's base predates the AGENTS.md
migration, the `docs/architecture.md` relocation, the Shabbos/Yom-Tov zmanim
classification fix, live `haftarah_chabad`, SHA-pinned actions, and Go 1.25.
Re-applying its source edits would revert newer work.

## Legend

- **pick-now** — carried into the working tree this session.
- **pick-defer** — useful; recorded in `technical-debt.md`, not yet applied.
- **have** — `main` already has an equal or better version.
- **drop** — superseded, stale, or redundant; not carried.
- **decide** — optional; a tooling-preference choice left to you.

## Carried into the working tree this session

- `.github/workflows/codeql.yml` — adopted (see below).
- `.github/workflows/ci.yml` — added least-privilege `permissions`,
  `persist-credentials: false`, a `Test` step, and a `zizmor` job; rewrote the
  `vale` job to install via Homebrew (ADR 0001). `Tidy` now uses
  `go mod tidy -diff`.

## CI / workflows / supply chain

| File | Verdict | Notes |
| --- | --- | --- |
| `.github/workflows/codeql.yml` | **pick-now** | Genuinely good. Adopted with three tweaks: `go-version-file: go.mod` (was `go-version: '1.22'`), checkout SHA aligned to `v6.0.3`, and `pull_request` scoped to `main`. Matrix scans `go` (`autobuild` — `build-mode: none` is not supported for Go) and `actions` (the workflows themselves). Keeps `permissions: {}` + per-job least privilege, `concurrency`, and the `repository_owner == 'toobuntu'` fork guard. **Pin to reconfirm:** the `codeql-action` SHA (`0d579ffd… # v4.32.6`) came from this PR; reconfirm with `pinact` and let Dependabot bump it. |
| `.github/workflows/ci.yml` | drop (reference) | Built on the old base. Its real value — least-privilege `permissions`, `persist-credentials: false`, and a unit-test step — was applied directly to current `ci.yml` instead. |
| `.github/dependabot.yml` | have (commit yours) | Your untracked local file already covers `gomod` + `github-actions`, grouped weekly — equal to the PR's. Two follow-ups: (1) commit it; (2) add the SPDX header (the PR's had one; yours doesn't, so `reuse lint` will flag it once tracked). The `github-actions` ecosystem is what keeps the action SHA pins fresh. |
| `.github/zizmor.yml` | drop | Placeholder only: `unpinned-uses.config.policies: {}` (empty = no effect). A zizmor config is needed only to *allow* an `@main` action (e.g. `Homebrew/actions/*`). The chosen Vale-via-`brew` approach uses no `@main` action, so no config is required; add one later only to suppress a specific finding. |
| `.github/workflows/actionlint.yml` | drop (redundant) | Standalone actionlint + zizmor(SARIF) workflow. actionlint already runs as a CI job (`raven-actions/actionlint`, pinned, annotates); zizmor now runs as its own job. The PR's `pip install zizmor` is also fragile under PEP 668 — `pipx run zizmor` / `uvx zizmor` avoid that. *If* you later want zizmor findings in the Security tab, lift only the SARIF-upload job from here. |
| `.github/actionlint-matcher.json` (+`.license`) | drop (redundant) | The official actionlint problem-matcher, needed only when invoking the `actionlint` binary directly. `raven-actions/actionlint` already registers annotations. |
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

Verdicts below for the larger files are inferred from the `cmd/didan/main.go`
spot-check plus the known base divergence, **not** a line-by-line diff of each
file. For certainty: `git diff <pr-sha>:<path> main:<path>` per file.

| File | Δ | Verdict | Notes |
| --- | --- | --- | --- |
| `cmd/didan/main.go` | +19/-11 | have (verified) | The `--lang` rename to `h/hn/a/ah/ahn/s/sh/shn` is already on `main`. No `version` subcommand — **P0-1 is not minable here.** |
| `internal/hebcal/client.go` | +134/-50 | drop (ref) | Against old base; superseded (see P1-6). |
| `internal/generator/generator.go` | +69/-35 | drop (ref) | Superseded. |
| `internal/specialdates/merge.go` | +54/-31 | drop (ref) | Superseded; redesign tracked (roadmap §12). |
| `internal/chabad/client.go` | +49/-47 | drop (ref) | Superseded by the zmanim-classification fix. |
| `internal/transliterator/transliterator.go` | +48/-25 | drop (ref) | Superseded. |
| `internal/haftorah/patcher.go` | +37/-6 | drop (ref) | Superseded by live `haftarah_chabad`. |
| `internal/cache/zmanim.go` | +31/-25 | drop (ref) | Superseded by cache versioning/retention. |
| `internal/types/types.go` | +13/-8 | drop (ref) | Superseded. |
| `internal/cache/httpcache.go` | +83 | have | New in PR; current repo already has it (P1-3 / P1-7). |
| `internal/{alarm/builder,attacher/zmanim,cleaner/description,embeddata/embeddata,fastday/builder,icalwriter/writer,patcher/candle}.go` | +3/-0 | have | The `+3/-0` is the SPDX header only; already present on `main`. |

## Go tests (new in PR)

| File | Δ | Verdict | Notes |
| --- | --- | --- | --- |
| `internal/{attacher,cache,chabad,fastday,icalwriter,patcher,specialdates,transliterator}/*_test.go`, `internal/integration/integration_test.go` | new | have | Equivalents already on `main`. |
| `internal/alarm/builder_test.go` | +117 | **pick-defer → P1-2** | No `alarm` test on `main`. Reference scaffolding (signatures may have drifted). |
| `internal/cleaner/description_test.go` | +100 | **pick-defer → P1-2** | No `cleaner` test on `main`. Reference scaffolding. |
| `internal/haftorah/patcher_test.go` | +442 | **pick-defer → P1-2** | No `haftorah` test on `main`. Mine the API-path cases; **drop the `loadTable()`/embedded-fallback cases** — P1-4 removes that path. |

PR #1 covers 3 of the 5 P1-2 gaps (`alarm`, `cleaner`, `haftorah`); `hebcal`
and `generator` are **not** covered here and remain to be written fresh.

## Closing message (for the PR)

> Superseded by current `main` (AGENTS.md migration, `docs/architecture.md`
> relocation, the Shabbos/Yom-Tov zmanim fix, live `haftarah_chabad`,
> SHA-pinned actions, Go 1.25), so this branch no longer merges and several of
> its source edits would revert newer work.
>
> Useful pieces carried forward on `main`: the CodeQL workflow (`go` +
> `actions`), zizmor, least-privilege permissions + `persist-credentials`, and
> a CI unit-test step. The `alarm`/`cleaner`/`haftorah` test scaffolding is
> tracked as reference for P1-2. Full disposition: `docs/reviews/pr1-review.md`.
