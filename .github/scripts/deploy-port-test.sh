#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${script_dir}/deploy-port.sh"

test_dir=$(mktemp -d)
trap 'rm -rf "${test_dir}"' EXIT
cat >"${test_dir}/ss" <<'SS'
#!/usr/bin/env bash
port=${*: -1}
case ${port} in
  'sport = :3342'|'sport = :3343') printf 'LISTEN\n' ;;
esac
SS
chmod +x "${test_dir}/ss"
PATH="${test_dir}:${PATH}"
export PATH

# The current service owns 3342. Keep that configured port across restart.
[[ $(select_deployment_port true 3342) == 3342 ]]

# Without an active service, skip occupied ports and choose the first free one.
[[ $(select_deployment_port false 3342) == 3344 ]]
