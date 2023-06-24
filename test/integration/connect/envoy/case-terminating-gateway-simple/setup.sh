#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

function docker_exec {
  if ! docker.exe exec -i "$@"; then
    echo "Failed to execute: docker exec -i $@" 1>&2
    return 1
  fi
}

function docker_consul {
  local DC=$1
  shift 1
  docker_exec envoy_consul-${DC}_1 "$@"
}

function upsert_config_entry {
  local DC="$1"
  local BODY="$2"

  echo "$BODY" | docker_consul "$DC" config write -
}



set -euo pipefail

upsert_config_entry primary '
kind = "terminating-gateway"
name = "terminating-gateway"
services = [
  {
    name = "s2"
  }
]
'

register_services primary

gen_envoy_bootstrap terminating-gateway 20000 primary true
gen_envoy_bootstrap s1 19000
