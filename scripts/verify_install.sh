#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="${XUI_INSTALL_DIR:-/usr/local/x-ui}"
BINARY_PATH="${XUI_BINARY_PATH:-${INSTALL_DIR}/x-ui}"
VERSION_FILE="${XUI_VERSION_FILE:-${INSTALL_DIR}/config/version}"

read_version_key() {
  local key="$1"
  local file="$2"
  [[ -f "${file}" ]] || return 0
  awk -F '[:=]' -v key="${key}" '
    $1 ~ "^[[:space:]]*" key "[[:space:]]*$" {
      sub(/^[[:space:]]+/, "", $2)
      sub(/[[:space:]]+$/, "", $2)
      print $2
      exit
    }
  ' "${file}"
}

binary_value() {
  local key="$1"
  [[ -x "${BINARY_PATH}" ]] || return 0
  "${BINARY_PATH}" version 2>/dev/null | sed -n "s/^${key}: //p" | head -n1
}

local_git_commit() {
  if git -C "${ROOT}" rev-parse HEAD >/dev/null 2>&1; then
    git -C "${ROOT}" rev-parse HEAD
  fi
}

installed_commit="$(binary_value "Commit")"
installed_version="$(binary_value "Version")"
release_commit="$(read_version_key "commit" "${VERSION_FILE}")"
release_version="$(read_version_key "version" "${VERSION_FILE}")"
local_commit="$(local_git_commit)"

status="consistent"
message="installed commit matches release commit"

if [[ -z "${installed_commit}" || -z "${release_commit}" ]]; then
  status="drift detected"
  message="missing installed commit or release commit metadata"
elif [[ "${installed_commit}" != "${release_commit}" ]]; then
  status="drift detected"
  message="installed commit differs from release commit"
elif [[ -n "${local_commit}" && "${local_commit}" != "${release_commit}" ]]; then
  status="drift detected"
  message="local git commit is ahead of or differs from installed release"
fi

printf 'installed version : %s\n' "${installed_version:-未知}"
printf 'installed commit  : %s\n' "${installed_commit:-未知}"
printf 'release version   : %s\n' "${release_version:-未知}"
printf 'release commit    : %s\n' "${release_commit:-未知}"
printf 'local git commit  : %s\n' "${local_commit:-未检测到}"

if [[ "${status}" == "consistent" ]]; then
  printf 'status            : consistent\n'
else
  printf 'status            : drift detected\n'
fi
printf 'message           : %s\n' "${message}"
