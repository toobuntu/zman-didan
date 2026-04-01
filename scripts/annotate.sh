#!/bin/sh
# SPDX-FileCopyrightText: 2026 toobuntu
# SPDX-License-Identifier: GPL-3.0-or-later

# Annotate unannotated files with SPDX headers for REUSE compliance.
# Usage: scripts/annotate.sh
#
# Requires: reuse (pip install reuse, or brew install reuse), jq

set -e

for tool in reuse jq; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "error: $tool is required but not found" >&2
        exit 1
    fi
done

files=$(reuse lint --json \
  | jq -r '.non_compliant
    | (.missing_copyright_info + .missing_licensing_info)
    | unique[]') || {
    echo "error: reuse lint or jq failed" >&2
    exit 1
}

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
# reuse's --style=python uses '#' comment style, which is correct for shell scripts.
printf '%s\n' "$files" | grep -E '\.sh$'  | annotate --style=python      || true
printf '%s\n' "$files" | grep -vE '\.(go|sh)$' | annotate --fallback-dot-license || true

echo "Done. Review changes with: git diff"
