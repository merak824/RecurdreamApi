#!/usr/bin/env bash
set -Eeuo pipefail

dotenv_get() {
    local key="$1"
    local default="${2:-}"
    local line=""
    local value=""

    if [ -f ".env" ]; then
        line="$(grep -E "^[[:space:]]*${key}=" .env | tail -n 1 || true)"
        if [ -n "$line" ]; then
            value="${line#*=}"
            value="${value%$'\r'}"
            case "$value" in
                \"*\") value="${value#\"}"; value="${value%\"}" ;;
                \'*\') value="${value#\'}"; value="${value%\'}" ;;
            esac
            printf '%s' "$value"
            return
        fi
    fi

    printf '%s' "$default"
}

COMPOSE_FILE="${COMPOSE_FILE:-$(dotenv_get COMPOSE_FILE docker-compose.hot.yml)}"
STATE_DIR="${STATE_DIR:-$(dotenv_get STATE_DIR .hotdeploy)}"
STATE_FILE="${STATE_FILE:-$(dotenv_get STATE_FILE "${STATE_DIR}/active-slot")}"
NGINX_TEMPLATE="${NGINX_TEMPLATE:-$(dotenv_get NGINX_TEMPLATE nginx-sub2api.conf.template)}"
NGINX_CONF_DIR="${NGINX_CONF_DIR:-$(dotenv_get NGINX_CONF_DIR nginx/conf.d)}"
NGINX_CONF_FILE="${NGINX_CONF_FILE:-$(dotenv_get NGINX_CONF_FILE "${NGINX_CONF_DIR}/sub2api.conf")}"
CLIENT_MAX_BODY_SIZE="${CLIENT_MAX_BODY_SIZE:-$(dotenv_get CLIENT_MAX_BODY_SIZE 256m)}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-$(dotenv_get HEALTH_TIMEOUT 180)}"
HEALTH_INTERVAL="${HEALTH_INTERVAL:-$(dotenv_get HEALTH_INTERVAL 3)}"
HEALTH_PATH="${HEALTH_PATH:-$(dotenv_get HEALTH_PATH /health)}"
DRAIN_SECONDS="${DRAIN_SECONDS:-$(dotenv_get DRAIN_SECONDS 15)}"
KEEP_OLD="${KEEP_OLD:-$(dotenv_get KEEP_OLD false)}"
NO_PULL="${NO_PULL:-$(dotenv_get NO_PULL false)}"
BUILD_IMAGE="${BUILD_IMAGE:-$(dotenv_get BUILD_IMAGE false)}"
BUILD_CONTEXT="${BUILD_CONTEXT:-$(dotenv_get BUILD_CONTEXT ..)}"
BUILD_DOCKERFILE="${BUILD_DOCKERFILE:-$(dotenv_get BUILD_DOCKERFILE ../Dockerfile)}"
LOAD_IMAGE_ARCHIVE="${LOAD_IMAGE_ARCHIVE:-$(dotenv_get LOAD_IMAGE_ARCHIVE "")}"
HOT_DEPLOY_MIN_BUILD_MEM_MB="${HOT_DEPLOY_MIN_BUILD_MEM_MB:-$(dotenv_get HOT_DEPLOY_MIN_BUILD_MEM_MB 3072)}"
ALLOW_LOW_RESOURCE_BUILD="${ALLOW_LOW_RESOURCE_BUILD:-$(dotenv_get ALLOW_LOW_RESOURCE_BUILD false)}"
TAKEOVER_LEGACY="${TAKEOVER_LEGACY:-$(dotenv_get TAKEOVER_LEGACY true)}"
ALLOW_ACTIVE_SLOT_DEPLOY="${ALLOW_ACTIVE_SLOT_DEPLOY:-$(dotenv_get ALLOW_ACTIVE_SLOT_DEPLOY false)}"
PROXY_BUFFER_SIZE="${PROXY_BUFFER_SIZE:-$(dotenv_get PROXY_BUFFER_SIZE 32k)}"
PROXY_BUFFERS="${PROXY_BUFFERS:-$(dotenv_get PROXY_BUFFERS "8 32k")}"
PROXY_BUSY_BUFFERS_SIZE="${PROXY_BUSY_BUFFERS_SIZE:-$(dotenv_get PROXY_BUSY_BUFFERS_SIZE 64k)}"

BLUE_SLOT="blue"
GREEN_SLOT="green"

info() {
    printf '[INFO] %s\n' "$*"
}

success() {
    printf '[SUCCESS] %s\n' "$*"
}

warn() {
    printf '[WARNING] %s\n' "$*"
}

fail() {
    printf '[ERROR] %s\n' "$*" >&2
    exit 1
}

require_setting() {
    local name="$1"
    local value="$2"

    [ -n "$value" ] || fail "$name must be set for hot deployment"
}

require_min_length() {
    local name="$1"
    local value="$2"
    local min_length="$3"

    if [ "${#value}" -lt "$min_length" ]; then
        fail "$name must be at least ${min_length} characters"
    fi
}

usage() {
    cat <<'EOF'
Usage:
  bash hot-deploy.sh [--slot blue|green] [--image IMAGE] [--build] [--load-image ARCHIVE] [--no-pull] [--keep-old]

Environment:
  POSTGRES_PASSWORD          Required. Must not be the example value.
  JWT_SECRET                 Required. At least 32 characters.
  TOTP_ENCRYPTION_KEY        Required. At least 32 characters.
  SUB2API_IMAGE              Image to deploy. Default: weishaw/sub2api:latest
  LOAD_IMAGE_ARCHIVE         docker save tar/tar.gz archive to load before deploy.
  SERVER_PORT                Host port exposed by nginx. Default: 8080
  BIND_HOST                  Host bind address. Default: 0.0.0.0
  HEALTH_TIMEOUT             Seconds to wait for the new slot. Default: 180
  DRAIN_SECONDS              Seconds to keep the old slot after switching. Default: 15
  BUILD_CONTEXT              Docker build context for --build. Default: ..
  BUILD_DOCKERFILE           Dockerfile path for --build. Default: ../Dockerfile
  BUILD_TAG                  Image tag for --build. Default: sub2api-hot:<timestamp>
  HOT_DEPLOY_MIN_BUILD_MEM_MB Refuse --build below this MemTotal. Default: 3072.
  ALLOW_LOW_RESOURCE_BUILD=true
                             Allow --build on hosts below HOT_DEPLOY_MIN_BUILD_MEM_MB.
  TAKEOVER_LEGACY=true       Stop legacy container "sub2api" only when nginx first takes over.
  ALLOW_ACTIVE_SLOT_DEPLOY=false
                             Allow deploying into the active slot. This is not hot.
  KEEP_OLD=true              Leave the old slot running after traffic switch.
  NO_PULL=true               Skip docker compose pull.

Examples:
  bash hot-deploy.sh
  bash hot-deploy.sh --build
  bash hot-deploy.sh --load-image /tmp/sub2api-hot.tar.gz --keep-old
  bash hot-deploy.sh --image ghcr.io/merak824/recurdreamapi:latest
  SUB2API_IMAGE=weishaw/sub2api:v1.2.3 bash hot-deploy.sh
EOF
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --slot)
            [ "$#" -ge 2 ] || fail "--slot requires blue or green"
            TARGET_SLOT="$2"
            shift 2
            ;;
        --image)
            [ "$#" -ge 2 ] || fail "--image requires an image reference"
            export SUB2API_IMAGE="$2"
            shift 2
            ;;
        --no-pull)
            NO_PULL=true
            shift
            ;;
        --build)
            BUILD_IMAGE=true
            NO_PULL=true
            shift
            ;;
        --load-image)
            [ "$#" -ge 2 ] || fail "--load-image requires a docker save tar or tar.gz archive"
            LOAD_IMAGE_ARCHIVE="$2"
            NO_PULL=true
            shift 2
            ;;
        --keep-old)
            KEEP_OLD=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            fail "unknown argument: $1"
            ;;
    esac
