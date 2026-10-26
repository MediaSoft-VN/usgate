#!/usr/bin/env bash

set -Eeuo pipefail

if (( EUID != 0 )); then
  echo "Run this installer as root" >&2
  exit 1
fi

: "${CERTBOT_EMAIL:?CERTBOT_EMAIL is required}"
: "${APP_PORT:=3000}"
: "${NEW_API_APP_UNIT:=new-api.service}"

if [[ ! ${NEW_API_APP_UNIT} =~ ^[A-Za-z0-9_.@:-]+\.service$ ]] ||
  [[ ! ${APP_PORT} =~ ^[0-9]{1,5}$ ]] || (( APP_PORT < 1 || APP_PORT > 65535 )) ||
  [[ ! ${CERTBOT_EMAIL} =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]; then
  echo "Invalid provisioner configuration" >&2
  exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if ! command -v nginx >/dev/null || ! command -v certbot >/dev/null ||
  ! command -v curl >/dev/null || ! command -v python3 >/dev/null ||
  ! command -v flock >/dev/null || ! command -v systemctl >/dev/null; then
  echo "Install Nginx, Certbot, curl, Python 3, and systemd before running this installer" >&2
  exit 1
fi
if ! systemctl is-active --quiet nginx; then
  echo "Start Nginx before enabling partner domain provisioning" >&2
  exit 1
fi

app_unit_present=false
if systemctl cat "${NEW_API_APP_UNIT}" >/dev/null 2>&1; then
  app_unit_present=true
  app_user="$(systemctl show -p User --value "${NEW_API_APP_UNIT}")"
  app_user="${app_user:-root}"
  app_group="$(systemctl show -p Group --value "${NEW_API_APP_UNIT}")"
  app_group="${app_group:-$(id -gn "${app_user}")}"
  app_uid="$(id -u "${app_user}")"
else
  app_uid=${APP_UID:-0}
  app_group=${APP_GROUP:-root}
  if [[ ! ${app_uid} =~ ^[0-9]+$ ]] || ! getent group "${app_group}" >/dev/null; then
    echo "Set APP_UID and APP_GROUP to the API process identity" >&2
    exit 1
  fi
fi

install -d -o root -g root -m 0755 /etc/new-api /usr/local/libexec
printf 'APP_PORT=%q\nCERTBOT_EMAIL=%q\n' "${APP_PORT}" "${CERTBOT_EMAIL}" \
  >/etc/new-api/tenant-domain-provisioner.env
chown root:root /etc/new-api/tenant-domain-provisioner.env
chmod 0600 /etc/new-api/tenant-domain-provisioner.env
install -o root -g root -m 0700 "${script_dir}/tenant-domain-host.sh" \
  /usr/local/sbin/new-api-tenant-domain
install -o root -g root -m 0755 "${script_dir}/tenant-domain-socket.py" \
  /usr/local/libexec/new-api-tenant-domain-socket.py

install -d -m 0755 /etc/letsencrypt/renewal-hooks/deploy
cat >/etc/letsencrypt/renewal-hooks/deploy/new-api-nginx-reload <<'HOOK'
#!/usr/bin/env bash
set -Eeuo pipefail
/usr/sbin/nginx -t
/usr/bin/systemctl reload nginx
HOOK
chown root:root /etc/letsencrypt/renewal-hooks/deploy/new-api-nginx-reload
chmod 0755 /etc/letsencrypt/renewal-hooks/deploy/new-api-nginx-reload

cat >/etc/systemd/system/new-api-tenant-domain-provisioner.service <<UNIT
[Unit]
Description=New API partner domain provisioner
Wants=network-online.target
After=network-online.target nginx.service
Requires=nginx.service

[Service]
Type=simple
User=root
Group=root
Environment=NEW_API_DOMAIN_APP_UID=${app_uid}
Environment=NEW_API_DOMAIN_APP_GROUP=${app_group}
ExecStartPre=/usr/bin/rm -f /run/new-api-tenant-domain.sock
ExecStart=/usr/local/libexec/new-api-tenant-domain-socket.py
Restart=on-failure
RestartSec=2
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=/run /etc/new-api /etc/nginx/sites-available /etc/nginx/sites-enabled /etc/letsencrypt /var/www/html /var/lib/letsencrypt /var/log/letsencrypt

[Install]
WantedBy=multi-user.target
UNIT

if [[ ${app_unit_present} == true ]]; then
  install -d -m 0755 "/etc/systemd/system/${NEW_API_APP_UNIT}.d"
  cat >"/etc/systemd/system/${NEW_API_APP_UNIT}.d/tenant-domain-provisioner.conf" <<'UNIT'
[Service]
Environment=TENANT_DOMAIN_PROVISIONER_ENABLED=true
UNIT
fi

systemctl daemon-reload
systemctl enable --now new-api-tenant-domain-provisioner.service
if [[ ${app_unit_present} == true ]]; then
  systemctl restart "${NEW_API_APP_UNIT}"
fi

echo "Tenant domain host provisioner installed and enabled"
