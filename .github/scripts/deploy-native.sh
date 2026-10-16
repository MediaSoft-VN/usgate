#!/usr/bin/env bash

set -euo pipefail

app_dir=/opt/new-api
release_id=${RELEASE_ID:?RELEASE_ID is required}
expected_sha256=${EXPECTED_SHA256:?EXPECTED_SHA256 is required}
uploaded_file="/tmp/new-api-${release_id}"
release_file="${app_dir}/releases/new-api-${release_id}"
previous_target="$(readlink "${app_dir}/new-api" 2>/dev/null || true)"

if [[ ! "${release_id}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Invalid release id" >&2
  exit 1
fi

if [[ ! "${expected_sha256}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "Invalid release checksum" >&2
  exit 1
fi

if [[ ! -f "${uploaded_file}" ]]; then
  echo "Uploaded release was not found" >&2
  exit 1
fi

actual_sha256="$(sha256sum "${uploaded_file}" | cut -d' ' -f1)"
if [[ "${actual_sha256}" != "${expected_sha256}" ]]; then
  echo "Release checksum verification failed" >&2
  exit 1
fi

sudo -n install -d -o new-api -g new-api -m 0755 "${app_dir}/releases"
sudo -n install -o new-api -g new-api -m 0755 "${uploaded_file}" "${release_file}"
sudo -n ln -sfn "${release_file}" "${app_dir}/new-api"

rollback() {
  echo "Deployment health check failed; restoring the previous release" >&2
  if [[ -n "${previous_target}" ]]; then
    sudo -n ln -sfn "${previous_target}" "${app_dir}/new-api"
    sudo -n systemctl restart new-api || true
  fi
}

if ! sudo -n systemctl restart new-api; then
  rollback
  exit 1
fi

healthy=false
for _ in {1..30}; do
  if curl --fail --silent --show-error http://127.0.0.1:3000/api/status >/dev/null; then
    healthy=true
    break
  fi
  sleep 2
done

if [[ "${healthy}" != true ]]; then
  rollback
  exit 1
fi

rm -f "${uploaded_file}"
find "${app_dir}/releases" -maxdepth 1 -type f -name 'new-api-*' -printf '%T@ %p\n' \
  | sort -nr \
  | awk 'NR > 5 { sub(/^[^ ]+ /, ""); print }' \
  | xargs -r sudo -n rm -f

echo "Deployed ${release_id} successfully"