done

command -v docker >/dev/null 2>&1 || fail "docker is required"
[ -f "$COMPOSE_FILE" ] || fail "missing $COMPOSE_FILE; run this script from the deploy directory"
[ -f "$NGINX_TEMPLATE" ] || fail "missing $NGINX_TEMPLATE"
[ -f ".env" ] || warn ".env not found; compose will rely on exported environment variables"

compose() {
    docker compose -f "$COMPOSE_FILE" "$@"
}

slot_service() {
    case "$1" in
        "$BLUE_SLOT") printf 'sub2api-blue' ;;
        "$GREEN_SLOT") printf 'sub2api-green' ;;
        *) fail "invalid slot: $1" ;;
    esac
}

opposite_slot() {
    case "$1" in
        "$BLUE_SLOT") printf '%s' "$GREEN_SLOT" ;;
        "$GREEN_SLOT") printf '%s' "$BLUE_SLOT" ;;
        *) fail "invalid slot: $1" ;;
    esac
}

container_running() {
    local container="$1"
    [ "$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || true)" = "true" ]
}

nginx_running() {
    container_running "sub2api-nginx"
}

render_nginx_config() {
    local slot="$1"
    render_nginx_config_to "$slot" "$NGINX_CONF_FILE"
}

render_nginx_config_to() {
    local slot="$1"
    local output_file="$2"
    local service
    service="$(slot_service "$slot")"
    mkdir -p "$(dirname "$output_file")" "$STATE_DIR"

    sed \
        -e "s|\${UPSTREAM_SERVICE}|${service}|g" \
        -e "s|\${CLIENT_MAX_BODY_SIZE}|${CLIENT_MAX_BODY_SIZE}|g" \
        -e "s|\${PROXY_BUFFER_SIZE}|${PROXY_BUFFER_SIZE}|g" \
        -e "s|\${PROXY_BUFFERS}|${PROXY_BUFFERS}|g" \
        -e "s|\${PROXY_BUSY_BUFFERS_SIZE}|${PROXY_BUSY_BUFFERS_SIZE}|g" \
        "$NGINX_TEMPLATE" | sed $'1s/^\xef\xbb\xbf//' > "$output_file"
}

