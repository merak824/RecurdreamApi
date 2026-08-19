#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'hot deploy workflow defaults test failed: %s\n' "$1" >&2
  exit 1
}

keep_old_block=$(awk '/^      keep_old:/{in_block=1; next} /^      [a-zA-Z_][a-zA-Z0-9_]*:/{in_block=0} in_block{print}' .github/workflows/deploy.yml)
printf '%s\n' "$keep_old_block" | grep -Eq '^[[:space:]]+default:[[:space:]]+false[[:space:]]*$' || fail 'keep_old default must be false'

drain_block=$(awk '/^      drain_seconds:/{in_block=1; next} /^      [a-zA-Z_][a-zA-Z0-9_]*:/{in_block=0} in_block{print}' .github/workflows/deploy.yml)
printf '%s\n' "$drain_block" | grep -Eq '^[[:space:]]+default:[[:space:]]+"300"[[:space:]]*$' || fail 'drain_seconds default must be 300'

if grep -Fq 'By default, keep_old is enabled' deploy/README.md; then
  fail 'README must not describe keep_old as the default'
fi

if grep -Fq 'bash hot-deploy.sh --load-image /tmp/sub2api-hot-<sha>.tar.gz --keep-old' deploy/README.md; then
  fail 'README routine workflow command must not force --keep-old'
fi

printf 'hot deploy workflow defaults test passed\n'
