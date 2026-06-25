#!/usr/bin/env bash
set -euo pipefail

APP_NAME="x-ui"
INSTALL_DIR="/usr/local/x-ui"
CONFIG_DIR="/etc/x-ui"
DB_PATH="${CONFIG_DIR}/x-ui.db"
SERVICE_PATH="/etc/systemd/system/x-ui.service"
COMMAND_PATH="/usr/bin/x-ui"
PORT_GUARD_SYNC_PATH="/usr/local/bin/zui-port-guard-sync"
PORT_GUARD_SERVICE_PATH="/etc/systemd/system/zui-port-guard-sync.service"
PORT_GUARD_TIMER_PATH="/etc/systemd/system/zui-port-guard-sync.timer"
XRAY_ACCESS_LOGROTATE_PATH="/etc/logrotate.d/x-ui-xray-access"
REPO="${XUI_REPO:-torr9522/z-ui}"
XUI_RELEASE_VERSION="v1.0.4"
TMP_DIR=""

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

print_header() {
  cat <<'EOF'
╔════════════════════════════════════╗
║            z-ui Installer          ║
║     Lightweight · Stable · Secure  ║
╚════════════════════════════════════╝
EOF
}

step_title() {
  printf '\n【STEP %s】%s\n' "$1" "$2"
}

ok_line() {
  printf '✔ %s\n' "$1"
}

info_line() {
  printf '%s\n' "$1"
}

progress_line() {
  printf 'Progress: [████████████████] 100%%\n'
}

box_row() {
  local label="$1"
  local value="$2"
  printf '║ %-9s: %-47s ║\n' "${label}" "${value}"
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
      apt-get install -y curl wget tar unzip ca-certificates systemd sqlite3 nftables logrotate
      ;;
    rhel)
      local pm="yum"
      if command -v dnf >/dev/null 2>&1; then
        pm="dnf"
      fi
      "${pm}" install -y curl wget tar unzip ca-certificates systemd sqlite nftables logrotate
      ;;
    *)
      fail "unsupported package manager for ${OS_FAMILY}"
      ;;
  esac
}