backup_nginx_config() {
    local backup_file=""
    if [ -f "$NGINX_CONF_FILE" ]; then
        backup_file="${STATE_DIR}/sub2api.conf.$(date +%Y%m%d%H%M%S).bak"
        mkdir -p "$STATE_DIR"
        cp "$NGINX_CONF_FILE" "$backup_file"
    fi
    printf '%s' "$backup_file"
}

restore_nginx_config() {
    local backup_file="${1:-}"
    if [ -n "$backup_file" ] && [ -f "$backup_file" ]; then
        cp "$backup_file" "$NGINX_CONF_FILE"
        return
    fi
    rm -f "$NGINX_CONF_FILE"
}

reload_nginx_or_restore() {
    local backup_file="$1"
    local reason="$2"

    warn "$reason"
    restore_nginx_config "$backup_file"
    if nginx_running; then
        compose exec -T nginx nginx -t >/dev/null 2>&1 \
            && compose exec -T nginx nginx -s reload >/dev/null 2>&1 \
            || true
    fi
    exit 1
}

dump_container_diagnostics() {
    local container="$1"

    printf '[ERROR] Container state: '
    docker inspect -f '{{json .State}}' "$container" 2>&1 || true
    printf '[ERROR] Container health log: '
    docker inspect -f '{{json .State.Health.Log}}' "$container" 2>&1 || true
    printf '[ERROR] Container logs (tail 200):\n'
    docker logs --tail=200 "$container" 2>&1 || true
}

wait_for_container_health() {
    local container="$1"
    local deadline
    local status
    deadline=$((SECONDS + HEALTH_TIMEOUT))

    while [ "$SECONDS" -lt "$deadline" ]; do
        status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || true)"
        if [ "$status" = "healthy" ]; then
            return 0
        fi
        if [ "$status" = "exited" ] || [ "$status" = "dead" ]; then
            dump_container_diagnostics "$container"
            return 1
        fi
        sleep "$HEALTH_INTERVAL"
    done

    dump_container_diagnostics "$container"
    return 1
}

wait_for_public_health() {
    local port="${SERVER_PORT:-8080}"
    local host="${BIND_HOST:-127.0.0.1}"
    local url
    if [ "$host" = "0.0.0.0" ]; then
        host="127.0.0.1"
    fi
    url="http://${host}:${port}${HEALTH_PATH}"

    if command -v curl >/dev/null 2>&1; then
        local deadline=$((SECONDS + 30))
        while [ "$SECONDS" -lt "$deadline" ]; do
            if curl -fsS --max-time 5 "$url" >/dev/null; then
                return 0
            fi
            sleep 2
        done
        return 1
    fi

    warn "curl not found; skipped public health check at $url"
    return 0
}

