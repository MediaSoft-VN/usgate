#!/usr/bin/env bash

set -Eeuo pipefail
umask 077

readonly config_file="${TENANT_DOMAIN_PROVISIONER_CONFIG:-/etc/new-api/tenant-domain-provisioner.env}"
readonly marker='# Managed by new-api tenant domain provisioner'

if [[ ! -f ${config_file} ]]; then
  exit 1
fi
# This root-owned file is created by install-tenant-domain-provisioner.sh.
# shellcheck source=/dev/null
source "${config_file}"
site_dir=${NGINX_SITES_AVAILABLE:-/etc/nginx/sites-available}
enabled_dir=${NGINX_SITES_ENABLED:-/etc/nginx/sites-enabled}
nginx_bin=${NGINX_BIN:-nginx}
systemctl_bin=${SYSTEMCTL_BIN:-systemctl}
certbot_bin=${CERTBOT_BIN:-certbot}
openssl_bin=${OPENSSL_BIN:-openssl}
lock_file=${DOMAIN_LOCK_FILE:-/run/lock/new-api-tenant-domain.lock}
cert_renewal_dir=${CERT_RENEWAL_DIR:-/etc/letsencrypt/renewal}
cert_live_dir=${CERT_LIVE_DIR:-/etc/letsencrypt/live}
cert_manifest_dir=${CERT_MANIFEST_DIR:-/etc/new-api/partner-domain-certificates}
acme_webroot_dir=${ACME_WEBROOT_DIR:-/var/www/html}

