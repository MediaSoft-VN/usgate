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

nginx_domain_server_config_files() {
  local hostname=$1
  local snippet_file=$2
  local config_dump

  config_dump=$(sudo -n nginx -T 2>/dev/null) || return 2
  printf '%s\n' "${config_dump}" | awk -v hostname="${hostname}" -v snippet_file="${snippet_file}" '
    function name_matches(name, host, suffix) {
      name = tolower(name)
      if (name == host) return 1
      if (substr(name, 1, 2) == "*.") {
        suffix = substr(name, 2)
        return length(host) > length(suffix) && substr(host, length(host) - length(suffix) + 1) == suffix
      }
      if (substr(name, 1, 1) == ".") {
        suffix = name
        return host == substr(suffix, 2) || (length(host) > length(suffix) && substr(host, length(host) - length(suffix) + 1) == suffix)
      }
      return 0
    }
    BEGIN { hostname = tolower(hostname); depth = 0 }
    /^# configuration file / {
      source = $0
      sub(/^# configuration file /, "", source)
      sub(/:$/, "", source)
      next
    }
    {
      line = $0
      sub(/#.*/, "", line)
      if (!in_server && line ~ /^[[:space:]]*server[[:space:]]*\{/) {
        in_server = 1
        server_depth = depth + 1
        server_source = source
        server_matches = 0
        server_has_proxy = 0
      }
      if (in_server) {
        if (line ~ /^[[:space:]]*server_name[[:space:]]/) {
          names = line
          sub(/^[[:space:]]*server_name[[:space:]]+/, "", names)
          sub(/;.*/, "", names)
          count = split(names, name_list, /[[:space:]]+/)
          for (i = 1; i <= count; i++) if (name_matches(name_list[i], hostname)) server_matches = 1
        }
        if (line ~ /^[[:space:]]*proxy_pass[[:space:]]+http:\/\/127\.0\.0\.1:[0-9]+/ ||
          index(line, "include " snippet_file ";") > 0) server_has_proxy = 1
      }
      opens = gsub(/\{/, "{", line)
      closes = gsub(/\}/, "}", line)
      depth += opens - closes
      if (in_server && depth < server_depth) {
        if (server_matches && server_has_proxy && !seen[server_source]++) {
          print server_source
          found = 1
        }
        in_server = 0
      }
    }
    END { if (!found) exit 1 }
  '
}

rewrite_nginx_domain_proxy() {
  local config_file=$1
  local hostname=$2
  local app_port=$3
  local snippet_file=$4

  sudo -n python3 - "${config_file}" "${hostname}" "${app_port}" "${snippet_file}" <<'PY'
import os
import re
import stat
import sys
import tempfile
from pathlib import Path

config_path, hostname, app_port, snippet_file = sys.argv[1:]
config_path = Path(config_path).resolve(strict=True)
lines = config_path.read_text(encoding="utf-8").splitlines(keepends=True)
hostname = hostname.lower()

def host_matches(name):
    name = name.lower()
    if name == hostname:
        return True
    if name.startswith("*."):
        suffix = name[1:]
        return hostname.endswith(suffix) and len(hostname) > len(suffix)
    if name.startswith("."):
        return hostname == name[1:] or hostname.endswith(name)
    return False

def rewrite_server(block):
    names = []
    for line in block:
        clean = line.split("#", 1)[0]
        match = re.match(r"\s*server_name\s+(.+?);", clean)
        if match:
            names.extend(match.group(1).split())
    if not any(host_matches(name) for name in names):
        return block, False, False

    for line in block:
        clean = line.split("#", 1)[0]
        match = re.match(r"\s*proxy_pass\s+http://127\.0\.0\.1:([0-9]+);", clean)
        if match:
            domain_ports.add(match.group(1))

    updated = []
    found_proxy = False
    changed = False
    proxy_pattern = re.compile(r"^(\s*)proxy_pass\s+http://127\.0\.0\.1:[0-9]+;([ \t]*(?:#.*)?(?:\r?\n)?)$")
    include_pattern = re.compile(r"^\s*include\s+" + re.escape(snippet_file) + r"\s*;")
    for line in block:
        match = proxy_pattern.match(line)
        if match:
            found_proxy = True
            updated.append(f"{match.group(1)}include {snippet_file};{match.group(2)}")
            changed = True
        else:
            if include_pattern.match(line):
                found_proxy = True
            updated.append(line)
    return updated, found_proxy, changed

output = []
depth = 0
block = None
server_depth = 0
found_proxy = False
changed = False
domain_ports = set()
for line in lines:
    clean = line.split("#", 1)[0]
    if block is None and re.match(r"\s*server\s*\{", clean):
        block = []
        server_depth = depth + 1
    if block is not None:
        block.append(line)
    else:
        output.append(line)
    depth += clean.count("{") - clean.count("}")
    if block is not None and depth < server_depth:
        block, server_found_proxy, server_changed = rewrite_server(block)
        output.extend(block)
        found_proxy = found_proxy or server_found_proxy
        changed = changed or server_changed
        block = None

if block is not None:
    raise SystemExit("Unterminated Nginx server block")
if len(domain_ports) > 1:
    raise SystemExit("Multiple loopback proxy ports are configured for the deployment domain")
if not found_proxy:
    raise SystemExit("No supported loopback proxy directive found for deployment domain")
if changed:
    metadata = config_path.stat()
    descriptor, temporary_path = tempfile.mkstemp(prefix=f".{config_path.name}.", dir=config_path.parent)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8", newline="") as temporary:
            temporary.writelines(output)
        os.chmod(temporary_path, stat.S_IMODE(metadata.st_mode))
        os.chown(temporary_path, metadata.st_uid, metadata.st_gid)
        os.replace(temporary_path, config_path)
    finally:
        if os.path.exists(temporary_path):
            os.unlink(temporary_path)
PY
}

ensure_nginx_domain_vhost() {
  local hostname=$1
  local app_port=$2
  local site_file=$3
  local snippet_file=$4
  local inspection_status
  local config_files
  local config_file

  if nginx_server_name_configured "${hostname}"; then
    config_files=$(nginx_domain_server_config_files "${hostname}" "${snippet_file}") || {
      echo "Could not find the active Nginx configuration for ${hostname}" >&2
      return 1
    }
    sudo -n install -d -m 0755 "$(dirname "${snippet_file}")"
    printf 'proxy_pass http://127.0.0.1:%s;\n' "${app_port}" | sudo -n tee "${snippet_file}" >/dev/null
    while IFS= read -r config_file; do
      rewrite_nginx_domain_proxy "${config_file}" "${hostname}" "${app_port}" "${snippet_file}"
    done <<<"${config_files}"
    echo "Reusing existing Nginx server configuration for ${hostname} with port ${app_port}"
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

  sudo -n install -d -m 0755 "$(dirname "${snippet_file}")"
  printf 'proxy_pass http://127.0.0.1:%s;\n' "${app_port}" | sudo -n tee "${snippet_file}" >/dev/null
  sudo -n install -d -m 0755 "$(dirname "${site_file}")"
  sudo -n tee "${site_file}" >/dev/null <<NGINX
# Managed by new-api native deployment
server {
    listen 80;
    listen [::]:80;
    server_name ${hostname};
    client_max_body_size 100m;

    location / {
        include ${snippet_file};
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
