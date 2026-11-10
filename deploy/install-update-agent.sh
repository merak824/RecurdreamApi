#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Usage: install-update-agent.sh <agent-binary> <service-template>

Environment:
  UPDATE_AGENT_USER                 User allowed to run Docker and deploy (default: root)
  UPDATE_AGENT_GROUP                Group for the service (default: user's primary group)
  UPDATE_AGENT_DEPLOY_PATH          Hot-deploy directory (required)
  UPDATE_AGENT_HOT_DEPLOY_SCRIPT    Fixed hot-deploy script path
  UPDATE_AGENT_SOCKET               Host Unix socket path
  UPDATE_AGENT_IMAGE                Allowlisted image repository
  UPDATE_AGENT_CURRENT_VERSION_FILE Version state file
  UPDATE_AGENT_LOG_DIR              Agent log directory
  UPDATE_AGENT_DRAIN_SECONDS        Must be 300
  UPDATE_AGENT_INITIAL_VERSION      Seed version when state file is absent
EOF
}

if [ "$#" -ne 2 ]; then
  usage >&2
  exit 2
fi

agent_binary=$1
service_template=$2
agent_user=${UPDATE_AGENT_USER:-root}
agent_group=${UPDATE_AGENT_GROUP:-}
deploy_path=${UPDATE_AGENT_DEPLOY_PATH:-}
socket_path=${UPDATE_AGENT_SOCKET:-/run/recurdream/update-agent.sock}
image_repository=${UPDATE_AGENT_IMAGE:-ghcr.io/merak824/recurdreamapi}
hot_deploy_script=${UPDATE_AGENT_HOT_DEPLOY_SCRIPT:-$deploy_path/hot-deploy.sh}
current_version_file=${UPDATE_AGENT_CURRENT_VERSION_FILE:-$deploy_path/.update-agent/current-version}
log_dir=${UPDATE_AGENT_LOG_DIR:-/var/log/recurdream-update-agent}
drain_seconds=${UPDATE_AGENT_DRAIN_SECONDS:-300}
initial_version=${UPDATE_AGENT_INITIAL_VERSION:-0.0.0}
install_root=${UPDATE_AGENT_INSTALL_ROOT:-/usr/local/libexec}
service_name=recurdream-update-agent.service
env_dir=/etc/recurdream
env_file=$env_dir/update-agent.env
service_file=/etc/systemd/system/$service_name

fail() {
  printf 'update-agent installation failed: %s\n' "$1" >&2
  exit 1
}

require_absolute_path() {
  case "$1" in
    /*) ;;
    *) fail "$2 must be an absolute path" ;;
  esac
}

is_valid_version() {
  # 允许 semver 的预发布/构建后缀：本仓库版本形如 0.2.20-rd.4，
  # 镜像 tag 带 -rd.N，而旧规则只认纯 x.y.z，导致部署时
  # "current version file is invalid: 0.2.20-rd.2" 直接中断。
  [[ "$1" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.]+)?(\+[0-9A-Za-z.]+)?$ ]]
}

[ -f "$agent_binary" ] || fail "agent binary not found: $agent_binary"
[ -x "$agent_binary" ] || fail "agent binary is not executable: $agent_binary"
[ -f "$service_template" ] || fail "service template not found: $service_template"
[ -n "$deploy_path" ] || fail 'UPDATE_AGENT_DEPLOY_PATH is required'
[ -d "$deploy_path" ] || fail "deployment directory not found: $deploy_path"
[ -f "$hot_deploy_script" ] || fail "hot-deploy script not found: $hot_deploy_script"
[ -x "$hot_deploy_script" ] || fail "hot-deploy script is not executable: $hot_deploy_script"

require_absolute_path "$socket_path" UPDATE_AGENT_SOCKET
require_absolute_path "$deploy_path" UPDATE_AGENT_DEPLOY_PATH
require_absolute_path "$hot_deploy_script" UPDATE_AGENT_HOT_DEPLOY_SCRIPT
require_absolute_path "$current_version_file" UPDATE_AGENT_CURRENT_VERSION_FILE
require_absolute_path "$log_dir" UPDATE_AGENT_LOG_DIR

[ "$drain_seconds" = 300 ] || fail 'UPDATE_AGENT_DRAIN_SECONDS must be 300'
[[ "$image_repository" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]*$ ]] || fail 'UPDATE_AGENT_IMAGE is invalid'
id "$agent_user" >/dev/null 2>&1 || fail "service user not found: $agent_user"
if [ -z "$agent_group" ]; then
  agent_group=$(id -gn "$agent_user")
fi
getent group "$agent_group" >/dev/null 2>&1 || fail "service group not found: $agent_group"

socket_dir=$(dirname "$socket_path")
version_dir=$(dirname "$current_version_file")
mkdir -p "$install_root" "$env_dir" "$socket_dir" "$version_dir" "$log_dir"
install -m 0755 "$agent_binary" "$install_root/recurdream-update-agent"

if [ ! -f "$current_version_file" ] && [ -n "$initial_version" ]; then
  is_valid_version "$initial_version" || fail 'UPDATE_AGENT_INITIAL_VERSION is invalid'
  printf '%s\n' "$initial_version" > "$current_version_file"
  chmod 0600 "$current_version_file"
fi

if [ -f "$current_version_file" ]; then
  current_version=$(tr -d '\r\n[:space:]' < "$current_version_file")
  is_valid_version "$current_version" || fail "current version file is invalid: $current_version"
fi

umask 077
cat > "$env_file" <<EOF
UPDATE_AGENT_SOCKET=$socket_path
UPDATE_AGENT_DEPLOY_PATH=$deploy_path
UPDATE_AGENT_HOT_DEPLOY_SCRIPT=$hot_deploy_script
UPDATE_AGENT_IMAGE=$image_repository
UPDATE_AGENT_CURRENT_VERSION_FILE=$current_version_file
UPDATE_AGENT_LOG_DIR=$log_dir
UPDATE_AGENT_DRAIN_SECONDS=300
UPDATE_AGENT_TIMEOUT_MINUTES=${UPDATE_AGENT_TIMEOUT_MINUTES:-30}
EOF
chmod 0600 "$env_file"

sed \
  -e "s|__UPDATE_AGENT_USER__|$agent_user|g" \
  -e "s|__UPDATE_AGENT_GROUP__|$agent_group|g" \
  -e "s|__UPDATE_AGENT_DEPLOY_PATH__|$deploy_path|g" \
  -e "s|__UPDATE_AGENT_SOCKET_DIR__|$socket_dir|g" \
  -e "s|__UPDATE_AGENT_LOG_DIR__|$log_dir|g" \
  "$service_template" > "$service_file"
chmod 0644 "$service_file"

if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
  systemctl enable --now "$service_name"
else
  printf 'warning: systemctl is unavailable; installed files but did not start %s\n' "$service_name" >&2
fi

printf 'installed %s using image repository %s\n' "$service_name" "$image_repository"
