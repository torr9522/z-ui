#!/usr/bin/env bash
set -euo pipefail

APP_NAME="x-ui"
INSTALL_DIR="/usr/local/x-ui"
BIN="${INSTALL_DIR}/x-ui"
CONFIG_DIR="/etc/x-ui"
DB_PATH="${CONFIG_DIR}/x-ui.db"
SERVICE_NAME="x-ui"
REPO="${XUI_REPO:-FranzKafkaYu/x-ui}"

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

require_root() {
  [[ "${EUID}" -eq 0 ]] || fail "x-ui command must be run as root"
}

require_installed() {
  [[ -x "${BIN}" ]] || fail "x-ui is not installed at ${INSTALL_DIR}"
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

sqlite_value() {
  local query="$1"
  [[ -f "${DB_PATH}" ]] || return 0
  sqlite3 "${DB_PATH}" "${query}" 2>/dev/null || true
}

current_port() {
  local port
  port="$(sqlite_value "select value from settings where key='webPort' limit 1;")"
  printf '%s\n' "${port:-unknown}"
}

current_username() {
  local username
  username="$(sqlite_value "select username from users order by id asc limit 1;")"
  printf '%s\n' "${username:-unknown}"
}

cmd_start() {
  require_installed
  systemctl start "${SERVICE_NAME}"
  systemctl is-active --quiet "${SERVICE_NAME}"
  printf 'x-ui started\n'
}

cmd_stop() {
  require_installed
  systemctl stop "${SERVICE_NAME}"
  printf 'x-ui stopped\n'
}

cmd_restart() {
  require_installed
  systemctl restart "${SERVICE_NAME}"
  systemctl is-active --quiet "${SERVICE_NAME}"
  printf 'x-ui restarted\n'
}

cmd_status() {
  systemctl status "${SERVICE_NAME}" --no-pager
}

cmd_enable() {
  systemctl enable "${SERVICE_NAME}"
}

cmd_disable() {
  systemctl disable "${SERVICE_NAME}"
}

cmd_log() {
  journalctl -u "${SERVICE_NAME}" -e --no-pager -f
}

cmd_reset_user() {
  require_installed
  local username password
  username="$(generate_username)"
  password="$(generate_password)"
  "${BIN}" setting -username "${username}" -password "${password}" >/dev/null
  chmod 700 "${CONFIG_DIR}" 2>/dev/null || true
  chmod 600 "${DB_PATH}" 2>/dev/null || true
  systemctl restart "${SERVICE_NAME}"
  cat <<EOF
Username:
${username}

Password:
${password}
EOF
}

cmd_reset_port() {
  require_installed
  local port
  port="$(generate_port)"
  "${BIN}" setting -port "${port}" >/dev/null
  systemctl restart "${SERVICE_NAME}"
  cat <<EOF
Panel URL:
http://$(server_ip):${port}

Port:
${port}
EOF
}

cmd_info() {
  require_installed
  cat <<EOF
Panel URL:
http://$(server_ip):$(current_port)

Username:
$(current_username)

Password:
not displayed

Config:
 ${DB_PATH}

Service:
 systemctl status x-ui
EOF
}

cmd_update() {
  require_root
  bash <(curl -fsSL "https://raw.githubusercontent.com/${REPO}/master/install.sh")
}

cmd_uninstall() {
  require_root
  systemctl stop "${SERVICE_NAME}" >/dev/null 2>&1 || true
  systemctl disable "${SERVICE_NAME}" >/dev/null 2>&1 || true
  rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
  systemctl daemon-reload
  rm -rf "${INSTALL_DIR}"
  rm -f "/usr/bin/x-ui"
  printf 'x-ui uninstalled. Database preserved at %s\n' "${DB_PATH}"
}

usage() {
  cat <<'EOF'
x-ui command usage:
  x-ui start        Start x-ui
  x-ui stop         Stop x-ui
  x-ui restart      Restart x-ui
  x-ui status       Show service status
  x-ui enable       Enable service at boot
  x-ui disable      Disable service at boot
  x-ui log          Follow service logs
  x-ui reset-user   Generate a new random username and password
  x-ui reset-port   Generate a new random panel port
  x-ui info         Show panel URL and username
  x-ui update       Reinstall from latest release
  x-ui uninstall    Uninstall service and binaries, keep database
EOF
}

main() {
  require_root
  local cmd="${1:-}"
  case "${cmd}" in
    start) cmd_start ;;
    stop) cmd_stop ;;
    restart) cmd_restart ;;
    status) cmd_status ;;
    enable) cmd_enable ;;
    disable) cmd_disable ;;
    log) cmd_log ;;
    reset-user) cmd_reset_user ;;
    reset-port) cmd_reset_port ;;
    info) cmd_info ;;
    update) cmd_update ;;
    uninstall) cmd_uninstall ;;
    "" | help | -h | --help) usage ;;
    *) usage; fail "unknown command: ${cmd}" ;;
  esac
}

main "$@"
