#!/usr/bin/env bash

set -Eeuo pipefail

: "${APP_DIR:=/opt/new-api}"
: "${APP_PORT:=}"
: "${APP_USER:?APP_USER is required}"
: "${DOMAIN:?DOMAIN is required}"
: "${CERTBOT_EMAIL:?CERTBOT_EMAIL is required}"
: "${NEW_API_ENV:?NEW_API_ENV is required}"
: "${RELEASE_ID:?RELEASE_ID is required}"
: "${EXPECTED_SHA256:?EXPECTED_SHA256 is required}"
: "${CLIENT_EXPECTED_SHA256:?CLIENT_EXPECTED_SHA256 is required}"
: "${SSH_USER:?SSH_USER is required}"
: "${TENANT_DOMAIN_INSTALLER_SHA256:?TENANT_DOMAIN_INSTALLER_SHA256 is required}"
: "${TENANT_DOMAIN_HOST_SHA256:?TENANT_DOMAIN_HOST_SHA256 is required}"
: "${TENANT_DOMAIN_SOCKET_SHA256:?TENANT_DOMAIN_SOCKET_SHA256 is required}"
: "${OLD_DOMAIN:=}"
if [[ -z ${OLD_DOMAIN//[[:space:]]/} ]]; then
  OLD_DOMAIN=""
fi

uploaded_file="/tmp/new-api-${RELEASE_ID}.gz"
uploaded_client_file="/tmp/new-api-client-${RELEASE_ID}.tar.gz"
uploaded_provisioner_dir="/tmp/new-api-tenant-domain-${RELEASE_ID}"
uploaded_provisioner_installer="${uploaded_provisioner_dir}/install-tenant-domain-provisioner.sh"
uploaded_provisioner_host="${uploaded_provisioner_dir}/tenant-domain-host.sh"
uploaded_provisioner_socket="${uploaded_provisioner_dir}/tenant-domain-socket.py"
release_file="${APP_DIR}/releases/new-api-${RELEASE_ID}"
client_release_dir="${APP_DIR}/client/releases/client-${RELEASE_ID}"
service_file=/etc/systemd/system/new-api.service
nginx_site=/etc/nginx/sites-available/new-api.conf
env_file="${APP_DIR}/.env"
port_file="${APP_DIR}/.app-port"
previous_target="$(readlink "${APP_DIR}/new-api" 2>/dev/null || true)"

validate_value() {
  local name=$1
  local value=$2
  local pattern=$3
  if [[ ! ${value} =~ ${pattern} ]]; then
    echo "Invalid ${name}" >&2
    exit 1
  fi
}

validate_value APP_DIR "${APP_DIR}" '^/[A-Za-z0-9._/-]+$'
if [[ -n ${APP_PORT} ]]; then
  validate_value APP_PORT "${APP_PORT}" '^[0-9]+$'
fi
validate_value APP_USER "${APP_USER}" '^[A-Za-z_][A-Za-z0-9_-]*$'
validate_value DOMAIN "${DOMAIN}" '^[A-Za-z0-9.-]+$'
validate_value CERTBOT_EMAIL "${CERTBOT_EMAIL}" '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$'
validate_value RELEASE_ID "${RELEASE_ID}" '^[0-9a-f]{40}$'
validate_value EXPECTED_SHA256 "${EXPECTED_SHA256}" '^[0-9a-f]{64}$'
validate_value CLIENT_EXPECTED_SHA256 "${CLIENT_EXPECTED_SHA256}" '^[0-9a-f]{64}$'
validate_value SSH_USER "${SSH_USER}" '^[A-Za-z_][A-Za-z0-9_-]*$'
validate_value TENANT_DOMAIN_INSTALLER_SHA256 "${TENANT_DOMAIN_INSTALLER_SHA256}" '^[0-9a-f]{64}$'
validate_value TENANT_DOMAIN_HOST_SHA256 "${TENANT_DOMAIN_HOST_SHA256}" '^[0-9a-f]{64}$'
validate_value TENANT_DOMAIN_SOCKET_SHA256 "${TENANT_DOMAIN_SOCKET_SHA256}" '^[0-9a-f]{64}$'
validate_value OLD_DOMAIN "${OLD_DOMAIN}" '^[A-Za-z0-9.-]*$'

if [[ ! -f ${uploaded_file} ]]; then
  echo "Uploaded release was not found" >&2
  exit 1
fi
if [[ ! -f ${uploaded_client_file} ]]; then
  echo "Uploaded client release package was not found" >&2
  exit 1
fi
for uploaded_script in "${uploaded_provisioner_installer}" "${uploaded_provisioner_host}" "${uploaded_provisioner_socket}"; do
  if [[ ! -f ${uploaded_script} ]]; then
    echo "Tenant domain provisioner package is incomplete" >&2
    exit 1
  fi
done

gzip -t "${uploaded_file}"

actual_sha256="$(sha256sum "${uploaded_file}" | cut -d' ' -f1)"
if [[ ${actual_sha256} != "${EXPECTED_SHA256}" ]]; then
  echo "Release checksum verification failed" >&2
  exit 1
fi
client_actual_sha256="$(sha256sum "${uploaded_client_file}" | cut -d' ' -f1)"
if [[ ${client_actual_sha256} != "${CLIENT_EXPECTED_SHA256}" ]]; then
  echo "Client package checksum verification failed" >&2
  exit 1
fi
client_manifest="/tmp/new-api-client-${RELEASE_ID}.list"
tar -tzf "${uploaded_client_file}" >"${client_manifest}"
while IFS= read -r client_path; do
  client_path="${client_path#./}"
  if [[ ${client_path} == /* || ${client_path} == .. || ${client_path} == ../* ||
    ${client_path} == */../* || ${client_path} == */.. ]]; then
    echo "Client package contains an unsafe path" >&2
    rm -f "${client_manifest}"
    exit 1
  fi
done <"${client_manifest}"
rm -f "${client_manifest}"
if [[ $(sha256sum "${uploaded_provisioner_installer}" | cut -d' ' -f1) != "${TENANT_DOMAIN_INSTALLER_SHA256}" ||
  $(sha256sum "${uploaded_provisioner_host}" | cut -d' ' -f1) != "${TENANT_DOMAIN_HOST_SHA256}" ||
  $(sha256sum "${uploaded_provisioner_socket}" | cut -d' ' -f1) != "${TENANT_DOMAIN_SOCKET_SHA256}" ]]; then
  echo "Tenant domain provisioner checksum verification failed" >&2
  exit 1
fi

app_group="$(id -gn "${APP_USER}")"
sudo -n install -d -o "${APP_USER}" -g "${app_group}" -m 0755 \
  "${APP_DIR}" "${APP_DIR}/releases" "${APP_DIR}/client/releases" "${APP_DIR}/logs" "${APP_DIR}/data"

echo "========== Install client release package =========="
sudo -n rm -rf "${client_release_dir}"
sudo -n install -d -o "${APP_USER}" -g "${app_group}" -m 0755 "${client_release_dir}"
sudo -n tar --no-same-owner --no-same-permissions -xzf "${uploaded_client_file}" \
  -C "${client_release_dir}"
sudo -n chown -R "${APP_USER}:${app_group}" "${client_release_dir}"
sudo -n test -s "${client_release_dir}/index.html"
sudo -n ln -sfn "${client_release_dir}" "${APP_DIR}/client/current"

service_port=""
if sudo -n systemctl is-active --quiet new-api; then
  service_port="$(sudo -n sed -nE 's/^[[:space:]]*ExecStart=.*--port[[:space:]]+([0-9]+).*/\1/p' \
    "${service_file}" 2>/dev/null | sed -n '1p')"
  if [[ -z ${service_port} && -f ${port_file} ]]; then
    service_port="$(sudo -n cat "${port_file}")"
  fi
fi

# Keep the port already used by the running service. APP_PORT and .app-port
# are only preferences when no existing service is active.
preferred_port="${service_port:-${APP_PORT}}"
if [[ -z ${preferred_port} && -f ${port_file} ]]; then
  preferred_port="$(sudo -n cat "${port_file}")"
fi
preferred_port="${preferred_port:-3000}"
validate_value PREFERRED_PORT "${preferred_port}" '^[0-9]+$'
if (( preferred_port < 1 || preferred_port > 65535 )); then
  echo "Preferred port is outside the valid range" >&2
  exit 1
fi

# The preferred port is retained whenever possible; otherwise the first free
# port in the next 20 ports is selected. A running new-api service is never
# stopped just to probe its port.
if ! command -v ss >/dev/null 2>&1; then
  echo "The ss command is required for automatic port selection" >&2
  sudo -n systemctl start new-api 2>/dev/null || true
  exit 1
fi

selected_port=""
last_port=$((preferred_port + 19))
if (( last_port > 65535 )); then
  last_port=65535
fi
for ((candidate = preferred_port; candidate <= last_port; candidate++)); do
  if ! ss -ltnH "sport = :${candidate}" | grep -q .; then
    selected_port=${candidate}
    break
  fi
done
if [[ -z ${selected_port} ]]; then
  echo "No free application port found" >&2
  sudo -n systemctl start new-api 2>/dev/null || true
  exit 1
fi
APP_PORT=${selected_port}
printf '%s\n' "${APP_PORT}" | sudo -n tee "${port_file}" >/dev/null
sudo -n chown "${APP_USER}:${app_group}" "${port_file}"
sudo -n chmod 0644 "${port_file}"

# Keep runtime configuration beside the release directory. The content comes
# from the protected GitHub Environment secret and is never printed.
printf '%s\n' "${NEW_API_ENV}" | sudo -n tee "${env_file}" >/dev/null
sudo -n chown "${APP_USER}:${app_group}" "${env_file}"
sudo -n chmod 0600 "${env_file}"

gzip -dc "${uploaded_file}" | sudo -n tee "${release_file}" >/dev/null
sudo -n chown "${APP_USER}:${app_group}" "${release_file}"
sudo -n chmod 0755 "${release_file}"
sudo -n ln -sfn "${release_file}" "${APP_DIR}/new-api"

sudo -n tee "${service_file}" >/dev/null <<SERVICE
[Unit]
Description=New API
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=${APP_USER}
Group=${app_group}
WorkingDirectory=${APP_DIR}
EnvironmentFile=${env_file}
ExecStart=${APP_DIR}/new-api --port ${APP_PORT} --log-dir ${APP_DIR}/logs
Restart=always
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=${APP_DIR}/data ${APP_DIR}/logs

[Install]
WantedBy=multi-user.target
SERVICE

rollback_binary() {
  echo "Deployment health check failed; restoring the previous release" >&2
  if [[ -n ${previous_target} ]]; then
    sudo -n ln -sfn "${previous_target}" "${APP_DIR}/new-api"
    sudo -n systemctl restart new-api || true
  fi
}

sudo -n systemctl daemon-reload
sudo -n systemctl enable new-api
if ! sudo -n systemctl restart new-api; then
  rollback_binary
  exit 1
fi

healthy=false
for _ in {1..30}; do
  if curl --fail --silent --show-error "http://127.0.0.1:${APP_PORT}/api/status" >/dev/null; then
    healthy=true
    break
  fi
  sleep 2
done
if [[ ${healthy} != true ]]; then
  rollback_binary
  exit 1
fi

echo "========== Configure Nginx =========="
sudo -n apt-get update -y
sudo -n env DEBIAN_FRONTEND=noninteractive apt-get install -y curl nginx certbot python3-certbot-nginx
sudo -n install -d -m 0755 /etc/nginx/sites-available /etc/nginx/sites-enabled
sudo -n tee "${nginx_site}" >/dev/null <<NGINX
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};
    client_max_body_size 100m;

    location / {
        proxy_pass http://127.0.0.1:${APP_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_buffering off;
        proxy_read_timeout 1800s;
        proxy_send_timeout 1800s;
    }
}
NGINX
sudo -n ln -sfn "${nginx_site}" /etc/nginx/sites-enabled/new-api.conf
sudo -n nginx -t
sudo -n systemctl enable --now nginx
sudo -n systemctl reload nginx

echo "========== Issue or renew TLS =========="
if [[ -n ${OLD_DOMAIN} && ${OLD_DOMAIN} != "${DOMAIN}" ]]; then
  echo "Removing old TLS certificate for ${OLD_DOMAIN}"
  sudo -n certbot delete --cert-name "${OLD_DOMAIN}" -n >/dev/null 2>&1 || true
fi

if sudo -n certbot certificates 2>/dev/null | grep -Fq "Domains: ${DOMAIN}"; then
  sudo -n certbot renew --cert-name "${DOMAIN}" --quiet || true
  sudo -n certbot install --cert-name "${DOMAIN}" --nginx --redirect --non-interactive
else
  sudo -n certbot --nginx --non-interactive --agree-tos --no-eff-email \
    --email "${CERTBOT_EMAIL}" --redirect -d "${DOMAIN}"
fi

sudo -n nginx -t
sudo -n systemctl reload nginx

echo "========== Install tenant-domain host provisioner =========="
sudo -n env CERTBOT_EMAIL="${CERTBOT_EMAIL}" APP_PORT="${APP_PORT}" \
  NEW_API_APP_UNIT=new-api.service bash "${uploaded_provisioner_installer}"
sudo -n systemctl is-active --quiet new-api
sudo -n systemctl is-active --quiet new-api-tenant-domain-provisioner
sudo -n test -S /run/new-api-tenant-domain.sock
sudo -n systemctl show -p Environment --value new-api \
  | grep -Fq 'TENANT_DOMAIN_PROVISIONER_ENABLED=true'
curl --fail --silent --show-error \
  "http://127.0.0.1:${APP_PORT}/api/status" >/dev/null
curl --fail --silent --show-error --retry 10 --retry-delay 3 \
  "https://${DOMAIN}/api/status" >/dev/null

rm -f "${uploaded_file}"
rm -f "${uploaded_client_file}"
rm -rf "${uploaded_provisioner_dir}"
find "${APP_DIR}/releases" -maxdepth 1 -type f -name 'new-api-*' -printf '%T@ %p\n' \
  | sort -nr \
  | awk 'NR > 5 { sub(/^[^ ]+ /, ""); print }' \
  | xargs -r sudo -n rm -f

echo "Deployed ${RELEASE_ID} at ${DOMAIN} successfully"
