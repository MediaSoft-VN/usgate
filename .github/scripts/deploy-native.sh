#!/usr/bin/env bash

set -Eeuo pipefail

: "${APP_DIR:=/opt/new-api}"
: "${APP_PORT:=3000}"
: "${APP_USER:?APP_USER is required}"
: "${DOMAIN:?DOMAIN is required}"
: "${CERTBOT_EMAIL:?CERTBOT_EMAIL is required}"
: "${RELEASE_ID:?RELEASE_ID is required}"
: "${EXPECTED_SHA256:?EXPECTED_SHA256 is required}"
: "${SSH_USER:?SSH_USER is required}"
: "${OLD_DOMAIN:=}"

uploaded_file="/tmp/new-api-${RELEASE_ID}"
release_file="${APP_DIR}/releases/new-api-${RELEASE_ID}"
service_file=/etc/systemd/system/new-api.service
nginx_site=/etc/nginx/sites-available/new-api.conf
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
validate_value APP_PORT "${APP_PORT}" '^[0-9]+$'
validate_value APP_USER "${APP_USER}" '^[A-Za-z_][A-Za-z0-9_-]*$'
validate_value DOMAIN "${DOMAIN}" '^[A-Za-z0-9.-]+$'
validate_value CERTBOT_EMAIL "${CERTBOT_EMAIL}" '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$'
validate_value RELEASE_ID "${RELEASE_ID}" '^[0-9a-f]{40}$'
validate_value EXPECTED_SHA256 "${EXPECTED_SHA256}" '^[0-9a-f]{64}$'
validate_value SSH_USER "${SSH_USER}" '^[A-Za-z_][A-Za-z0-9_-]*$'
if [[ -n ${OLD_DOMAIN} ]]; then
  validate_value OLD_DOMAIN "${OLD_DOMAIN}" '^[A-Za-z0-9.-]+$'
fi

if [[ ! -f ${uploaded_file} ]]; then
  echo "Uploaded release was not found" >&2
  exit 1
fi

actual_sha256="$(sha256sum "${uploaded_file}" | cut -d' ' -f1)"
if [[ ${actual_sha256} != "${EXPECTED_SHA256}" ]]; then
  echo "Release checksum verification failed" >&2
  exit 1
fi

app_group="$(id -gn "${APP_USER}")"
sudo -n install -d -o "${APP_USER}" -g "${app_group}" -m 0755 \
  "${APP_DIR}" "${APP_DIR}/releases" "${APP_DIR}/logs" "${APP_DIR}/data"
sudo -n install -o "${APP_USER}" -g "${app_group}" -m 0755 \
  "${uploaded_file}" "${release_file}"
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
EnvironmentFile=-/etc/new-api.env
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
curl --fail --silent --show-error --retry 10 --retry-delay 3 \
  "https://${DOMAIN}/api/status" >/dev/null

rm -f "${uploaded_file}"
find "${APP_DIR}/releases" -maxdepth 1 -type f -name 'new-api-*' -printf '%T@ %p\n' \
  | sort -nr \
  | awk 'NR > 5 { sub(/^[^ ]+ /, ""); print }' \
  | xargs -r sudo -n rm -f

echo "Deployed ${RELEASE_ID} at ${DOMAIN} successfully"
