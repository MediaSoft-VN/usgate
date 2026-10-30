#!/usr/bin/env bash

# Selects the port for the deployment. An active service already owns its port,
# so that port must not be rejected as unavailable by the ss probe.
select_deployment_port() {
  local service_active=$1
  local preferred_port=$2

  if [[ ${service_active} == true ]]; then
    printf '%s\n' "${preferred_port}"
    return 0
  fi

  command -v ss >/dev/null 2>&1 || return 2

  local last_port=$((preferred_port + 19))
  if (( last_port > 65535 )); then
    last_port=65535
  fi
  for ((candidate = preferred_port; candidate <= last_port; candidate++)); do
    if ! ss -ltnH "sport = :${candidate}" | grep -q .; then
      printf '%s\n' "${candidate}"
      return 0
    fi
  done

  return 1
}
