#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

git config core.hooksPath .githooks

chmod +x \
  .githooks/pre-commit \
  .githooks/pre-push \
  tools/check-docs.sh \
  tools/check-workflows.sh \
  tools/install-git-hooks.sh

echo "Installed git hooks via core.hooksPath=.githooks"
echo "Current hooks path: $(git config --get core.hooksPath)"

