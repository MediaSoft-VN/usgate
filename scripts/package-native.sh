#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${NATIVE_OUTPUT:-deploy/new-api.gz}"
CLIENT_OUTPUT="${CLIENT_OUTPUT:-deploy/new-api-client.tar.gz}"

cd "${ROOT_DIR}"

command -v docker >/dev/null 2>&1 || {
  echo "docker is required" >&2
  exit 1
}
docker compose version >/dev/null 2>&1 || {
  echo "docker compose is required" >&2
  exit 1
}
test -f VERSION || {
  echo "VERSION file is missing" >&2
  exit 1
}

echo "==> Building native binary with Docker Compose: ${OUTPUT}"
mkdir -p "$(dirname -- "${OUTPUT}")"
mkdir -p "$(dirname -- "${CLIENT_OUTPUT}")"
if [[ ${OUTPUT} == *.gz ]]; then
  rm -f "${OUTPUT%.gz}"
fi
NATIVE_PLATFORM="${NATIVE_PLATFORM:-linux/amd64}" \
  docker compose -f docker-compose.build.yml run --rm --build native-builder

if [[ ${OUTPUT} != deploy/new-api.gz ]]; then
  mv deploy/new-api.gz "${OUTPUT}"
fi
if [[ ${CLIENT_OUTPUT} != deploy/new-api-client.tar.gz ]]; then
  mv deploy/new-api-client.tar.gz "${CLIENT_OUTPUT}"
fi

echo "==> Release checksum"
sha256sum "${OUTPUT}"
sha256sum "${CLIENT_OUTPUT}"
echo "==> Binary package ready: ${ROOT_DIR}/${OUTPUT}"
echo "==> Client package ready: ${ROOT_DIR}/${CLIENT_OUTPUT}"
