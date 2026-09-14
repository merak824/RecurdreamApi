#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'hot deploy diagnostics test failed: %s\n' "$1" >&2
  exit 1
}

assert_contains() {
  needle=$1
  grep -Fq "$needle" deploy/hot-deploy.sh || fail "deploy/hot-deploy.sh is missing: $needle"
}

assert_contains 'docker inspect -f '\''{{json .State}}'\'' "$container"'
assert_contains 'docker inspect -f '\''{{json .State.Health.Log}}'\'' "$container"'
assert_contains 'docker logs --tail=200 "$container"'
assert_contains 'find data/logs -type f -name '\''*.log'\'''

printf 'hot deploy diagnostics test passed\n'
