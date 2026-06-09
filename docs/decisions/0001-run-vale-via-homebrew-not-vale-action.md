<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

---
status: accepted
date: 2026-06-08
decision-makers: Todd Schulman
---

# Run Vale via Homebrew in CI rather than vale-cli/vale-action

## Context and Problem Statement

CI must enforce the en_US prose lint (`make vale`) on the project's Markdown.
The obvious choice is the official Vale GitHub Action, but the `errata-ai`
organization that published it has been renamed to `vale-cli`, and the action's
design conflicts with this project's requirements for a deterministic,
least-privilege CI gate. How should CI run Vale?

## Decision Drivers

* The job must fail the build, deterministically, on any error-level alert (an en_GB spelling).
* Least privilege: no `GITHUB_TOKEN` write scopes unless genuinely required.
* One source of truth for the linted file set (the Makefile `vale` target).
* Parity with the local install (`brew install vale`) to avoid CI/local drift.
* Supply-chain hygiene: pinned, auditable dependencies; nothing extra in the trust boundary.

## Considered Options

* `vale-cli/vale-action` (formerly `errata-ai/vale-action`)
* Download the pinned Vale release tarball and run `make vale`
* Install Vale via the runner's preinstalled Homebrew and run `make vale`

## Decision Outcome

Chosen option: **install Vale via Homebrew and run `make vale`**, because it is
the only option that is both deterministic as a gate and a match for the local
toolchain, with no third-party action inside the trust boundary.

### Consequences

* Good, because `vale` exits non-zero on error-level alerts on its own (`MinAlertLevel = error`), so the job needs no reviewdog, no reporter, and no `GITHUB_TOKEN`.
* Good, because the linted file set lives only in the Makefile `vale` target; CI runs `make vale`, so the two cannot drift.
* Good, because CI and local both obtain Vale from Homebrew, sidestepping the brew-vs-release-artifact vocabulary-handling discrepancy class.
* Good, because there is no third-party action to pin, audit, or trust for this step.
* Bad, because CI tracks whatever version Homebrew's index has rather than a pinned version. This is acceptable for a prose linter and is consistent with the `go install ...@latest` tooling already in CI; pin a version later if exact parity ever matters.
* Neutral, because Homebrew is preinstalled on the runner but not on `PATH`, so the job invokes `brew` by absolute path and prepends its bin to `GITHUB_PATH`.

### Confirmation

The `vale` job in `.github/workflows/ci.yml` installs Vale via
`/home/linuxbrew/.linuxbrew/bin/brew install vale` and runs `make vale`. A
single en_GB spelling in a linted file fails the job.

## Pros and Cons of the Options

### vale-cli/vale-action

The official action; Docker- and reviewdog-based.

* Good, because it is purpose-built and matches the "use a maintained action" pattern of the sibling linter jobs (actionlint, reuse).
* Bad, because `fail_on_error: true` has repeatedly failed to fail the build even when error-level alerts are present (errata-ai/vale issues #6 and #103) — disqualifying for a gate.
* Bad, because its reviewdog reporters need a `GITHUB_TOKEN` and `checks`/`pull-requests` write scopes.
* Bad, because the default `filter_mode: added` lints only diff-added lines, not the whole tree.
* Bad, because the publishing org was renamed (`errata-ai` to `vale-cli`), so every reference must be migrated and re-pinned.

### Pinned release tarball plus `make vale`

Download `vale_<version>_Linux_64-bit.tar.gz` from the GitHub release.

* Good, because deterministic and fully version-pinned.
* Good, because it reuses `make vale` as the single source of truth.
* Bad, because it hand-rolls a download/extract and introduces a second version knob that must be kept equal to the local install.
* Bad, because release-artifact builds have historically handled the vocabulary differently from Homebrew builds (errata-ai/vale issue #756), reintroducing CI/local divergence — the exact thing parity with `brew` is meant to avoid.

### Homebrew plus `make vale` (chosen)

See [Consequences](#consequences).

## More Information

* `MinAlertLevel = error` in `.vale.ini` is what makes the bare `vale` exit code a sufficient gate.
* Runner Homebrew location and the "not on PATH" caveat are documented in the actions/runner-images Ubuntu readme.
* This supersedes the initial tarball-based `vale` job.