host_mem_total_mb() {
    if [ -r /proc/meminfo ]; then
        awk '/^MemTotal:/ { printf "%d", $2 / 1024 }' /proc/meminfo
        return
    fi
    printf '0'
}

check_build_resources() {
    if [ "$BUILD_IMAGE" != "true" ] || [ "$ALLOW_LOW_RESOURCE_BUILD" = "true" ]; then
        return
    fi
    case "$HOT_DEPLOY_MIN_BUILD_MEM_MB" in
        ''|*[!0-9]*) return ;;
    esac

    local mem_total_mb
    mem_total_mb="$(host_mem_total_mb)"
    if [ "$mem_total_mb" -gt 0 ] && [ "$mem_total_mb" -lt "$HOT_DEPLOY_MIN_BUILD_MEM_MB" ]; then
        fail "host memory is ${mem_total_mb}MiB; refusing --build below HOT_DEPLOY_MIN_BUILD_MEM_MB=${HOT_DEPLOY_MIN_BUILD_MEM_MB}. Build the image on a larger machine, copy it with docker save/load, then run: bash hot-deploy.sh --load-image ARCHIVE --keep-old. To override, set ALLOW_LOW_RESOURCE_BUILD=true."
    fi
}

prepare_runtime_config() {
    mkdir -p data

    if [ ! -f "config.yaml" ]; then
        return
    fi

    if [ ! -f "data/config.yaml" ]; then
        warn "legacy ./config.yaml exists and ./data/config.yaml is missing; copying it for the hot-deploy volume layout"
        cp -p "config.yaml" "data/config.yaml"
        return
    fi

    if ! cmp -s "config.yaml" "data/config.yaml"; then
        fail "both ./config.yaml and ./data/config.yaml exist but differ. This can silently change runtime config after deploy because docker-compose.hot.yml mounts ./data. Copy the active config to ./data/config.yaml or remove the stale file, then rerun."
    fi
}

load_image_archive() {
    local archive="$1"
    local load_output
    local loaded_image

    [ -f "$archive" ] || fail "image archive not found: $archive"
    info "Loading image archive: $archive"

    case "$archive" in
        *.tar.gz|*.tgz|*.gz)
            command -v gzip >/dev/null 2>&1 || fail "gzip is required to load compressed image archives"
            if ! load_output="$(gzip -dc "$archive" | docker load 2>&1)"; then
                printf '%s\n' "$load_output"
                fail "docker load failed"
            fi
            ;;
        *)
            if ! load_output="$(docker load -i "$archive" 2>&1)"; then
                printf '%s\n' "$load_output"
                fail "docker load failed"
            fi
            ;;
    esac

    printf '%s\n' "$load_output"
    loaded_image="$(printf '%s\n' "$load_output" | awk -F': ' '/Loaded image:/ { image=$2 } END { print image }')"
    if [ -z "${SUB2API_IMAGE:-}" ]; then
        [ -n "$loaded_image" ] || fail "image archive loaded without a tag; pass --image IMAGE explicitly"
        export SUB2API_IMAGE="$loaded_image"
    fi
}

active_slot_from_state() {
    if [ -f "$STATE_FILE" ]; then
        tr -d '[:space:]' < "$STATE_FILE"
    fi
}

active_slot_from_nginx_config() {
    if [ ! -f "$NGINX_CONF_FILE" ]; then
        return
    fi

    if grep -Eq 'server[[:space:]]+sub2api-blue:8080' "$NGINX_CONF_FILE"; then
        printf '%s' "$BLUE_SLOT"
        return
    fi
    if grep -Eq 'server[[:space:]]+sub2api-green:8080' "$NGINX_CONF_FILE"; then
        printf '%s' "$GREEN_SLOT"
        return
    fi
}