check_env() {
  step_title 1 "环境检测"
  require_root
  detect_os
  detect_arch
  install_dependencies
  for cmd in curl tar systemctl nft logrotate; do
    command -v "${cmd}" >/dev/null 2>&1 || fail "missing command after dependency install: ${cmd}"
  done
  ok_line "curl"
  ok_line "tar"
  ok_line "systemd"
  ok_line "nftables"
  ok_line "logrotate"
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

download_package() {
  local tmp_dir="$1"
  local package_path="${tmp_dir}/x-ui.tar.gz"
  local url="https://github.com/${REPO}/releases/download/${XUI_RELEASE_VERSION}/z-ui-linux-${ARCH}.tar.gz"

  step_title 2 "下载资源" >&2
  info_line "Source: GitHub Releases" >&2
  info_line "Status: downloading..." >&2
  curl -fL --retry 3 --connect-timeout 15 -o "${package_path}" "${url}" >&2
  [[ -s "${package_path}" ]] || fail "downloaded package is empty"
  tar -tzf "${package_path}" >/dev/null
  progress_line >&2
  info_line "Status: downloaded" >&2
  printf '%s\n' "${package_path}"
}

random_hex() {
  local bytes="$1"
  od -An -N"${bytes}" -tx1 /dev/urandom | tr -d ' \n'
}

random_alnum() {
  local length="$1"
  local value=""
  while [[ "${#value}" -lt "${length}" ]]; do
    value+="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c "${length}" || true)"
  done
  printf '%s' "${value:0:${length}}"
}

generate_username() {
  random_alnum 10
  printf '\n'
}

generate_password() {
  random_alnum 18
  printf '\n'
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
  if [[ -f "${INSTALL_DIR}/scripts/zui-port-guard-sync" ]]; then
    install -m 0755 "${INSTALL_DIR}/scripts/zui-port-guard-sync" "${PORT_GUARD_SYNC_PATH}"
  fi
  if [[ -f "${INSTALL_DIR}/zui-port-guard-sync.service" ]]; then
    install -m 0644 "${INSTALL_DIR}/zui-port-guard-sync.service" "${PORT_GUARD_SERVICE_PATH}"
  fi
  if [[ -f "${INSTALL_DIR}/zui-port-guard-sync.timer" ]]; then
    install -m 0644 "${INSTALL_DIR}/zui-port-guard-sync.timer" "${PORT_GUARD_TIMER_PATH}"
  fi
}

install_xray_access_logrotate() {
  local source_path="${INSTALL_DIR}/packaging/logrotate/x-ui-xray-access"
  mkdir -p /var/log/xray
  if [[ -f "${source_path}" ]]; then
    install -m 0644 "${source_path}" "${XRAY_ACCESS_LOGROTATE_PATH}"
  else
    cat >"${XRAY_ACCESS_LOGROTATE_PATH}" <<'EOF'
/var/log/xray/access.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
EOF
    chmod 0644 "${XRAY_ACCESS_LOGROTATE_PATH}"
  fi
  if systemctl list-unit-files logrotate.timer >/dev/null 2>&1; then
    systemctl enable --now logrotate.timer >/dev/null 2>&1 || true
  fi
}

bootstrap_zui_nftables() {
  local check_path="${INSTALL_DIR}/scripts/nft_check.sh"
  local init_path="${INSTALL_DIR}/scripts/nft_init_zui.sh"
  if [[ -x "${check_path}" && -x "${init_path}" ]]; then
    "${check_path}" >/dev/null 2>&1 || log "z-ui nftables check skipped"
    "${init_path}" >/dev/null 2>&1 || log "z-ui nftables init skipped"
  fi
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

start_port_guard() {
  mkdir -p /var/lib/z-ui/port-guard /var/log/z-ui
  if [[ -x "${PORT_GUARD_SYNC_PATH}" && -f "${PORT_GUARD_SERVICE_PATH}" && -f "${PORT_GUARD_TIMER_PATH}" ]]; then
    systemctl daemon-reload
    systemctl enable --now zui-port-guard-sync.timer >/dev/null 2>&1 || true
    systemctl start zui-port-guard-sync.service >/dev/null 2>&1 || true
  fi
}

install_service() {
  local package_path="$1"
  step_title 3 "安装系统服务"
  backup_existing_db
  install_files "${package_path}" "${TMP_DIR}"
  ok_line "extract package"
  ok_line "install binary"
  install_xray_access_logrotate
  bootstrap_zui_nftables
  ok_line "create systemd service"
}

init_config() {
  step_title 4 "初始化配置"
  initialize_panel
  ok_line "generate username"
  ok_line "generate password"
  ok_line "allocate port"
  ok_line "init database"
  start_service
  start_port_guard
  ok_line "enable auto-start"
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

print_success_box() {
  local ip panel_url version commit branch build_time
  ip="$(server_ip)"
  panel_url="http://${ip}:${PORT}"
  version="$("${INSTALL_DIR}/x-ui" version 2>/dev/null | sed -n 's/^Version: //p' | head -n1)"
  commit="$("${INSTALL_DIR}/x-ui" version 2>/dev/null | sed -n 's/^Commit: //p' | head -n1)"
  branch="$("${INSTALL_DIR}/x-ui" version 2>/dev/null | sed -n 's/^Branch: //p' | head -n1)"
  build_time="$("${INSTALL_DIR}/x-ui" version 2>/dev/null | sed -n 's/^BuildTime: //p' | head -n1)"
  version="${version:-未知}"
  commit="${commit:-未知}"
  branch="${branch:-未知}"
  build_time="${build_time:-未知}"
  cat <<'EOF'
【STEP 5】安装完成
╔════════════════════════════════════════════════════════════╗
║                  INSTALLATION SUCCESS                    ║
╠════════════════════════════════════════════════════════════╣
EOF
  box_row "Panel URL" "${panel_url}"
  box_row "Username" "${USERNAME}"
  box_row "Password" "${PASSWORD}"
  box_row "Port" "${PORT}"
  box_row "Version" "${version}"
  box_row "Commit" "${commit}"
  cat <<'EOF'
╠════════════════════════════════════════════════════════════╣
EOF
  box_row "Branch" "${branch}"
  box_row "BuildTime" "${build_time}"
  box_row "Command" "x-ui"
  box_row "Config" "/etc/x-ui/x-ui.db"
  cat <<'EOF'
╚════════════════════════════════════════════════════════════╝
EOF
}

main() {
  local package_path
  print_header
  check_env

  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT

  package_path="$(download_package "${TMP_DIR}")"
  install_service "${package_path}"
  init_config
  print_success_box
}

main "$@"
