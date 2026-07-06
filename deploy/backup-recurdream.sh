#!/usr/bin/env bash
set -Eeuo pipefail

MODE="${1:-daily}"
BACKUP_ROOT="/backup/recurdream"
DAILY_ROOT="${BACKUP_ROOT}/daily"
WEEKLY_ROOT="${BACKUP_ROOT}/weekly"
DATE_STAMP="$(date +%F)"
TIME_STAMP="$(date +%Y%m%d-%H%M%S)"
LOG_FILE="/var/log/recurdream-backup.log"

log() {
  printf '[%s] %s\n' "$(date '+%F %T')" "$*" | tee -a "$LOG_FILE"
}

run_daily_backup() {
  local dest="${DAILY_ROOT}/${DATE_STAMP}-${TIME_STAMP}"
  mkdir -p "$dest"

  log "Starting daily database backup into ${dest}"

  docker exec sub2api-postgres pg_dump -U sub2api -d sub2api \
    | gzip -9 > "${dest}/postgres-sub2api.sql.gz"

  docker exec acg-faka-mysql sh -lc 'mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction --routines --triggers --events acg_faka' \
    | gzip -9 > "${dest}/mysql-acg_faka.sql.gz"

  if [ -f /data/sub2api/redis_data/dump.rdb ]; then
    cp -a /data/sub2api/redis_data/dump.rdb "${dest}/redis-dump.rdb"
  else
    log "Redis dump not found at /data/sub2api/redis_data/dump.rdb"
  fi

  {
    echo "backup_type=daily"
    echo "created_at=$(date -Is)"
    echo "host=$(hostname)"
    echo "postgres_container=sub2api-postgres"
    echo "mysql_container=acg-faka-mysql"
    echo "redis_dump=/data/sub2api/redis_data/dump.rdb"
  } > "${dest}/backup-info.txt"

  (cd "$dest" && sha256sum ./* > SHA256SUMS)
  find "$DAILY_ROOT" -mindepth 1 -maxdepth 1 -type d -mtime +13 -exec rm -rf {} +

  log "Daily database backup completed: ${dest}"
}

run_weekly_backup() {
  local dest="${WEEKLY_ROOT}/${DATE_STAMP}-${TIME_STAMP}"
  mkdir -p "$dest"

  log "Starting weekly file backup into ${dest}"

  local tar_status=0
  tar \
    --warning=no-file-changed \
    --exclude='/data/sub2api/postgres_data' \
    --exclude='/data/sub2api/data/logs' \
    --exclude='/opt/acg-faka/mysql' \
    --exclude='/www/wwwroot/recurdream-docs/node_modules' \
    -czf "${dest}/files.tar.gz" \
    /data/sub2api \
    /opt/acg-faka \
    /www/wwwroot/recurdream-docs \
    /www/server/panel/vhost \
    /www/server/nginx/conf \
    /root/.acme.sh \
    /etc/systemd/system/recurdream-docs-admin.service \
    /etc/systemd/system/site_total.service || tar_status=$?

  if [ "$tar_status" -gt 1 ]; then
    log "Weekly file backup failed: tar exited with ${tar_status}"
    exit "$tar_status"
  fi

  {
    echo "backup_type=weekly"
    echo "created_at=$(date -Is)"
    echo "host=$(hostname)"
    echo "excluded=/data/sub2api/postgres_data"
    echo "excluded=/data/sub2api/data/logs"
    echo "excluded=/opt/acg-faka/mysql"
    echo "excluded=/www/wwwroot/recurdream-docs/node_modules"
  } > "${dest}/backup-info.txt"

  (cd "$dest" && sha256sum ./* > SHA256SUMS)
  find "$WEEKLY_ROOT" -mindepth 1 -maxdepth 1 -type d -mtime +55 -exec rm -rf {} +

  log "Weekly file backup completed: ${dest}"
}

mkdir -p "$DAILY_ROOT" "$WEEKLY_ROOT"

case "$MODE" in
  daily)
    run_daily_backup
    ;;
  weekly)
    run_weekly_backup
    ;;
  all)
    run_daily_backup
    run_weekly_backup
    ;;
  *)
    echo "Usage: $0 {daily|weekly|all}" >&2
    exit 64
    ;;
esac
