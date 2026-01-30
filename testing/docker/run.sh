#!/usr/bin/env bash

# docker compose -f testing/docker/docker-compose.yml up -d
# mariadb -h127.0.0.1 -P3306 -uroot

set -e
shopt -s expand_aliases

CUR_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
cd "${CUR_SCRIPT_DIR}"

alias docker_compose='docker compose -f docker-compose.yml'

function docker_cleanup() {
  docker_compose down --remove-orphans
  docker_compose rm -sfv # remove any leftovers
}
trap docker_cleanup EXIT

docker_cleanup
docker_compose up --remove-orphans
