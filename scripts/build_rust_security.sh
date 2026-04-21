#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
RUST_DIR="${REPO_ROOT}/rust-security"
PROFILE="${PROFILE:-release}"

if [[ ! -f "${RUST_DIR}/Cargo.toml" ]]; then
  echo "[ERROR] rust-security/Cargo.toml not found"
  exit 1
fi

if ! command -v cargo >/dev/null 2>&1; then
  echo "[ERROR] cargo is required but not found in PATH"
  exit 1
fi

pushd "${RUST_DIR}" >/dev/null
if [[ "${PROFILE}" == "release" ]]; then
  cargo build --release
else
  cargo build
fi
popd >/dev/null

echo "[INFO] Rust security library build completed."
