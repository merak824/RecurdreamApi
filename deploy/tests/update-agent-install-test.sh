#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'update agent install test failed: %s\n' "$1" >&2
  exit 1
}

grep -Eq '^    - \$\{UPDATE_AGENT_SOCKET:-/run/recurdream/update-agent\.sock\}:/run/recurdream/update-agent\.sock$' deploy/docker-compose.hot.yml \
  || fail 'blue/green app containers must mount the configured update-agent socket'
grep -Eq '^    - "\$\{UPDATE_AGENT_SOCKET_GID:-0\}"$' deploy/docker-compose.hot.yml \
  || fail 'blue/green app containers must receive the socket group'
grep -Fq 'UPDATE_AGENT_SOCKET=/run/recurdream/update-agent.sock' deploy/docker-compose.hot.yml \
  || fail 'containers must use the fixed in-container socket path'
if grep -Fq '/var/run/docker.sock' deploy/docker-compose.hot.yml; then
  fail 'application compose must not mount the Docker Socket'
fi

grep -Eq '^UPDATE_AGENT_DRAIN_SECONDS=300$' deploy/.env.example \
  || fail 'environment example must pin the agent drain window to 300 seconds'
grep -Fq 'ExecStart=/usr/local/libexec/recurdream-update-agent' deploy/update-agent.service \
  || fail 'systemd unit must run the fixed update-agent binary'
grep -Fq 'EnvironmentFile=/etc/recurdream/update-agent.env' deploy/update-agent.service \
  || fail 'systemd unit must use the private environment file'
grep -Fq 'UPDATE_AGENT_DRAIN_SECONDS=300' .github/workflows/deploy.yml \
  || fail 'deployment workflow must install the fixed 300-second drain policy'
grep -Fq 'update-agent.service' .github/workflows/deploy.yml \
  || fail 'deployment workflow must package and install the systemd unit'
if grep -Fq '/var/run/docker.sock' deploy/update-agent.service deploy/install-update-agent.sh; then
  fail 'agent installation files must not expose the Docker Socket to the application'
fi

bash -n deploy/install-update-agent.sh
printf 'update agent install test passed\n'
