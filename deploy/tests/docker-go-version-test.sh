#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'docker Go version test failed: %s\n' "$1" >&2
  exit 1
}

go_version=$(awk '$1 == "go" { gsub(/\r/, "", $2); print $2; exit }' backend/go.mod)
[ -n "$go_version" ] || fail 'backend/go.mod does not declare a Go version'

expected_image="golang:${go_version}-alpine"

for dockerfile in Dockerfile deploy/Dockerfile Dockerfile.prebuilt; do
  grep -Eq "^ARG GOLANG_IMAGE=${expected_image}\r?$" "$dockerfile" || \
    fail "$dockerfile must set GOLANG_IMAGE to $expected_image"
done

grep -Eq "^FROM ${expected_image}\r?$" backend/Dockerfile || \
  fail "backend/Dockerfile must build from $expected_image"

printf 'docker Go version test passed (%s)\n' "$go_version"
