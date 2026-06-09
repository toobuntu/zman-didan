<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

---
status: accepted
date: 2026-06-08
decision-makers: Todd Schulman
---

# Track the latest stable Go as the go.mod version floor

## Context and Problem Statement

The `go` directive in `go.mod` is a version floor: it declares the minimum Go
the module is compatible with, gates which language and standard-library
features the code may use, and influences toolchain selection. CI reads it
through `actions/setup-go` with `go-version-file: go.mod` and
`check-latest: true`, so the directive is the single knob that determines which
Go the project builds and tests with (always the newest patch of the declared
minor).

The floor was just moved from 1.22 to 1.25 because `govulncheck` flagged a
standard-library advisory fixed only in a newer Go. That raised a recurring
question: how should the floor be chosen and maintained over time, and can the
maintenance be automated rather than remembered? The hope was that a bot could
keep it current the way Dependabot keeps module dependencies current.

## Decision Drivers

* Minimal recurring overhead — no routine pull requests or manual ritual for the floor itself.
* Security — a stdlib CVE fixed in a newer Go must be actionable, and CI must surface it.
* One source of truth — the floor lives only in `go.mod`, already consumed by CI.
* No downstream cost — didan is an application (a CLI), not a library other modules import, so a recent floor constrains nobody.
* Honesty about automation — adopt a bot only if it actually does the job, not as cargo cult.

## Considered Options

* Conservative floor: pin an older Go for maximum compatibility; bump rarely.
* Track the latest stable Go as the floor; bump only on a forcing signal.
* Automate floor bumps with a bot or scheduled action.

## Decision Outcome

Chosen option: **track the latest stable Go as the floor, bumping the `go`
directive only when a concrete signal forces it** — a `govulncheck` finding, a
dependency requirement surfaced by `go mod tidy`, or a wanted language/stdlib
feature. No bot bumps the directive; CI already follows `go.mod` and absorbs
newer patch releases on its own.

This is chosen because didan is a leaf application with no importers, so a
recent floor costs nothing externally, while a recent floor maximizes available
language and stdlib features and keeps the project on a supported, patched Go.
The maintenance burden is near zero: the directive changes only on a signal, and
the `govulncheck` step that just forced the 1.25 bump is the standing tripwire
for the next one.

### Consequences

* Good, because there are no routine floor-bump pull requests to triage; the directive moves only when something concrete requires it.
* Good, because `govulncheck` in CI fails the build on a stdlib advisory, making "the floor must move" a visible, gating signal rather than a thing to remember.
* Good, because the floor lives only in `go.mod`, and CI consumes it via `go-version-file` + `check-latest`, so CI and the declared floor cannot drift.
* Good, because `check-latest: true` means CI uses the newest patch of the current minor automatically — a security patch within the same minor is absorbed with no `go.mod` edit at all.
* Bad, because moving the floor itself is manual (`go mod edit -go=NN`, then `go mod tidy`); the project accepts this in exchange for not running a bot that would open noisy or unpinned PRs.
* Bad, because the floor drifts forward over time. This is irrelevant for an application, but the policy would need revisiting if didan were ever published as an imported library, where a high floor would constrain consumers.
* Neutral, because no `toolchain` directive is set. CI's `go-version-file` + `check-latest` selects the toolchain instead, keeping a single in-repo version source and letting CI float to the newest patch.

### Confirmation

`go.mod` declares `go 1.25`. The CI `build` job sets
`go-version-file: go.mod` and `check-latest: true`, and runs `govulncheck`,
which is the tripwire that forces the next floor bump. A floor change is a
one-line `go.mod` edit reviewed like any other change.

## Pros and Cons of the Options

### Conservative floor

Pin an older Go (for example, stay on 1.22) for the widest compatibility.

* Good, because it maximizes the range of toolchains that can build the project.
* Bad, because compatibility breadth has no value for a leaf application nobody imports.
* Bad, because it forecloses newer language and stdlib features for no offsetting benefit.
* Bad, because it still has to move for stdlib CVEs, so it does not actually remove the maintenance — it only delays and accumulates it.

### Track the latest stable Go, bump on a signal (chosen)

See [Consequences](#consequences).

### Automate floor bumps with a bot or scheduled action

Have a tool keep the `go` directive current automatically.

* Good, in principle, because it removes the manual edit.
* Bad, because Dependabot — already used here for `gomod` and `github-actions` — does not update the `go` directive at all; it updates module dependencies only, and does not bump the directive even when an updated dependency requires a newer Go (dependabot-core #9057, #9527).
* Bad, because Renovate's default is deliberately not to propose `go` directive upgrades (it follows the Go team's guidance to bump only when necessary, manually); opting into always-bump reintroduces churn for a change that warrants human judgment, since it changes the minimum supported Go and can affect toolchain selection.
* Bad, because a scheduled third-party action that opens floor-bump PRs adds an unpinned moving part and PR noise for a decision that should be deliberate.

## More Information

* The `go` directive means "compatible with this version or later" and is a floor; the separate `toolchain` directive means "use exactly this toolchain". Renovate bumps `toolchain` by default but not `go`. This project sets neither a `toolchain` line nor a bot, relying on `go-version-file` + `check-latest` in CI to pick the newest patch.
* Dependabot cannot bump the `go` directive: dependabot-core issues #9057 (directive not updated) and #9527 (not updated even when a dependency requires it). A standing feature request (#13520) covers bumping the `toolchain` directive, not `go`.
* The 1.25 bump (commits `907d852`, `2485276`) is the worked example of the forcing-signal model and the precedent this ADR generalizes.
