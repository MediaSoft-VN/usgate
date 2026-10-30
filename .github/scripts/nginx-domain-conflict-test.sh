#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${script_dir}/nginx-domain-conflict.sh"

test_dir=$(mktemp -d)
trap 'rm -rf "${test_dir}"' EXIT
cat >"${test_dir}/nginx" <<'NGINX'
#!/usr/bin/env bash
[[ ${1:-} == -T ]] || exit 2
cat "${NGINX_TEST_DUMP}"
NGINX
chmod +x "${test_dir}/nginx"
PATH="${test_dir}:${PATH}"
export PATH

sudo() {
  [[ ${1:-} == -n ]] || return 2
  shift
  "$@"
}

dump_file="${test_dir}/nginx.dump"
export NGINX_TEST_DUMP=${dump_file}
active_site="${test_dir}/app.example.com.conf"
cat >"${active_site}" <<'CONFIG'
server {
    server_name App.Example.Com;
    location ^~ /.well-known/acme-challenge/ {
        root /var/www/html;
        try_files $uri =404;
    }
    location / {
        proxy_pass http://127.0.0.1:3342;
        proxy_read_timeout 1800s;
    }
    listen 443 ssl;
    ssl_certificate /etc/letsencrypt/live/app.example.com/fullchain.pem;
}
server {
    server_name app.example.com;
    listen 80;
    return 301 https://$host$request_uri;
}
CONFIG
wildcard_site="${test_dir}/wildcard.conf"
cat >"${wildcard_site}" <<'CONFIG'
server {
    server_name *.example.net;
}
CONFIG
printf '# configuration file %s:\n' "${active_site}" >"${dump_file}"
cat "${active_site}" >>"${dump_file}"
printf '# configuration file %s:\n' "${wildcard_site}" >>"${dump_file}"
cat "${wildcard_site}" >>"${dump_file}"

nginx_server_name_configured app.example.com
nginx_server_name_configured api.example.net
if nginx_server_name_configured unused.example.org; then
  echo 'unexpected Nginx hostname collision' >&2
  exit 1
fi
if nginx_server_name_configured app.example.com "${active_site}"; then
  echo 'managed config was not ignored' >&2
  exit 1
fi

reused_site="${test_dir}/reused.conf"
snippet_file="${test_dir}/snippets/new-api-proxy.conf"
ensure_nginx_domain_vhost app.example.com 3343 "${reused_site}" "${snippet_file}"
[[ ! -e ${reused_site} ]]
grep -Fq 'include '"${snippet_file}"';' "${active_site}"
grep -Fq 'root /var/www/html;' "${active_site}"
grep -Fq 'ssl_certificate /etc/letsencrypt/live/app.example.com/fullchain.pem;' "${active_site}"
! grep -Fq 'proxy_pass http://127.0.0.1:3342;' "${active_site}"
grep -Fq 'proxy_pass http://127.0.0.1:3343;' "${snippet_file}"

created_site="${test_dir}/created.conf"
ensure_nginx_domain_vhost fresh.example.org 4321 "${created_site}" "${snippet_file}"
grep -Fq '# Managed by new-api native deployment' "${created_site}"
grep -Fq 'server_name fresh.example.org;' "${created_site}"
grep -Fq 'include '"${snippet_file}"';' "${created_site}"
grep -Fq 'proxy_pass http://127.0.0.1:4321;' "${snippet_file}"

unmanaged_site="${test_dir}/unmanaged.conf"
printf '%s\n' 'server { server_name someone-else.example.org; }' >"${unmanaged_site}"
if ensure_nginx_domain_vhost unused.example.org 3000 "${unmanaged_site}" "${snippet_file}"; then
  echo 'unmanaged Nginx configuration was overwritten' >&2
  exit 1
fi

cat >"${test_dir}/nginx" <<'NGINX'
#!/usr/bin/env bash
exit 1
NGINX
chmod +x "${test_dir}/nginx"
if nginx_server_name_configured unused.example.org; then
  echo 'Nginx inspection failure was treated as a successful match' >&2
  exit 1
else
  [[ $? == 2 ]]
fi
if ensure_nginx_domain_vhost unavailable.example.org 3000 "${test_dir}/unavailable.conf" "${snippet_file}"; then
  echo 'Nginx configuration was created without successful inspection' >&2
  exit 1
fi
[[ ! -e ${test_dir}/unavailable.conf ]]
