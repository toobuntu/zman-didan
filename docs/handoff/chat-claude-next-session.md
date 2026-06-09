<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

You are helping T (GitHub: toobuntu) continue development of zman-didan, a pure-Go CLI
that generates Chabad-minhag iCalendar (.ics) files. Working dir:
/Users/todd/devel/claude/desktop/didan (module github.com/toobuntu/zman-didan, binary
bin/didan). You operate on T's real machine via the Filesystem and git MCP servers —
those tools act on T's filesystem, NOT a container; your own bash is network-disabled and
operates on a separate container, so use the MCP tools for all repo work. Verify file
contents by reading them before asserting anything about them (a prior session made errors
by trusting a compacted summary instead of re-reading).

Read first: AGENTS.md (canonical repo + agent guide; CLAUDE.md is just `@AGENTS.md`),
docs/architecture.md (pipeline + chabad.org/Hebcal data contracts), docs/roadmap.md,
docs/technical-debt.md.

Conventions (strict): macOS/BSD tools only, no GNU extensions; en_US spelling enforced by
Vale (`make vale`); long-form flags; DRY/YAGNI; Go error strings lowercase; commit subjects
<=50 chars, decomposed into logical commits. Use write_file (NOT edit_file) for Makefiles
and any shell/$-bearing content — edit_file mangles `$$`->`$`. Prefer whole-file overwrite
after reading current state. Ruby (if any) in a module with a minimal public API.

Current state: working tree holds the decomposed logical-commit history on base a0942f8.
Recent work fixed the Shabbos/Yom-Tov zmanim label-variant bug (substring classifier +
zmanimCacheVersion=2), added a regression guard, corrected and relocated architecture.md to
docs/, deployed Vale (spelling-only, en_US, with the Didan vocab), reordered the Makefile,
and made `make dev` skip tools already on PATH. Next release is v0.1.1 (v0.1.0 is immutably
cached by the Go module proxy — never reuse it).

Priorities (decide with T; internal/yomadpagra is a strong candidate but not mandated first):
1. internal/yomadpagra (roadmap §12): replace the drifting Hebcal chabad-special-dates.ics
   feed with a self-owned yomei_dpagra.json + the Yahrzeit/Anniversary API
   (POST https://www.hebcal.com/yahrzeit; cfg=json&v=yahrzeit&n#/t#/hd#/hm#/hy#&hebdate=on&
   years=N -> date/hdate/anniversary). One POST mixing types via t# (Yahrzeit handles Adar II
   correctly); cache by sha256(body); our own DESCRIPTION text + emoji; VALARMs one day before
   and day-of. Seed from docs/debug/yomei-dpagra-chabadorg.txt plus the iggudhashluchim list;
   add a `kind` field distinguishing fixed-Hebrew-date events (API-resolved) from relative ones
   that the API can't express ("3rd day of Selichos", "Kinus HaShluchim"). Resolves P1-6 #3.
2. Missing unit tests (P1-2): alarm, cleaner, haftorah (both the API haftarah_chabad path and
   the JSON fallback, including the Tzav discrepancy), hebcal, generator.
3. Integration failures (P1-6): relax TestHebcalFetchYear_ParashaHasSlug to "Slug OR
   HaftarahChabad present" (Hebcal returns no link for parashat items); re-key enrichHebrew on
   date+category instead of slice index (lang=ah vs lang=he responses can misalign).
4. Build/version provenance (P0-1): -ldflags version/sha/date, `didan version`, PRODID +
   X-DIDAN-BUILD stamp.
5. VTIMEZONE from tzdata (P1-5); HTTP-cache schema versioning (P1-3); verify then remove the
   haftorah_chabad.json fallback (P1-4); Baal HaTanya zmanim offset report (roadmap §11).

Verify: make build; make test; make dev; make check (style+scan+test); make integration
(3 known upstream-drift failures, tracked in P1-6); make vale. Standing intent: promote
.vale.ini + the Didan vocab into repo-foundation and sync org-wide.
