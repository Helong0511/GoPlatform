#!/usr/bin/env bash
set -euo pipefail

# MySQL 备份脚本
# 用法:
#   BACKUP_DIR=/data/backups/mysql \
#   DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=root DB_NAME=ad_platform \
#   RETENTION_DAYS=7 ./scripts/backup_mysql.sh

BACKUP_DIR="${BACKUP_DIR:-/data/backups/mysql}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-root}"
DB_NAME="${DB_NAME:-ad_platform}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"

mkdir -p "${BACKUP_DIR}"

FULL_SQL_GZ="${BACKUP_DIR}/${DB_NAME}_full_${TIMESTAMP}.sql.gz"
SCHEMA_SQL_GZ="${BACKUP_DIR}/${DB_NAME}_schema_${TIMESTAMP}.sql.gz"

echo "[INFO] backup start: ${TIMESTAMP}"
echo "[INFO] target dir: ${BACKUP_DIR}"

mysqldump -h "${DB_HOST}" -P "${DB_PORT}" -u "${DB_USER}" \
  --single-transaction --quick --routines --triggers \
  "${DB_NAME}" | gzip -9 > "${FULL_SQL_GZ}"

mysqldump -h "${DB_HOST}" -P "${DB_PORT}" -u "${DB_USER}" \
  --no-data --routines --triggers \
  "${DB_NAME}" | gzip -9 > "${SCHEMA_SQL_GZ}"

find "${BACKUP_DIR}" -type f -name "*.sql.gz" -mtime +"${RETENTION_DAYS}" -delete

echo "[INFO] backup done:"
echo "       ${FULL_SQL_GZ}"
echo "       ${SCHEMA_SQL_GZ}"
