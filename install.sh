#!/usr/bin/env bash
set -euo pipefail

APP_NAME="x-ui"
INSTALL_DIR="/usr/local/x-ui"
CONFIG_DIR="/etc/x-ui"
DB_PATH="${CONFIG_DIR}/x-ui.db"
SERVICE_PATH="/etc/systemd/system/x-ui.service"
COMMAND_PATH="/usr/bin/x-ui"
REPO="${XUI_REPO:-torr9522/z-ui}"
XUI_RELEASE_VERSION="${1:-${XUI_VERSION:-}}"
TMP_DIR=""

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

require_root() {
  [[ "${EUID}" -eq 0 ]] || fail "install.sh must be run as root"
}

detect_os() {
  [[ -r /etc/os-release ]] || fail "cannot detect operating system"
  # shellcheck disable=SC1091
  . /etc/os-release
  case "${ID:-}" in
    debian)
      OS_FAMILY="debian"
      ;;
    ubuntu)
      OS_FAMILY="ubuntu"
      ;;
    centos | rocky | almalinux | rhel)
      OS_FAMILY="rhel"
      ;;
    *)
      case " ${ID_LIKE:-} " in
        *" debian "*)
          OS_FAMILY="debian"
          ;;
        *" rhel "* | *" fedora "*)
          OS_FAMILY="rhel"
          ;;
        *)
          fail "unsupported operating system: ${ID:-unknown}"
          ;;
      esac
      ;;
  esac
}

install_dependencies() {
  if [[ "${XUI_SKIP_DEP_INSTALL:-}" == "1" ]]; then
    return
  fi

  case "${OS_FAMILY}" in
    debian | ubuntu)
      export DEBIAN_FRONTEND=noninteractive
      apt-get update -y
      apt-get install -y curl wget tar unzip ca-certificates systemd sqlite3
      ;;
    rhel)
      local pm="yum"
      if command -v dnf >/dev/null 2>&1; then
        pm="dnf"
      fi
      "${pm}" install -y curl wget tar unzip ca-certificates systemd sqlite
      ;;
    *)
      fail "unsupported package manager for ${OS_FAMILY}"
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64)
      ARCH="amd64"
      ;;
    aarch64 | arm64)
      ARCH="arm64"
      ;;
    *)
      fail "unsupported architecture: $(uname -m)"
      ;;
  esac
}

latest_version() {
  local api="https://api.github.com/repos/${REPO}/releases/latest"
  curl -fsSL "${api}" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1
}

download_package() {
  local tmp_dir="$1"
  local package_path="${tmp_dir}/x-ui.tar.gz"
  local url="${XUI_INSTALL_PACKAGE_URL:-}"

  if [[ -z "${url}" ]]; then
    if [[ -z "${XUI_RELEASE_VERSION}" ]]; then
      XUI_RELEASE_VERSION="$(latest_version)"
    fi
    [[ -n "${XUI_RELEASE_VERSION}" ]] || fail "failed to resolve latest release version"
    url="https://github.com/${REPO}/releases/download/${XUI_RELEASE_VERSION}/z-ui-linux-${ARCH}.tar.gz"
  fi

  log "Downloading ${APP_NAME} package: ${url}" >&2
  curl -fL --retry 3 --connect-timeout 15 -o "${package_path}" "${url}"
  [[ -s "${package_path}" ]] || fail "downloaded package is empty"
  tar -tzf "${package_path}" >/dev/null
  printf '%s\n' "${package_path}"
}

random_hex() {
  local bytes="$1"
  od -An -N"${bytes}" -tx1 /dev/urandom | tr -d ' \n'
}

generate_username() {
  printf 'xui_%s\n' "$(random_hex 4 | cut -c1-8)"
}

generate_password() {
  random_hex 18
}

port_in_use() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltn "( sport = :${port} )" | grep -q ":${port}"
    return
  fi
  timeout 1 bash -c ":</dev/tcp/127.0.0.1/${port}" >/dev/null 2>&1
}

generate_port() {
  local n port
  for _ in $(seq 1 100); do
    n="$(od -An -N2 -tu2 /dev/urandom | tr -d ' ')"
    port=$((10000 + n % 50000))
    [[ "${port}" -ge 10000 && "${port}" -le 59999 ]] || continue
    [[ "${port}" -ne 54321 ]] || continue
    if ! port_in_use "${port}"; then
      printf '%s\n' "${port}"
      return
    fi
  done
  fail "failed to allocate an unused web port"
}

