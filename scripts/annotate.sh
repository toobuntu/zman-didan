#!/bin/sh
# SPDX-FileCopyrightText: 2026 toobuntu
# SPDX-License-Identifier: GPL-3.0-or-later

# Annotate unannotated files with SPDX headers for REUSE compliance.
# Usage: scripts/annotate.sh
#
# Requires: reuse (pip install reuse, or brew install reuse)

set -e

files=$(reuse lint --json \
  | jq -r '.non_compliant
    | add(.missing_copyright_info, .missing_licensing_info)
    | unique[]' 2>/dev/null) || true

if [ -z "$files" ]; then
    echo "All files are REUSE compliant."
    exit 0
fi

annotate() {
    xargs -r reuse annotate \
        --copyright="toobuntu" \
        --merge-copyrights \
        --license=GPL-3.0-or-later \
        --copyright-prefix=spdx-string \
        "$@"
}

printf '%s\n' "$files" | grep -E '\.go$'  | annotate --style=go          || true
printf '%s\n' "$files" | grep -E '\.sh$'  | annotate --style=python      || true
printf '%s\n' "$files" | grep -vE '\.(go|sh)$' | annotate --fallback-dot-license || true

echo "Done. Review changes with: git diff"
