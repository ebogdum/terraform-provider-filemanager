#!/usr/bin/env bash

set -euo pipefail

MODE="ci"
if [[ "${1:-}" == "--staged" ]]; then
  MODE="staged"
elif [[ "${1:-}" == "--ci" ]]; then
  MODE="ci"
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-$ROOT_DIR/.cache/go-build}"
mkdir -p "$GOCACHE"

GO_BIN="$(go env GOPATH)/bin"
export PATH="$GO_BIN:$PATH"

if ! command -v tfplugindocs >/dev/null 2>&1; then
  echo "[docs-check] Installing tfplugindocs..."
  go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
fi

echo "[docs-check] Running tfplugindocs generate --provider-name=filemanager"
tfplugindocs generate --provider-name=filemanager

if [[ "$MODE" == "staged" ]]; then
  if ! git diff --exit-code -- docs >/dev/null; then
    echo "Docs are out of date. Run 'tfplugindocs generate --provider-name=filemanager' and stage updated docs."
    exit 1
  fi
else
  if ! git diff --exit-code >/dev/null; then
    echo "Docs are out of date. Run 'tfplugindocs generate --provider-name=filemanager' and commit."
    exit 1
  fi
fi

