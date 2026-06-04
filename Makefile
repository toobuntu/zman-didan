.PHONY: build install dev fmt style scan check test integration clean tidy hooks actionlint reuse

BINARY := bin/didan

build:
	go build -o $(BINARY) ./cmd/didan

# install: build and install didan to $(go env GOBIN) or $GOPATH/bin.
install:
	go install ./cmd/didan

# dev: install the developer tooling used by the style/scan/actionlint targets.
# reuse is not go-installable — use `brew install reuse` or `pipx install reuse`.
dev:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/rhysd/actionlint/cmd/actionlint@latest

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

# test: run all unit tests with -count=1 to bypass the Go test result cache
test:
	go test -count=1 ./...

# integration: run tests that make real network requests (hebcal.com, chabad.org)
# Requires network access; excluded from normal `make test` by build tag.
integration:
	go test -count=1 -tags integration -timeout 60s ./internal/integration/

# check: full local suite mirroring CI (style + scan + test)
check: style scan test

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