backup_existing_db() {
  mkdir -p "${CONFIG_DIR}"
  chmod 700 "${CONFIG_DIR}"
  if [[ -f "${DB_PATH}" ]]; then
    local timestamp backup_path
    timestamp="$(date -u +%Y%m%d%H%M%S)"
    backup_path="${DB_PATH}.bak.${timestamp}"
    mv "${DB_PATH}" "${backup_path}"
    chmod 600 "${backup_path}"
    log "Existing database backed up to ${backup_path}"
  fi
}

install_files() {
  local package_path="$1"
  local tmp_dir="$2"
  local extract_dir="${tmp_dir}/extract"
  local source_dir

  mkdir -p "${extract_dir}"
  tar -xzf "${package_path}" -C "${extract_dir}"

  if [[ -d "${extract_dir}/x-ui" ]]; then
    source_dir="${extract_dir}/x-ui"
  else
    source_dir="${extract_dir}"
  fi

  [[ -x "${source_dir}/x-ui" || -f "${source_dir}/x-ui" ]] || fail "package missing x-ui binary"
  [[ -f "${source_dir}/x-ui.sh" ]] || fail "package missing x-ui management script"
  [[ -f "${source_dir}/x-ui.service" ]] || fail "package missing systemd service"

  rm -rf "${INSTALL_DIR}"
  mkdir -p "${INSTALL_DIR}"
  cp -a "${source_dir}/." "${INSTALL_DIR}/"
  chmod 755 "${INSTALL_DIR}/x-ui"
  chmod 755 "${INSTALL_DIR}/x-ui.sh"
  if [[ -f "${INSTALL_DIR}/bin/xray-linux-${ARCH}" ]]; then
    chmod 755 "${INSTALL_DIR}/bin/xray-linux-${ARCH}"
  fi

  install -m 0755 "${INSTALL_DIR}/x-ui.sh" "${COMMAND_PATH}"
  install -m 0644 "${INSTALL_DIR}/x-ui.service" "${SERVICE_PATH}"
}

initialize_panel() {
  USERNAME="$(generate_username)"
  PASSWORD="$(generate_password)"
  PORT="$(generate_port)"

  "${INSTALL_DIR}/x-ui" setting -username "${USERNAME}" -password "${PASSWORD}" >/dev/null
  "${INSTALL_DIR}/x-ui" setting -port "${PORT}" >/dev/null

  chmod 700 "${CONFIG_DIR}"
  if [[ -f "${DB_PATH}" ]]; then
    chmod 600 "${DB_PATH}"
  fi
}

start_service() {
  systemctl daemon-reload
  systemctl enable x-ui
  if ! systemctl restart x-ui; then
    journalctl -u x-ui -n 50 --no-pager || true
    fail "failed to start x-ui service"
  fi
  for _ in $(seq 1 30); do
    if curl -fsS "http://127.0.0.1:${PORT}/" >/dev/null 2>&1; then
      return
    fi
    sleep 1
  done
  journalctl -u x-ui -n 50 --no-pager || true
  fail "x-ui service started but panel did not become reachable"
}

server_ip() {
  local ip=""
  ip="$(curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null || true)"
  if [[ -z "${ip}" ]]; then
    ip="$(curl -fsS --max-time 5 https://ifconfig.me 2>/dev/null || true)"
  fi
  if [[ -z "${ip}" ]]; then
    ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  fi
  printf '%s\n' "${ip:-127.0.0.1}"
}

print_success() {
  local ip
  ip="$(server_ip)"
  cat <<EOF
================================
x-ui installed successfully

Panel URL:
http://${ip}:${PORT}

Username:
${USERNAME}

Password:
${PASSWORD}

Command:
x-ui

Config:
 /etc/x-ui/x-ui.db

Service:
 systemctl status x-ui
================================
EOF
}

main() {
  require_root
  detect_os
  detect_arch
  install_dependencies

  local package_path
  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT

  package_path="$(download_package "${TMP_DIR}")"
  backup_existing_db
  install_files "${package_path}" "${TMP_DIR}"
  initialize_panel
  start_service
  print_success
}

main "$@"
