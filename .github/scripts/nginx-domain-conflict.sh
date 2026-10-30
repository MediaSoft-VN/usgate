#!/usr/bin/env bash

# Returns 0 when nginx -T contains a server_name that already claims hostname,
# 1 when it is unused, and 2 when the active Nginx configuration cannot be read.
nginx_server_name_configured() {
  local hostname=$1
  local ignored_source=${2:-}
  local config_dump

  config_dump=$(sudo -n nginx -T 2>/dev/null) || return 2
  printf '%s\n' "${config_dump}" | awk -v hostname="${hostname}" -v ignored_source="${ignored_source}" '
    BEGIN { hostname = tolower(hostname) }
    /^# configuration file / {
      source = $0
      sub(/^# configuration file /, "", source)
      sub(/:$/, "", source)
      next
    }
    {
      if (source == ignored_source) next
      line = $0
      sub(/#.*/, "", line)
      if (line !~ /^[[:space:]]*server_name[[:space:]]/) next
      sub(/^[[:space:]]*server_name[[:space:]]+/, "", line)
      sub(/;.*/, "", line)
      count = split(line, names, /[[:space:]]+/)
      for (i = 1; i <= count; i++) {
        name = tolower(names[i])
        if (name == hostname) found = 1
        if (substr(name, 1, 2) == "*.") {
          suffix = substr(name, 2)
          if (length(hostname) > length(suffix) && substr(hostname, length(hostname) - length(suffix) + 1) == suffix) found = 1
        }
        if (substr(name, 1, 1) == ".") {
          suffix = name
          if (hostname == substr(suffix, 2) || (length(hostname) > length(suffix) && substr(hostname, length(hostname) - length(suffix) + 1) == suffix)) found = 1
        }
      }
    }
    END { exit found ? 0 : 1 }
  '
}

ensure_nginx_domain_vhost() {
  local hostname=$1
  local app_port=$2
  local site_file=$3
  local inspection_status

  if nginx_server_name_configured "${hostname}"; then
    echo "Reusing existing Nginx server configuration for ${hostname}"
    return 0
  else
    inspection_status=$?
  fi
  if [[ ${inspection_status} != 1 ]]; then
    echo "Could not inspect active Nginx configuration" >&2
    return 1
  fi

  if [[ -e ${site_file} ]] && ! grep -Fq '# Managed by new-api native deployment' "${site_file}"; then
    echo "Refusing to overwrite an unmanaged Nginx configuration" >&2
    return 1
  fi

  sudo -n install -d -m 0755 "$(dirname "${site_file}")"
  sudo -n tee "${site_file}" >/dev/null <<NGINX
# Managed by new-api native deployment
server {
    listen 80;
    listen [::]:80;
    server_name ${hostname};
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
  echo "Created Nginx server configuration for ${hostname}"
}
