#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL 恢复脚本
# 用法:
#   DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=postgres DB_NAME=ad_platform \
#   ./scripts/restore_postgres.sh /data/backups/postgres/ad_platform_full_20260425_120000.sql.gz

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <backup.sql.gz>"
  exit 1
fi

BACKUP_FILE="$1"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-ad_platform}"

if [[ ! -f "${BACKUP_FILE}" ]]; then
  echo "[ERROR] backup file not found: ${BACKUP_FILE}"
  exit 1
fi

echo "[WARN] restore will overwrite data in ${DB_NAME}"
echo "[WARN] backup file: ${BACKUP_FILE}"
echo -n "Type YES to continue: "
read -r CONFIRM
if [[ "${CONFIRM}" != "YES" ]]; then
  echo "[INFO] cancelled"
  exit 0
fi

echo "[INFO] restore start..."
gunzip -c "${BACKUP_FILE}" | psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -v ON_ERROR_STOP=1
echo "[INFO] restore done"
