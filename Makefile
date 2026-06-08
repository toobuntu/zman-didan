# SPDX-FileCopyrightText: Copyright 2026 Todd Schulman
#
# SPDX-License-Identifier: GPL-3.0-or-later

.PHONY: build install dev test integration check fmt style scan vale actionlint reuse tidy hooks clean

BINARY := bin/didan

build:
	go build -o $(BINARY) ./cmd/didan

# install: build and install didan to $(go env GOBIN) or $GOPATH/bin.
install:
	go install ./cmd/didan

# dev: install missing developer tooling. Prefer Homebrew, then `go install`
# for Go tools; print an install hint and fail if every method is unavailable.
# Each entry is "<tool>|<go-install path>" ("" = no Go installer; brew/pipx only).
dev:
	@if command -v brew >/dev/null 2>&1; then brew=yes; else brew=no; fi; \
	for entry in \
		'staticcheck|honnef.co/go/tools/cmd/staticcheck@latest' \
		'govulncheck|golang.org/x/vuln/cmd/govulncheck@latest' \
		'actionlint|github.com/rhysd/actionlint/cmd/actionlint@latest' \
		'vale|' \
		'reuse|'; do \
		tool=$${entry%%|*}; gopath=$${entry#*|}; \
		if command -v "$$tool" >/dev/null 2>&1; then continue; fi; \
		if [ "$$brew" = yes ] && brew install "$$tool"; then continue; fi; \
		if [ -n "$$gopath" ] && go install "$$gopath"; then continue; fi; \
		echo "error: could not install '$$tool'." >&2; \
		echo "  install manually: brew install $$tool$${gopath:+  OR  go install $$gopath}" >&2; \
		exit 1; \
	done; \
	echo "dev tooling present."

# test: run all unit tests with -count=1 to bypass the Go test result cache.
test:
	go test -count=1 ./...

# integration: run tests that make real network requests (hebcal.com, chabad.org).
# Requires network access; excluded from normal `make test` by build tag.
integration:
	go test -count=1 -tags integration -timeout 60s ./internal/integration/

# check: full local suite mirroring CI (style + scan + test).
check: style scan test

# fmt: format all Go source files in place.
fmt:
	gofmt -l -w .

# style: format check + vet + staticcheck.
style:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "error: unformatted files (run 'make fmt'):"; \
		printf '%s\n' "$$unformatted"; \
		exit 1; \
	fi
	go vet ./...
	@command -v staticcheck >/dev/null 2>&1 || { \
		echo "error: staticcheck not found. Install with one of:"; \
		echo "  brew install staticcheck"; \
		echo "  go install honnef.co/go/tools/cmd/staticcheck@latest  (or: make dev)"; \
		exit 1; \
	}
	staticcheck ./...

# scan: vulnerability check.
scan:
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "error: govulncheck not found. Install with one of:"; \
		echo "  brew install govulncheck"; \
		echo "  go install golang.org/x/vuln/cmd/govulncheck@latest  (or: make dev)"; \
		exit 1; \
	}
	govulncheck ./...

# vale: en_US prose linter for Markdown docs.
# Install: brew install vale
vale:
	@command -v vale >/dev/null 2>&1 || { \
		echo "error: vale not found. Install: brew install vale"; \
		exit 1; \
	}
	vale README.md AGENTS.md docs/

# actionlint: lint workflow files.
# Install: brew install actionlint  OR  go install github.com/rhysd/actionlint/cmd/actionlint@latest
actionlint:
	actionlint .github/workflows/ci.yml

# reuse: REUSE license compliance.
# Install: brew install reuse  OR  pipx install reuse
reuse:
	reuse lint

# tidy: prune go.mod / go.sum.
tidy:
	go mod tidy

# hooks: register .githooks/ as the Git hooks directory for this clone.
hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit
	@echo "Pre-commit hook installed."
	@echo "Tools used by the hook (install via brew or go install):"
	@echo "  staticcheck  actionlint  reuse"

clean:
	rm -f $(BINARY)