if [[ $# != 2 ]]; then
  exit 2
fi
operation=$1
hostname=$2
if [[ ${operation} != check && ${operation} != provision && ${operation} != cleanup ]] ||
  [[ ! ${hostname} =~ ^[a-z0-9]([a-z0-9.-]{0,251}[a-z0-9])?$ ]] ||
  [[ ${hostname} == *..* || ${hostname} == *.-* || ${hostname} == -* ]]; then
  exit 2
fi
if [[ ! ${APP_PORT:-} =~ ^[0-9]{1,5}$ ]] || (( APP_PORT < 1 || APP_PORT > 65535 )) ||
  [[ ! ${CERTBOT_EMAIL:-} =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]; then
  exit 1
fi

exec 9>"${lock_file}"
flock -x 9

readonly cert_name="partner-${hostname}"
readonly cert_renewal_file="${cert_renewal_dir}/${cert_name}.conf"
readonly cert_file="${cert_live_dir}/${cert_name}/cert.pem"
readonly cert_manifest="${cert_manifest_dir}/${cert_name}"
readonly site_file="${site_dir}/partner-${hostname}.conf"
readonly enabled_file="${enabled_dir}/partner-${hostname}.conf"

site_is_managed() {
  [[ -f ${site_file} ]] && grep -Fqx "${marker}" "${site_file}"
}

nginx_config_has_hostname() {
  local effective_config owned_config=""
  effective_config=$("${nginx_bin}" -T 2>/dev/null) || return 2
  if site_is_managed && [[ -L ${enabled_file} ]] &&
    [[ $(readlink "${enabled_file}") == "${site_file}" ]]; then
    owned_config=${enabled_file}
  fi
  printf '%s\n' "${effective_config}" | awk -v hostname="${hostname}" -v owned_config="${owned_config}" '
    /^# configuration file / {
      source = $0
      sub(/^# configuration file /, "", source)
      sub(/:$/, "", source)
      next
    }
    {
      line = $0
      sub(/#.*/, "", line)
      if (owned_config != "" && source == owned_config) next
      if (line !~ /^[[:space:]]*server_name[[:space:]]/) next
      sub(/^[[:space:]]*server_name[[:space:]]+/, "", line)
      sub(/;.*/, "", line)
      count = split(line, names, /[[:space:]]+/)
      for (i = 1; i <= count; i++) {
        if (names[i] == hostname) found = 1
        if (substr(names[i], 1, 2) == "*.") {
          suffix = substr(names[i], 2)
          if (length(hostname) > length(suffix) && substr(hostname, length(hostname) - length(suffix) + 1) == suffix) found = 1
        }
        if (substr(names[i], 1, 1) == ".") {
          suffix = names[i]
          if (hostname == substr(suffix, 2) || (length(hostname) > length(suffix) && substr(hostname, length(hostname) - length(suffix) + 1) == suffix)) found = 1
        }
      }
    }
    END { exit found ? 0 : 1 }
  '
}

ensure_hostname_available() {
  local result
  if nginx_config_has_hostname; then
    return 1
  else
    result=$?
    [[ ${result} == 1 ]]
  fi
}

cert_is_managed() {
  [[ -f ${cert_renewal_file} && -f ${cert_manifest} && $(<"${cert_manifest}") == "${hostname}" ]] || return 1
  grep -Fqx 'authenticator = webroot' "${cert_renewal_file}" || return 1
  [[ ! -f ${cert_file} ]] || cert_matches_hostname
}

cert_matches_hostname() {
  "${openssl_bin}" x509 -in "${cert_file}" -noout -ext subjectAltName 2>/dev/null |
    grep -Eo 'DNS:[^,[:space:]]+' | sed 's/^DNS://' | sort -u |
    diff -u <(printf '%s\n' "${hostname}") - >/dev/null 2>&1
}

restore_previous_site() {
  if [[ ${had_previous} == true ]]; then
    cp -p "${previous_file}" "${site_file}"
  else
    rm -f "${site_file}"
  fi
  if [[ ${had_enabled} != true ]]; then
    rm -f "${enabled_file}"
  fi
  "${nginx_bin}" -t >/dev/null 2>&1 && "${systemctl_bin}" reload nginx >/dev/null 2>&1 || true
  rm -f "${previous_file}"
}

case ${operation} in
  check)
    ensure_hostname_available
    ;;
  provision)
    ensure_hostname_available || exit 1
    app_port=${APP_PORT}
    if [[ -e ${cert_manifest} ]] &&
      { [[ ! -f ${cert_manifest} ]] || [[ $(<"${cert_manifest}") != "${hostname}" ]]; }; then
      exit 1
    fi
    if [[ -f ${cert_renewal_file} ]] && ! cert_is_managed; then
      exit 1
    fi
    had_certificate=false
    if [[ -f ${cert_renewal_file} ]]; then
      had_certificate=true
    fi
    install -d -m 0755 "${acme_webroot_dir}/.well-known/acme-challenge"
    install -d -m 0755 "${site_dir}" "${enabled_dir}"
    if [[ -e ${site_file} ]] && ! site_is_managed; then
      exit 1
    fi
    if [[ -e ${enabled_file} && ! -L ${enabled_file} ]] ||
      [[ -L ${enabled_file} && $(readlink "${enabled_file}") != "${site_file}" ]]; then
      exit 1
    fi
    previous_file=$(mktemp)
    had_previous=false
    had_enabled=false
    if [[ -e ${site_file} ]]; then
      cp -p "${site_file}" "${previous_file}"
      had_previous=true
    fi
    if [[ -L ${enabled_file} ]]; then
      had_enabled=true
    fi
    if [[ ${had_previous} != true ]]; then
      temp_file=$(mktemp "${site_dir}/.partner-domain.XXXXXX")
      cat >"${temp_file}" <<NGINX
${marker}
server {
    listen 80;
    listen [::]:80;
    server_name ${hostname};
    client_max_body_size 100m;

    location /.well-known/acme-challenge/ {
        root ${acme_webroot_dir};
    }
    location / {
        return 503;
    }
}
NGINX
      chmod 0644 "${temp_file}"
      mv -f "${temp_file}" "${site_file}"
    fi
    ln -sfn "${site_file}" "${enabled_file}"
    if ! "${nginx_bin}" -t >/dev/null 2>&1 || ! "${systemctl_bin}" reload nginx >/dev/null 2>&1; then
      restore_previous_site
      exit 1
    fi
    install -d -m 0700 "${cert_manifest_dir}"
    manifest_tmp=$(mktemp "${cert_manifest_dir}/.partner-domain.XXXXXX")
    printf '%s\n' "${hostname}" >"${manifest_tmp}"
    chmod 0600 "${manifest_tmp}"
    mv -f "${manifest_tmp}" "${cert_manifest}"
    if ! "${certbot_bin}" certonly --webroot --webroot-path "${acme_webroot_dir}" \
      --non-interactive --agree-tos --no-eff-email --email "${CERTBOT_EMAIL}" \
      --cert-name "${cert_name}" -d "${hostname}" \
      >/dev/null 2>&1; then
      restore_previous_site
      if [[ ${had_certificate} != true ]]; then
        rm -f "${cert_manifest}"
      fi
      exit 1
    fi
    if [[ ! -f ${cert_renewal_file} ]] || ! grep -Fqx 'authenticator = webroot' "${cert_renewal_file}" || ! cert_matches_hostname; then
      restore_previous_site
      if [[ ${had_certificate} != true ]]; then
        rm -f "${cert_manifest}"
      fi
      exit 1
    fi
    final_file=$(mktemp "${site_dir}/.partner-domain.XXXXXX")
    cat >"${final_file}" <<NGINX
${marker}
server {
    listen 80;
    listen [::]:80;
    server_name ${hostname};
    client_max_body_size 100m;

    location /.well-known/acme-challenge/ {
        root ${acme_webroot_dir};
    }
    location / {
        return 301 https://\$host\$request_uri;
    }
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name ${hostname};
    ssl_certificate /etc/letsencrypt/live/${cert_name}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${cert_name}/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    client_max_body_size 100m;

    location / {
        proxy_pass http://127.0.0.1:${app_port};
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
    chmod 0644 "${final_file}"
    mv -f "${final_file}" "${site_file}"
    if ! "${nginx_bin}" -t >/dev/null 2>&1 || ! "${systemctl_bin}" reload nginx >/dev/null 2>&1; then
      restore_previous_site
      exit 1
    fi
    rm -f "${previous_file}"
    curl --fail --silent --show-error --retry 5 --retry-delay 1 \
      --resolve "${hostname}:443:127.0.0.1" "https://${hostname}/api/status" \
      >/dev/null 2>&1 || exit 1
    ;;
  cleanup)
    if [[ -e ${cert_manifest} ]] &&
      { [[ ! -f ${cert_manifest} ]] || [[ $(<"${cert_manifest}") != "${hostname}" ]]; }; then
      exit 1
    fi
    if [[ -f ${cert_renewal_file} ]] && ! cert_is_managed; then
      exit 1
    fi
    if [[ -e ${site_file} ]]; then
      if ! site_is_managed; then
        exit 1
      fi
    fi
    if [[ -e ${enabled_file} && ! -L ${enabled_file} ]] ||
      [[ -L ${enabled_file} && $(readlink "${enabled_file}") != "${site_file}" ]]; then
      exit 1
    fi
    backup_dir=""
    if [[ -e ${site_file} || -L ${enabled_file} ]]; then
      backup_dir=$(mktemp -d)
      if [[ -e ${site_file} ]]; then cp -p "${site_file}" "${backup_dir}/site"; fi
      if [[ -L ${enabled_file} ]]; then cp -P "${enabled_file}" "${backup_dir}/enabled"; fi
      rm -f "${enabled_file}" "${site_file}"
      if ! "${nginx_bin}" -t >/dev/null 2>&1; then
        if [[ -f ${backup_dir}/site ]]; then cp -p "${backup_dir}/site" "${site_file}"; fi
        if [[ -L ${backup_dir}/enabled ]]; then cp -P "${backup_dir}/enabled" "${enabled_file}"; fi
        rm -rf "${backup_dir}"
        exit 1
      fi
      if ! "${systemctl_bin}" reload nginx >/dev/null 2>&1; then
        if [[ -f ${backup_dir}/site ]]; then cp -p "${backup_dir}/site" "${site_file}"; fi
        if [[ -L ${backup_dir}/enabled ]]; then cp -P "${backup_dir}/enabled" "${enabled_file}"; fi
        "${nginx_bin}" -t >/dev/null 2>&1 && "${systemctl_bin}" reload nginx >/dev/null 2>&1 || true
        rm -rf "${backup_dir}"
        exit 1
      fi
    fi
    if [[ -f ${cert_renewal_file} ]]; then
      if ! "${certbot_bin}" delete --cert-name "${cert_name}" --non-interactive >/dev/null 2>&1; then
        if [[ -f ${cert_renewal_file} && -n ${backup_dir} ]]; then
          if [[ -f ${backup_dir}/site ]]; then cp -p "${backup_dir}/site" "${site_file}"; fi
          if [[ -L ${backup_dir}/enabled ]]; then cp -P "${backup_dir}/enabled" "${enabled_file}"; fi
          "${nginx_bin}" -t >/dev/null 2>&1 && "${systemctl_bin}" reload nginx >/dev/null 2>&1 || true
        fi
        [[ -z ${backup_dir} ]] || rm -rf "${backup_dir}"
        exit 1
      fi
    fi
    [[ -z ${backup_dir} ]] || rm -rf "${backup_dir}"
    rm -f "${cert_manifest}"
    ;;
esac
