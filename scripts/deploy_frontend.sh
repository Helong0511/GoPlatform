#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

FRONTEND_DIR="${FRONTEND_DIR:-${REPO_ROOT}/admin}"
BUILD_DIR="${BUILD_DIR:-${FRONTEND_DIR}/dist}"
TARGET_DIR="${TARGET_DIR:-/data/GoPlatform/admin/dist}"

echo "[INFO] Repo root: ${REPO_ROOT}"
echo "[INFO] Frontend dir: ${FRONTEND_DIR}"
echo "[INFO] Build dir: ${BUILD_DIR}"
echo "[INFO] Target dir: ${TARGET_DIR}"

if [[ ! -f "${FRONTEND_DIR}/package.json" ]]; then
  echo "[ERROR] package.json not found in ${FRONTEND_DIR}"
  exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "[ERROR] npm is required but not found in PATH"
  exit 1
fi

pushd "${FRONTEND_DIR}" >/dev/null
echo "[INFO] Installing dependencies..."
npm ci

echo "[INFO] Building frontend..."
npm run build
popd >/dev/null

if [[ ! -d "${BUILD_DIR}" ]]; then
  echo "[ERROR] Build output not found: ${BUILD_DIR}"
  exit 1
fi

echo "[INFO] Publishing assets..."
mkdir -p "${TARGET_DIR}"

if command -v rsync >/dev/null 2>&1; then
  rsync -av --delete "${BUILD_DIR}/" "${TARGET_DIR}/"
else
  rm -rf "${TARGET_DIR:?}/"*
  cp -r "${BUILD_DIR}/." "${TARGET_DIR}/"
fi

echo "[INFO] Frontend deploy completed successfully."
