#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${NATIVE_OUTPUT:-deploy/new-api}"

cd "${ROOT_DIR}"

command -v bun >/dev/null 2>&1 || {
  echo "bun is required" >&2
  exit 1
}
command -v go >/dev/null 2>&1 || {
  echo "go is required" >&2
  exit 1
}
test -s VERSION || {
  echo "VERSION file is missing or empty" >&2
  exit 1
}

echo "==> Building frontend"
(
  cd web
  bun install --frozen-lockfile
  DISABLE_ESLINT_PLUGIN=true VITE_REACT_APP_VERSION="$(<../VERSION)" bun run build
)

echo "==> Building native binary: ${OUTPUT}"
mkdir -p "$(dirname -- "${OUTPUT}")"
GOWORK=off go mod download
GOWORK=off go build \
  -trimpath \
  -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=$(<VERSION)" \
  -o "${OUTPUT}" \
  .
chmod 0755 "${OUTPUT}"

echo "==> Release checksum"
sha256sum "${OUTPUT}"
echo "==> Package ready: ${ROOT_DIR}/${OUTPUT}"
