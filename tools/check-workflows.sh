#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-$ROOT_DIR/.cache/go-build}"
mkdir -p "$GOCACHE"

GO_BIN="$(go env GOPATH)/bin"
export PATH="$GO_BIN:$PATH"

run_step() {
  local title="$1"
  shift
  echo "[workflow-check] $title"
  "$@"
}

run_step "go mod download" go mod download
run_step "go build -v ." go build -v .
run_step "go vet ./..." go vet ./...
run_step "go test -v ./..." go test -v ./...
run_step "verify docs are up to date" "$ROOT_DIR/tools/check-docs.sh" --ci

if ! command -v goreleaser >/dev/null 2>&1; then
  echo "[workflow-check] Installing goreleaser..."
  go install github.com/goreleaser/goreleaser/v2@latest
fi

run_step "goreleaser check" goreleaser check

echo "[workflow-check] All local workflow checks passed."

