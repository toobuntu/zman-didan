.PHONY: build fmt vet lint vuln check clean tidy hooks

BINARY := bin/didan

build:
	go build -o $(BINARY) ./cmd/didan

# Format all Go source files in place.
fmt:
	gofmt -w ./...

vet:
	go vet ./...

# Requires: go install honnef.co/go/tools/cmd/staticcheck@latest
lint: vet
	staticcheck ./...

# Requires: go install golang.org/x/vuln/cmd/govulncheck@latest
vuln:
	govulncheck ./...

# Requires: brew install actionlint
actionlint:
	actionlint .github/workflows/ci.yml

# Requires: pip install reuse
reuse:
	reuse lint

# Run the full local check suite (mirrors CI).
check: fmt vet lint vuln

tidy:
	go mod tidy

# Register .githooks/ as the Git hooks directory for this clone.
# Requires: go install honnef.co/go/tools/cmd/staticcheck@latest
# Optional: brew install actionlint; pip install reuse
hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit
	@echo "Pre-commit hook installed."
	@echo "Optional tools for full hook coverage:"
	@echo "  brew install actionlint"
	@echo "  pip install reuse"

clean:
	rm -f $(BINARY)
