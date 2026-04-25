#!/usr/bin/env bash
set -euo pipefail

# MySQL 恢复脚本
# 用法:
#   DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=root DB_NAME=ad_platform \
#   ./scripts/restore_mysql.sh /data/backups/mysql/ad_platform_full_20260425_120000.sql.gz

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <backup.sql.gz>"
  exit 1
fi

BACKUP_FILE="$1"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-root}"
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
gunzip -c "${BACKUP_FILE}" | mysql -h "${DB_HOST}" -P "${DB_PORT}" -u "${DB_USER}" "${DB_NAME}"
echo "[INFO] restore done"
