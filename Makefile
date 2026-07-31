SHELL := /usr/bin/env bash
GO    ?= go
BIN   := bin
PKGS  := ./...

# Every GOOS/GOARCH pair the provider ships, matching .goreleaser.yml.
PLATFORMS := \
	freebsd/amd64 freebsd/386 freebsd/arm freebsd/arm64 \
	windows/amd64 windows/386 windows/arm64 \
	linux/amd64 linux/386 linux/arm linux/arm64 \
	darwin/amd64 darwin/arm64

.PHONY: all
all: build

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: fmt
fmt:
	$(GO) fmt $(PKGS)

.PHONY: vet
vet:
	$(GO) vet $(PKGS)

.PHONY: build
build:
	mkdir -p $(BIN)
	$(GO) build -o $(BIN)/terraform-provider-filemanager .

.PHONY: build-all
build-all:
	@failed=""; \
	for platform in $(PLATFORMS); do \
	  goos="$${platform%/*}"; goarch="$${platform#*/}"; \
	  printf '%-20s ' "$$platform"; \
	  if CGO_ENABLED=0 GOOS="$$goos" GOARCH="$$goarch" $(GO) build -o /dev/null $(PKGS) 2>/dev/null; then \
	    echo OK; \
	  else \
	    echo FAILED; failed="$$failed $$platform"; \
	  fi; \
	done; \
	if [ -n "$$failed" ]; then echo; echo "Failed:$$failed"; exit 1; fi; \
	echo; echo "All platforms build successfully."

.PHONY: test
test:
	$(GO) test -count=1 -race $(PKGS)

.PHONY: docs
docs:
	./tools/check-docs.sh

.PHONY: vulncheck
vulncheck:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest $(PKGS)

.PHONY: release-snapshot
release-snapshot:
	goreleaser release --snapshot --clean --skip=publish,sign

.PHONY: release
release:
	@# Tag the release first: git tag -s vX.Y.Z -m '...'
	@# Requires GPG_FINGERPRINT in env for signing.
	goreleaser release --clean

.PHONY: clean
clean:
	rm -rf $(BIN) dist
