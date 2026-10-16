#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${NATIVE_OUTPUT:-deploy/new-api.gz}"

cd "${ROOT_DIR}"

command -v docker >/dev/null 2>&1 || {
  echo "docker is required" >&2
  exit 1
}
test -f VERSION || {
  echo "VERSION file is missing" >&2
  exit 1
}

echo "==> Building native binary in Docker: ${OUTPUT}"
mkdir -p "$(dirname -- "${OUTPUT}")"
image_name="new-api-native-builder:local"
container_name="new-api-native-builder-$$"
cleanup() {
  docker rm -f "${container_name}" >/dev/null 2>&1 || true
  docker image rm "${image_name}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker build \
  --platform "${NATIVE_PLATFORM:-linux/amd64}" \
  --target builder2 \
  --tag "${image_name}" \
  .
docker create --name "${container_name}" "${image_name}" >/dev/null
binary_output="$(mktemp "${TMPDIR:-/tmp}/new-api.XXXXXX")"
docker cp "${container_name}:/build/new-api" "${binary_output}"
chmod 0755 "${binary_output}"
gzip -n -c "${binary_output}" > "${OUTPUT}"
rm -f "${binary_output}"

echo "==> Release checksum"
sha256sum "${OUTPUT}"
echo "==> Package ready: ${ROOT_DIR}/${OUTPUT}"