detect_active_slot() {
    local nginx_slot
    local state_slot

    nginx_slot="$(active_slot_from_nginx_config || true)"
    case "$nginx_slot" in
        "$BLUE_SLOT"|"$GREEN_SLOT")
            printf '%s' "$nginx_slot"
            return
            ;;
    esac

    state_slot="$(active_slot_from_state || true)"
    case "$state_slot" in
        "$BLUE_SLOT"|"$GREEN_SLOT")
            printf '%s' "$state_slot"
            return
            ;;
    esac

    if container_running "$(slot_service "$BLUE_SLOT")"; then
        printf '%s' "$BLUE_SLOT"
        return
    fi
    if container_running "$(slot_service "$GREEN_SLOT")"; then
        printf '%s' "$GREEN_SLOT"
        return
    fi
}

ACTIVE_SLOT="$(detect_active_slot || true)"
if [ -n "${TARGET_SLOT:-}" ]; then
    case "$TARGET_SLOT" in
        "$BLUE_SLOT"|"$GREEN_SLOT") ;;
        *) fail "--slot must be blue or green" ;;
    esac
else
    if [ -n "$ACTIVE_SLOT" ]; then
        TARGET_SLOT="$(opposite_slot "$ACTIVE_SLOT")"
    else
        TARGET_SLOT="$BLUE_SLOT"
    fi
fi

OLD_SLOT=""
if [ "$TARGET_SLOT" = "$BLUE_SLOT" ]; then
    OLD_SLOT="$GREEN_SLOT"
else
    OLD_SLOT="$BLUE_SLOT"
fi

if [ -n "$ACTIVE_SLOT" ] && [ "$TARGET_SLOT" = "$ACTIVE_SLOT" ] && [ "$ALLOW_ACTIVE_SLOT_DEPLOY" != "true" ]; then
    fail "target slot '$TARGET_SLOT' is already active; omit --slot or set ALLOW_ACTIVE_SLOT_DEPLOY=true"
fi

TARGET_SERVICE="$(slot_service "$TARGET_SLOT")"
TARGET_CONTAINER="$TARGET_SERVICE"
OLD_SERVICE="$(slot_service "$OLD_SLOT")"
LEGACY_STOPPED=false
SERVER_PORT="${SERVER_PORT:-$(dotenv_get SERVER_PORT 8080)}"
BIND_HOST="${BIND_HOST:-$(dotenv_get BIND_HOST 0.0.0.0)}"
SUB2API_IMAGE="${SUB2API_IMAGE:-$(dotenv_get SUB2API_IMAGE "")}"
POSTGRES_PASSWORD_VALUE="${POSTGRES_PASSWORD:-$(dotenv_get POSTGRES_PASSWORD "")}"
JWT_SECRET_VALUE="${JWT_SECRET:-$(dotenv_get JWT_SECRET "")}"
TOTP_ENCRYPTION_KEY_VALUE="${TOTP_ENCRYPTION_KEY:-$(dotenv_get TOTP_ENCRYPTION_KEY "")}"

require_setting POSTGRES_PASSWORD "$POSTGRES_PASSWORD_VALUE"
[ "$POSTGRES_PASSWORD_VALUE" != "change_this_secure_password" ] || fail "POSTGRES_PASSWORD is still the example value"
require_setting JWT_SECRET "$JWT_SECRET_VALUE"
require_min_length JWT_SECRET "$JWT_SECRET_VALUE" 32
require_setting TOTP_ENCRYPTION_KEY "$TOTP_ENCRYPTION_KEY_VALUE"
require_min_length TOTP_ENCRYPTION_KEY "$TOTP_ENCRYPTION_KEY_VALUE" 32

check_build_resources
prepare_runtime_config

if [ -n "$LOAD_IMAGE_ARCHIVE" ]; then
    load_image_archive "$LOAD_IMAGE_ARCHIVE"
fi

if [ "$BUILD_IMAGE" = "true" ]; then
    BUILD_TAG="${BUILD_TAG:-sub2api-hot:$(date +%Y%m%d%H%M%S)}"
    info "Building local image: $BUILD_TAG"
    docker build -f "$BUILD_DOCKERFILE" -t "$BUILD_TAG" "$BUILD_CONTEXT"
    export SUB2API_IMAGE="$BUILD_TAG"
fi

IMAGE="${SUB2API_IMAGE:-weishaw/sub2api:latest}"
export SUB2API_IMAGE="$IMAGE"

