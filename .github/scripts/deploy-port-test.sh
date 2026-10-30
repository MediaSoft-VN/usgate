#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${script_dir}/deploy-port.sh"

test_dir=$(mktemp -d)
trap 'rm -rf "${test_dir}"' EXIT
cat >"${test_dir}/ss" <<'SS'
#!/usr/bin/env bash
port=${!#}
case ${port} in
  'sport = :3342')
    if [[ -n ${TEST_PID_3342:-} ]]; then
      printf 'LISTEN users:("new-api",pid=%s,fd=3)\n' "${TEST_PID_3342}"
    fi
    ;;
  'sport = :3343')
    if [[ -n ${TEST_PID_3343:-} ]]; then
      printf 'LISTEN users:("other",pid=%s,fd=3)\n' "${TEST_PID_3343}"
    fi
    ;;
esac
SS
chmod +x "${test_dir}/ss"
PATH="${test_dir}:${PATH}"
export PATH
sudo() {
  [[ ${1:-} == -n ]] || return 2
  shift
  "$@"
}

# Preserve the active service's port when its own PID owns the listener.
TEST_PID_3342=123
export TEST_PID_3342
[[ $(select_deployment_port true 3342 123) == 3342 ]]

# If another process owns the preferred port, choose the next free port.
TEST_PID_3342=456
TEST_PID_3343=789
export TEST_PID_3342 TEST_PID_3343
[[ $(select_deployment_port true 3342 123) == 3344 ]]

# If the listener owner cannot be determined, do not switch ports speculatively.
[[ $(select_deployment_port true 3342 '') == 3342 ]]

# With no active service, also skip ports occupied by other processes.
[[ $(select_deployment_port false 3342 '') == 3344 ]]
