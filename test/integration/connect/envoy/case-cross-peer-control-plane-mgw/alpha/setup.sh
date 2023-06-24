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

upsert_config_entry alpha '
kind = "proxy-defaults"
name = "global"
config {
  protocol = "tcp"
}
'

upsert_config_entry alpha '
kind = "mesh"
peering {
  peer_through_mesh_gateways = true
}
'

upsert_config_entry alpha '
kind = "exported-services"
name = "default"
services = [
  {
    name = "s2"
    consumers = [
      {
        peer = "alpha-to-primary"
      }
    ]
  }
]
'

register_services alpha

gen_envoy_bootstrap s2 19002 alpha
gen_envoy_bootstrap mesh-gateway 19003 alpha true
