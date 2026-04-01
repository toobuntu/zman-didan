<!--
SPDX-FileCopyrightText: 2026 toobuntu

SPDX-License-Identifier: GPL-3.0-or-later
-->

# Copilot Instructions for toobuntu/zman-didan

Full instructions are in [CLAUDE.md](../CLAUDE.md). This file is a brief
orientation; defer to CLAUDE.md for authoritative detail.

## Quick start

```sh
make build          # go build -o bin/didan ./cmd/didan
make test           # go test -count=1 ./...
make check          # style + scan + test (mirrors CI)
```

## Before committing

- Run `make style` (gofmt check + go vet + staticcheck)
- Run `make test`
- Do **not** hand-write SPDX headers; run `scripts/annotate.sh` instead
- All Go files use `// SPDX-…` comment style; non-Go files use `.license` sidecars

## Key conventions

- Go 1.22+, internal packages under `internal/`, no exported types except from `types/`
- Lang flags: `h|hn|a|ah|ahn|s|sh|shn` (mapped in `internal/hebcal/client.go`)
- Tests: `go test -count=1 ./...`; integration: `make integration`
- Style: `gofmt` + `go vet` + `staticcheck` (`make check`)
