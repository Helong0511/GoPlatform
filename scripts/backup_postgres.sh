#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL 备份脚本
# 用法:
#   BACKUP_DIR=/data/backups/postgres \
#   DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=postgres DB_NAME=ad_platform \
#   RETENTION_DAYS=7 ./scripts/backup_postgres.sh

BACKUP_DIR="${BACKUP_DIR:-/data/backups/postgres}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-ad_platform}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"

mkdir -p "${BACKUP_DIR}"

FULL_SQL_GZ="${BACKUP_DIR}/${DB_NAME}_full_${TIMESTAMP}.sql.gz"
SCHEMA_SQL_GZ="${BACKUP_DIR}/${DB_NAME}_schema_${TIMESTAMP}.sql.gz"

echo "[INFO] backup start: ${TIMESTAMP}"
echo "[INFO] target dir: ${BACKUP_DIR}"

pg_dump -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" \
  --no-owner --no-privileges \
  | gzip -9 > "${FULL_SQL_GZ}"

pg_dump -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" \
  --schema-only --no-owner --no-privileges \
  | gzip -9 > "${SCHEMA_SQL_GZ}"

find "${BACKUP_DIR}" -type f -name "*.sql.gz" -mtime +"${RETENTION_DAYS}" -delete

echo "[INFO] backup done:"
echo "       ${FULL_SQL_GZ}"
echo "       ${SCHEMA_SQL_GZ}"