info "Deploying image: $IMAGE"
info "Target slot: $TARGET_SLOT"
[ -n "$ACTIVE_SLOT" ] && info "Current active slot: $ACTIVE_SLOT"

mkdir -p data postgres_data redis_data "$NGINX_CONF_DIR" "$STATE_DIR"

if [ ! -f "$NGINX_CONF_FILE" ]; then
    if nginx_running; then
        fail "nginx is running but $NGINX_CONF_FILE is missing; inspect the server before hot deploy"
    fi
    info "Creating initial nginx config for $TARGET_SLOT"
    render_nginx_config "$TARGET_SLOT"
fi

info "Starting database, redis, and target slot"
if [ "$NO_PULL" != "true" ]; then
    compose --profile "$TARGET_SLOT" pull "$TARGET_SERVICE"
fi
compose up -d --no-recreate postgres redis
info "Waiting for postgres and redis to become healthy"
wait_for_container_health sub2api-postgres || fail "postgres failed health check"
wait_for_container_health sub2api-redis || fail "redis failed health check"
compose --profile "$TARGET_SLOT" up -d --no-deps --force-recreate "$TARGET_SERVICE"

info "Waiting for $TARGET_SERVICE to become healthy"
if ! wait_for_container_health "$TARGET_CONTAINER"; then
    warn "New slot failed health check; leaving existing traffic untouched"
    compose --profile "$TARGET_SLOT" stop "$TARGET_SERVICE" >/dev/null 2>&1 || true
    exit 1
fi

info "Switching nginx upstream to $TARGET_SERVICE"
NGINX_CONF_BACKUP="$(backup_nginx_config)"
render_nginx_config "$TARGET_SLOT"

if ! nginx_running; then
    info "Testing nginx config before public port handoff"
    if ! compose run --rm --no-deps nginx nginx -t >/dev/null; then
        restore_nginx_config "$NGINX_CONF_BACKUP"
        fail "nginx config test failed; legacy traffic untouched"
    fi
    if [ "$TAKEOVER_LEGACY" = "true" ] && container_running "sub2api"; then
        info "Stopping legacy sub2api container so nginx can bind the public port"
        docker stop sub2api >/dev/null
        LEGACY_STOPPED=true
    fi
    if ! compose up -d nginx; then
        restore_nginx_config "$NGINX_CONF_BACKUP"
        if [ "$LEGACY_STOPPED" = "true" ]; then
            warn "nginx failed to start; restarting legacy sub2api container"
            docker start sub2api >/dev/null
        fi
        exit 1
    fi
else
    if ! compose exec -T nginx nginx -t >/dev/null; then
        reload_nginx_or_restore "$NGINX_CONF_BACKUP" "nginx config test failed; keeping previous traffic target"
    fi
    if ! compose exec -T nginx nginx -s reload; then
        reload_nginx_or_restore "$NGINX_CONF_BACKUP" "nginx reload failed; restoring previous traffic target"
    fi
fi

if ! wait_for_public_health; then
    warn "Traffic switch completed, but public health check failed"
    if container_running "$OLD_SERVICE"; then
        warn "Rolling nginx back to $OLD_SERVICE"
        render_nginx_config "$OLD_SLOT"
        compose exec -T nginx nginx -t >/dev/null
        compose exec -T nginx nginx -s reload
        printf '%s\n' "$OLD_SLOT" > "$STATE_FILE"
    elif [ "$LEGACY_STOPPED" = "true" ]; then
        warn "Restarting legacy sub2api container"
        restore_nginx_config "$NGINX_CONF_BACKUP"
        compose stop nginx >/dev/null 2>&1 || true
        docker start sub2api >/dev/null
    fi
    exit 1
fi

printf '%s\n' "$TARGET_SLOT" > "$STATE_FILE"

if [ "$KEEP_OLD" != "true" ] && container_running "$OLD_SERVICE"; then
    if [ "$DRAIN_SECONDS" -gt 0 ]; then
        info "Draining old slot for ${DRAIN_SECONDS}s"
        sleep "$DRAIN_SECONDS"
    fi
    info "Stopping old slot: $OLD_SERVICE"
    compose --profile "$OLD_SLOT" stop "$OLD_SERVICE"
fi

success "Hot deployment complete. Active slot: $TARGET_SLOT"
