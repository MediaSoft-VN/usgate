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
test -f VERSION || {
  echo "VERSION file is missing" >&2
  exit 1
}

echo "==> Building native binary in Docker: ${OUTPUT}"
mkdir -p "$(dirname -- "${OUTPUT}")"
mkdir -p "$(dirname -- "${CLIENT_OUTPUT}")"
if [[ ${OUTPUT} == *.gz ]]; then
  rm -f "${OUTPUT%.gz}"
fi
image_name="new-api-native-builder:local"
container_name="new-api-native-builder-$$"
client_stage="$(mktemp -d "${TMPDIR:-/tmp}/new-api-client.XXXXXX")"
cleanup() {
  docker rm -f "${container_name}" >/dev/null 2>&1 || true
  docker image rm "${image_name}" >/dev/null 2>&1 || true
  rm -rf "${client_stage}"
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
docker cp "${container_name}:/build/web/dist/." "${client_stage}/"
tar -czf "${CLIENT_OUTPUT}" -C "${client_stage}" .

echo "==> Release checksum"
sha256sum "${OUTPUT}"
sha256sum "${CLIENT_OUTPUT}"
echo "==> Binary package ready: ${ROOT_DIR}/${OUTPUT}"
echo "==> Client package ready: ${ROOT_DIR}/${CLIENT_OUTPUT}"
