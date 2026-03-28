#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CACHE_ROOT="${REPO_ROOT}/.cache"

export GOCACHE="${GOCACHE:-${CACHE_ROOT}/go-build-cache}"
export GOTMPDIR="${GOTMPDIR:-${CACHE_ROOT}/go-build-tmp}"
export TMPDIR="${TMPDIR:-${CACHE_ROOT}/tmp}"

mkdir -p "$GOCACHE" "$GOTMPDIR" "$TMPDIR"
