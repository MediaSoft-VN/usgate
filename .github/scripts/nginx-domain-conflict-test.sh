#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${script_dir}/nginx-domain-conflict.sh"
source "${script_dir}/deploy-port.sh"

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
cat >"${dump_file}" <<'CONFIG'
# configuration file /etc/nginx/conf.d/tenant.conf:
server {
    server_name App.Example.Com;
    location / {
        proxy_pass http://127.0.0.1:3342;
    }
}
# configuration file /etc/nginx/conf.d/wildcard.conf:
server {
    server_name *.example.net;
}
CONFIG

nginx_server_name_configured app.example.com
nginx_server_name_configured api.example.net
[[ $(nginx_domain_loopback_upstream_port app.example.com) == 3342 ]]
[[ $(select_deployment_port true "$(nginx_domain_loopback_upstream_port app.example.com)") == 3342 ]]
if nginx_server_name_configured unused.example.org; then
  echo 'unexpected Nginx hostname collision' >&2
  exit 1
fi
if nginx_server_name_configured app.example.com /etc/nginx/conf.d/tenant.conf; then
  echo 'managed config was not ignored' >&2
  exit 1
fi

printf '%s\n' '# configuration file /etc/nginx/conf.d/other.conf:' 'server {' '    server_name app.example.com;' '}' >>"${dump_file}"
nginx_server_name_configured app.example.com /etc/nginx/conf.d/tenant.conf

reused_site="${test_dir}/reused.conf"
ensure_nginx_domain_vhost app.example.com 3000 "${reused_site}"
[[ ! -e ${reused_site} ]]

created_site="${test_dir}/created.conf"
ensure_nginx_domain_vhost fresh.example.org 4321 "${created_site}"
grep -Fq '# Managed by new-api native deployment' "${created_site}"
grep -Fq 'server_name fresh.example.org;' "${created_site}"
grep -Fq 'proxy_pass http://127.0.0.1:4321;' "${created_site}"

unmanaged_site="${test_dir}/unmanaged.conf"
printf '%s\n' 'server { server_name someone-else.example.org; }' >"${unmanaged_site}"
if ensure_nginx_domain_vhost unused.example.org 3000 "${unmanaged_site}"; then
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
if ensure_nginx_domain_vhost unavailable.example.org 3000 "${test_dir}/unavailable.conf"; then
  echo 'Nginx configuration was created without successful inspection' >&2
  exit 1
fi
[[ ! -e ${test_dir}/unavailable.conf ]]
