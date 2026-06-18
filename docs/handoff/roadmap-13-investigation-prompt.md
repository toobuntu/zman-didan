<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

# Prompt — roadmap §13 investigation (chabad-org-zmanim JSON web service)

The bare one-liner ("Read `docs/handoff/chat-claude-next-session.md`. Commence
the investigation in `docs/roadmap.md` §13.") *almost* suffices, because §13 is
self-contained. The gap: that handoff is general onboarding and its priority
list puts §12 first, never mentioning §13 — an agent could drift. This prompt
pins the task to §13 and states the deliverable. Paste into a new Claude Code
session in the didan repo.

>>>

Read first, in order: `docs/handoff/chat-claude-next-session.md` (onboarding,
conventions, current state), then `AGENTS.md`, `docs/architecture.md`, and
`docs/roadmap.md` §13. Verify file contents by reading them; don't trust
summaries.

Your task is **roadmap §13 only** — evaluate the `Get_Zmanim` JSON web service
(`https://www.chabad.org/webservices/zmanim/zmanim/Get_Zmanim`) as an
alternative to didan's current RSS zmanim source. Ignore the numbered priority
list in the handoff (that targets §12); §13 is this session's work.

§13 already specifies the endpoint, the typed `ZmanType` enum, the
`startdate`/`enddate` range capability, the `Default`-flag + `Footnotes`
semantics, the access recipe (`locationid`/`locationtype`/`tdate`/`aid` +
headers), the ASP.NET `/Date(ms)/` format, and three decision tiers. A
zero-dependency TypeScript reference client, `toolsforshlichus/chabad-org-zmanim`,
is cloned locally for study — read it for the exact request/response shapes, but
take **no** runtime dependency on it (didan is pure Go; reimplement from observed
behavior, and check its LICENSE before copying any code — API parameters and
field names are facts, not expression).

Deliverables:

1. **Mine now (no rewrite):** a written investigation under `docs/` — either a
   new `docs/zmanim_get_zmanim_investigation.md` or an addition to
   `docs/architecture.md` — recording the `ZmanType` enum values, the
   param/header recipe, the `Default`-flag + `Footnotes` semantics, and the
   ASP.NET-date / HTML-entity handling, in enough detail that a future Go
   reimplementation needs no rediscovery.
2. **Evaluate:** a recommendation on whether `internal/chabad` should be
   reimplemented in Go against `Get_Zmanim` (range-capable, typed, would retire
   the RSS substring classifier behind the Shabbos/Yom-Tov regression).
   Prototype one known date against both the RSS path and `Get_Zmanim`, diff the
   results, and record the decision as a MADR-format ADR in `docs/decisions/`
   (didan's `0003` slot is free; `0001` is the Vale-via-Homebrew ADR;
   `0002` is the Go-version-floor ADR).
3. **Caveats:** note the endpoint is undocumented with no stability contract and
   that chabad.org's RSS terms (contact before distribution) apply equally;
   state whether they change the recommendation.

Conventions (strict): macOS/BSD tools only; en_US spelling (`make vale`);
long-form flags; commit subjects <= 50 chars, decomposed; signed commits — if
you are sandboxed and cannot sign, commit unsigned and tell T to re-sign (see
`docs/handoff/agent-commit-signing-procedure.md`). Do not push or open PRs.

>>>
