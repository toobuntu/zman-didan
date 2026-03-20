.PHONY: build fmt style scan check clean tidy hooks

BINARY := bin/didan

build:
	go build -o $(BINARY) ./cmd/didan

# Format all Go source files in place.
fmt:
	gofmt -w ./...

# style: format check + vet + staticcheck
# Install staticcheck: brew install staticcheck  OR  go install honnef.co/go/tools/cmd/staticcheck@latest
style:
	@unformatted=$$(gofmt -l ./...); \
	if [ -n "$$unformatted" ]; then \
		echo "error: unformatted files (run 'make fmt'):"; \
		printf '%s\n' "$$unformatted"; \
		exit 1; \
	fi
	go vet ./...
	staticcheck ./...

# scan: vulnerability check
# Install: brew install govulncheck  OR  go install golang.org/x/vuln/cmd/govulncheck@latest
scan:
	govulncheck ./...

# check: full local suite mirroring CI (style + scan)
check: style scan

# Lint workflow files.
# Install: brew install actionlint  OR  go install github.com/rhysd/actionlint/cmd/actionlint@latest
actionlint:
	actionlint .github/workflows/ci.yml

# REUSE licence compliance.
# Install: brew install reuse  OR  pipx install reuse
reuse:
	reuse lint

tidy:
	go mod tidy

# Register .githooks/ as the Git hooks directory for this clone.
hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit
	@echo "Pre-commit hook installed."
	@echo "Tools used by the hook (install via brew or go install):"
	@echo "  staticcheck  actionlint  reuse"

clean:
	rm -f $(BINARY)
