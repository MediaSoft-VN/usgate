#!/usr/bin/env bash

# Keep an active service's port unless another process is confirmed to own it.
select_deployment_port() {
  local service_active=$1
  local preferred_port=$2
  local service_pid=$3
  local listeners
  local last_port
  local candidate

  command -v ss >/dev/null 2>&1 || return 2
  if [[ ${service_active} == true ]]; then
    listeners=$(sudo -n ss -ltnpH "sport = :${preferred_port}") || return 2
    if [[ -z ${listeners} || -z ${service_pid} ]] ||
      { [[ -n ${service_pid} ]] && grep -Eq "pid=${service_pid}(,|\\))" <<<"${listeners}"; }; then
      printf '%s\n' "${preferred_port}"
      return 0
    fi
  fi

  last_port=$((preferred_port + 19))
  if (( last_port > 65535 )); then
    last_port=65535
  fi
  for ((candidate = preferred_port; candidate <= last_port; candidate++)); do
    if ! sudo -n ss -ltnH "sport = :${candidate}" | grep -q .; then
      printf '%s\n' "${candidate}"
      return 0
    fi
  done

  return 1
}
