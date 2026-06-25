#!/usr/bin/env bash
set -euo pipefail

APP_NAME="x-ui"
INSTALL_DIR="/usr/local/x-ui"
BIN="${INSTALL_DIR}/x-ui"
CONFIG_DIR="/etc/x-ui"
DB_PATH="${CONFIG_DIR}/x-ui.db"
CERTS_DIR="${CONFIG_DIR}/certs"
SERVICE_NAME="x-ui"
REPO="${XUI_REPO:-torr9522/z-ui}"
BRANCH="${XUI_BRANCH:-z-ui}"
DEFAULT_ACME_DOMAIN="cshtps.527270.xyz"
RENEW_SERVICE="/etc/systemd/system/x-ui-cert-renew.service"
RENEW_TIMER="/etc/systemd/system/x-ui-cert-renew.timer"
PORT_GUARD_SYNC="/usr/local/bin/zui-port-guard-sync"
PORT_GUARD_SERVICE="/etc/systemd/system/zui-port-guard-sync.service"
PORT_GUARD_TIMER="/etc/systemd/system/zui-port-guard-sync.timer"
PORT_GUARD_TABLE="zui_port_guard"
PORT_GUARD_LOG="/var/log/z-ui/port-guard.log"

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

ensure_certs_dir() {
  mkdir -p "${CERTS_DIR}"
  chmod 700 "${CERTS_DIR}"
}

cert_dir() {
  local name="$1"
  printf '%s/%s\n' "${CERTS_DIR}" "${name}"
}

is_ignored_cert_dir() {
  local name="$1"
  case "${name}" in
    deleted.*|*.bak.*|backup.*|tmp.*|test.*) return 0 ;;
    *) return 1 ;;
  esac
}

cert_meta_file() {
  local name="$1"
  printf '%s/meta.json\n' "$(cert_dir "$name")"
}

backup_path() {
  local prefix="$1"
  printf '%s.%s\n' "${prefix}" "$(date +%Y%m%d%H%M%S)"
}

cert_list_names() {
  ensure_certs_dir
  find "${CERTS_DIR}" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | while IFS= read -r name; do
    if is_ignored_cert_dir "${name}"; then
      continue
    fi
    printf '%s\n' "${name}"
  done | sort
}

cert_meta_value() {
  local meta_file="$1"
  local key="$2"
  python3 - "$meta_file" "$key" <<'PY'
import json, sys
path, key = sys.argv[1], sys.argv[2]
try:
    with open(path, 'r', encoding='utf-8') as handle:
        data = json.load(handle)
    value = data.get(key, "")
    if isinstance(value, bool):
        print("true" if value else "false")
    else:
        print(value)
except Exception:
    print("")
PY
}

write_cert_meta() {
  local meta_file="$1"
  local name="$2"
  local domain="$3"
  local issuer="$4"
  local cert_file="$5"
  local key_file="$6"
  local created_at="$7"
  local expire_at="$8"
  local auto_renew="$9"
  local cert_type="${10:-imported}"
  local last_renew_at="${11:-0}"
  python3 - "$meta_file" "$name" "$domain" "$issuer" "$cert_type" "$cert_file" "$key_file" "$created_at" "$expire_at" "$last_renew_at" "$auto_renew" <<'PY'
import json, sys
meta_file, name, domain, issuer, cert_type, cert_file, key_file, created_at, expire_at, last_renew_at, auto_renew = sys.argv[1:]
payload = {
    "name": name,
    "domain": domain,
    "issuer": issuer,
    "type": cert_type,
    "certFile": cert_file,
    "keyFile": key_file,
    "createdAt": int(created_at),
    "expireAt": int(expire_at),
    "lastRenewAt": int(last_renew_at),
    "autoRenew": auto_renew.lower() == "true",
}
with open(meta_file, "w", encoding="utf-8") as handle:
    json.dump(payload, handle, ensure_ascii=True, indent=2)
PY
  chmod 600 "${meta_file}"
}

openssl_enddate() {
  local cert_file="$1"
  command -v openssl >/dev/null 2>&1 || return 0
  openssl x509 -enddate -noout -in "${cert_file}" 2>/dev/null | sed 's/^notAfter=//'
}

openssl_expire_epoch() {
  local cert_file="$1"
  local enddate
  enddate="$(openssl_enddate "${cert_file}")"
  [[ -n "${enddate}" ]] || {
    printf '0\n'
    return
  }
  date -d "${enddate}" +%s 2>/dev/null || printf '0\n'
}

days_remaining() {
  local expire_at="$1"
  if [[ ! "${expire_at}" =~ ^[0-9]+$ ]] || [[ "${expire_at}" -le 0 ]]; then
    printf '0\n'
    return
  fi
  local now days
  now="$(date +%s)"
  days=$(((expire_at - now) / 86400))
  [[ "${days}" -gt 0 ]] || days=0
  printf '%s\n' "${days}"
}

cert_meta_set() {
  local meta_file="$1"
  local key="$2"
  local value="$3"
  python3 - "$meta_file" "$key" "$value" <<'PY'
import json, sys
path, key, value = sys.argv[1:]
with open(path, "r", encoding="utf-8") as handle:
    data = json.load(handle)
if value.lower() in ("true", "false"):
    data[key] = value.lower() == "true"
else:
    try:
        data[key] = int(value)
    except ValueError:
        data[key] = value
with open(path, "w", encoding="utf-8") as handle:
    json.dump(data, handle, ensure_ascii=True, indent=2)
PY
  chmod 600 "${meta_file}"
}

normalize_cert_meta() {
  local name="$1"
  local dir meta cert_file key_file domain issuer cert_type created_at expire_at last_renew_at auto_renew
  dir="$(cert_dir "${name}")"
  meta="$(cert_meta_file "${name}")"
  [[ -f "${meta}" ]] || return 0
  cert_file="$(cert_meta_value "${meta}" certFile)"
  key_file="$(cert_meta_value "${meta}" keyFile)"
  domain="$(cert_meta_value "${meta}" domain)"
  issuer="$(cert_meta_value "${meta}" issuer)"
  cert_type="$(cert_meta_value "${meta}" type)"
  created_at="$(cert_meta_value "${meta}" createdAt)"
  expire_at="$(cert_meta_value "${meta}" expireAt)"
  last_renew_at="$(cert_meta_value "${meta}" lastRenewAt)"
  auto_renew="$(cert_meta_value "${meta}" autoRenew)"
  cert_file="${cert_file:-${dir}/fullchain.pem}"
  key_file="${key_file:-${dir}/privkey.pem}"
  domain="${domain:-${name}}"
  issuer="${issuer:-manual}"
  created_at="${created_at:-0}"
  expire_at="${expire_at:-$(openssl_expire_epoch "${cert_file}")}"
  last_renew_at="${last_renew_at:-0}"
  auto_renew="${auto_renew:-false}"
  if [[ -z "${cert_type}" ]]; then
    if [[ "${name}" == ip-* ]]; then
      cert_type="ip"
    elif [[ "${issuer}" == "acme.sh" ]]; then
      cert_type="domain"
    else
      cert_type="imported"
    fi
  fi
  write_cert_meta "${meta}" "${name}" "${domain}" "${issuer}" "${cert_file}" "${key_file}" "${created_at}" "${expire_at}" "${auto_renew}" "${cert_type}" "${last_renew_at}"
}

cert_in_use() {
  local cert_file="$1"
  local key_file="$2"
  if [[ -f "${DB_PATH}" ]]; then
    if sqlite3 "${DB_PATH}" "select value from settings where key in ('webCertFile','webKeyFile');" 2>/dev/null | grep -Fxq -e "${cert_file}" -e "${key_file}"; then
      return 0
    fi
    if sqlite3 "${DB_PATH}" "select stream_settings from inbounds;" 2>/dev/null | grep -Fq -e "${cert_file}" -e "${key_file}"; then
      return 0
    fi
  fi
  return 1
}

reload_after_cert_change() {
  local cert_file="$1"
  local key_file="$2"
  if cert_in_use "${cert_file}" "${key_file}"; then
    systemctl reload "${SERVICE_NAME}" >/dev/null 2>&1 || systemctl restart "${SERVICE_NAME}" >/dev/null 2>&1 || true
    systemctl is-active --quiet "${SERVICE_NAME}" || systemctl start "${SERVICE_NAME}" >/dev/null 2>&1 || true
  fi
}

install_cert_renew_timer() {
  cat >"${RENEW_SERVICE}" <<'EOF'
[Unit]
Description=x-ui certificate renewal

[Service]
Type=oneshot
ExecStart=/usr/bin/x-ui cert renew
EOF
  cat >"${RENEW_TIMER}" <<'EOF'
[Unit]
Description=Daily x-ui certificate renewal

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true

[Install]
WantedBy=timers.target
EOF
  systemctl daemon-reload
  systemctl enable --now x-ui-cert-renew.timer >/dev/null
}

domain_ipv4s() {
  local domain="$1"
  python3 - "${domain}" <<'PY'
import socket, sys
domain = sys.argv[1]
values = set()
try:
    for item in socket.getaddrinfo(domain, None, socket.AF_INET, socket.SOCK_STREAM):
        values.add(item[4][0])
except Exception:
    pass
for value in sorted(values):
    print(value)
PY
}

acme_bin() {
  if command -v acme.sh >/dev/null 2>&1; then
    command -v acme.sh
    return
  fi
  if [[ -x "${HOME}/.acme.sh/acme.sh" ]]; then
    printf '%s\n' "${HOME}/.acme.sh/acme.sh"
  fi
}

ensure_acme_installed() {
  local bin
  bin="$(acme_bin || true)"
  if [[ -n "${bin}" ]]; then
    printf '%s\n' "${bin}"
    return
  fi
  command -v curl >/dev/null 2>&1 || fail "curl is required to install acme.sh"
  printf 'acme.sh not found, installing...\n' >&2
  curl -fsSL https://get.acme.sh | sh -s email="admin@${DEFAULT_ACME_DOMAIN}" >/dev/null
  bin="$(acme_bin || true)"
  [[ -n "${bin}" ]] || fail "acme.sh installation failed"
  printf '%s\n' "${bin}"
}

choose_certificate() {
  local names=()
  while IFS= read -r line; do
    [[ -n "${line}" ]] && names+=("${line}")
  done < <(cert_list_names)

  if [[ "${#names[@]}" -eq 0 ]]; then
    printf 'No certificates found in %s\n' "${CERTS_DIR}"
    return 1
  fi

  local i=1
  for name in "${names[@]}"; do
    printf '%d. %s\n' "${i}" "${name}" >&2
    i=$((i + 1))
  done

  local choice
  read -r -p "Select certificate [1-${#names[@]}]: " choice
  [[ "${choice}" =~ ^[0-9]+$ ]] || fail "invalid selection"
  [[ "${choice}" -ge 1 && "${choice}" -le "${#names[@]}" ]] || fail "selection out of range"
  printf '%s\n' "${names[$((choice - 1))]}"
}

current_port() {
  local port
  port="$(sqlite_value "select value from settings where key='webPort' limit 1;")"
  printf '%s\n' "${port:-unknown}"
}

current_setting() {
  local key="$1"
  sqlite_value "select value from settings where key='${key}' limit 1;"
}

current_web_cert() {
  current_setting "webCertFile"
}

current_web_key() {
  current_setting "webKeyFile"
}

panel_https_enabled() {
  local cert_file key_file
  cert_file="$(current_web_cert)"
  key_file="$(current_web_key)"
  [[ -n "${cert_file}" && -n "${key_file}" && -f "${cert_file}" && -f "${key_file}" ]]
}

cert_domain_from_file() {
  local cert_file="$1"
  local meta_file domain
  meta_file="$(dirname "${cert_file}")/meta.json"
  if [[ -f "${meta_file}" ]]; then
    domain="$(cert_meta_value "${meta_file}" domain)"
    if [[ -n "${domain}" ]]; then
      printf '%s\n' "${domain}"
      return
    fi
  fi
  if command -v openssl >/dev/null 2>&1 && [[ -f "${cert_file}" ]]; then
    domain="$(openssl x509 -noout -subject -in "${cert_file}" 2>/dev/null | sed -n 's/.*CN *= *//p' | sed 's#/$##' | head -n1)"
    if [[ -n "${domain}" ]]; then
      printf '%s\n' "${domain}"
      return
    fi
  fi
  server_ip
}

panel_scheme() {
  if panel_https_enabled; then
    printf 'https\n'
  else
    printf 'http\n'
  fi
}

panel_host() {
  if panel_https_enabled; then
    cert_domain_from_file "$(current_web_cert)"
  else
    server_ip
  fi
}

panel_url() {
  local port="${1:-$(current_port)}"
  printf '%s://%s:%s\n' "$(panel_scheme)" "$(panel_host)" "${port}"
}

current_username() {
  local username
  username="$(sqlite_value "select username from users order by id asc limit 1;")"
  printf '%s\n' "${username:-unknown}"
}

current_password() {
  local password
  password="$(sqlite_value "select password from users order by id asc limit 1;")"
  if [[ -n "${password}" ]]; then
    printf '%s\n' "${password}"
  else
    printf '未保存明文，请使用 x-ui reset-user 重置\n'
  fi
}

current_version() {
  if [[ -x "${BIN}" ]]; then
    "${BIN}" -v 2>/dev/null || printf '未知\n'
  else
    printf '未知\n'
  fi
}

binary_version_value() {
  local key="$1"
  if [[ -x "${BIN}" ]]; then
    "${BIN}" version 2>/dev/null | awk -F': ' -v key="${key}" '$1 == key {print $2; found=1} END {if (!found) print ""}'
  fi
}

current_commit() {
  local value
  value="$(binary_version_value "Commit")"
  if [[ -n "${value}" ]]; then
    printf '%s\n' "${value}"
  elif [[ -f "${INSTALL_DIR}/COMMIT" ]]; then
    head -n1 "${INSTALL_DIR}/COMMIT"
  elif [[ -d "${INSTALL_DIR}/.git" ]] && command -v git >/dev/null 2>&1; then
    git -C "${INSTALL_DIR}" rev-parse --short HEAD 2>/dev/null || printf '未知\n'
  else
    printf '未知\n'
  fi
}

current_branch() {
  local value
  value="$(binary_version_value "Branch")"
  if [[ -n "${value}" ]]; then
    printf '%s\n' "${value}"
  elif [[ -d "${INSTALL_DIR}/.git" ]] && command -v git >/dev/null 2>&1; then
    git -C "${INSTALL_DIR}" branch --show-current 2>/dev/null || printf '未知\n'
  else
    printf '未知\n'
  fi
}

current_build_time() {
  local value
  value="$(binary_version_value "BuildTime")"
  if [[ -n "${value}" ]]; then
    printf '%s\n' "${value}"
  else
    printf '未知\n'
  fi
}

service_status() {
  systemctl is-active "${SERVICE_NAME}" 2>/dev/null || printf 'unknown\n'
}

service_status_text() {
  case "$1" in
    active) printf '运行中' ;;
    inactive) printf '未运行' ;;
    activating) printf '启动中' ;;
    failed) printf '失败' ;;
    unknown) printf '未知' ;;
    *) printf '%s' "$1" ;;
  esac
}

pause_return() {
  local _
  read -r -p "按回车返回" _ || true
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

restart_x_ui_service() {
  systemctl restart "${SERVICE_NAME}"
  systemctl is-active --quiet "${SERVICE_NAME}"
}

apply_panel_user() {
  require_installed
  local username="$1"
  local password="$2"
  [[ -n "${username}" ]] || fail "用户名不能为空"
  [[ -n "${password}" ]] || fail "密码不能为空"
  "${BIN}" setting -username "${username}" -password "${password}" >/dev/null
  chmod 700 "${CONFIG_DIR}" 2>/dev/null || true
  chmod 600 "${DB_PATH}" 2>/dev/null || true
  restart_x_ui_service
  cat <<EOF
新用户名：
${username}

新密码：
${password}

服务已重启，配置已生效。
EOF
}

cmd_set_user() {
  local username="${1:-}"
  local password="${2:-}"
  [[ -n "${username}" ]] || fail "用法: x-ui set-user <用户名> <密码>"
  [[ -n "${password}" ]] || fail "用法: x-ui set-user <用户名> <密码>"
  apply_panel_user "${username}" "${password}"
}

cmd_reset_user() {
  require_installed
  local username password
  while true; do
    username="$(generate_username)"
    password="$(generate_password)"
    if [[ "${username}" != "admin" || "${password}" != "admin" ]]; then
      break
    fi
  done
  apply_panel_user "${username}" "${password}"
}

validate_panel_port() {
  local port="$1"
  if [[ -z "${port}" ]]; then
    printf '端口不能为空。\n' >&2
    return 1
  fi
  if [[ ! "${port}" =~ ^[0-9]+$ ]]; then
    printf '端口必须是数字。\n' >&2
    return 1
  fi
  local port_num=$((10#${port}))
  if [[ "${port_num}" -lt 1 || "${port_num}" -gt 65535 ]]; then
    printf '端口范围必须是 1-65535。\n' >&2
    return 1
  fi
}

apply_panel_port() {
  require_installed
  local port="$1"
  validate_panel_port "${port}"
  "${BIN}" setting -port "${port}" >/dev/null
  restart_x_ui_service
  cat <<EOF
面板地址：
$(panel_url "${port}")

面板端口：
${port}

服务已重启，配置已生效。
EOF
}

cmd_set_port() {
  local port="${1:-}"
  validate_panel_port "${port}"
  apply_panel_port "${port}"
}

cmd_reset_port() {
  require_installed
  local port
  port="$(generate_port)"
  apply_panel_port "${port}"
}

cmd_info() {
  require_installed
  cat <<EOF
面板地址：
$(panel_url)

当前协议：
$(panel_scheme | tr '[:lower:]' '[:upper:]')

当前端口：
$(current_port)

当前用户名：
$(current_username)

当前密码：
$(current_password)

服务状态：
$(service_status_text "$(service_status)")

当前版本：
$(current_version)

当前提交：
$(current_commit)

当前分支：
$(current_branch)

构建时间：
$(current_build_time)

数据库：
 ${DB_PATH}

服务：
 systemctl status x-ui
EOF
}

cmd_panel_info_cn() {
  require_installed
  local scheme port username password status version commit branch build_time
  scheme="$(panel_scheme)"
  port="$(current_port)"
  username="$(current_username)"
  password="$(current_password)"
  status="$(service_status)"
  version="$(current_version)"
  commit="$(current_commit)"
  branch="$(current_branch)"
  build_time="$(current_build_time)"
  cat <<EOF
面板地址：$(panel_url "${port}")
当前协议：${scheme^^}
当前端口：${port}
当前用户名：${username}
当前密码：${password}
服务状态：$(service_status_text "${status}")
当前版本：${version}
当前提交：${commit}
当前分支：${branch}
构建时间：${build_time}
EOF
}

cmd_cert_list() {
  ensure_certs_dir
  local found=0
  while IFS= read -r name; do
    [[ -n "${name}" ]] || continue
    normalize_cert_meta "${name}"
    found=1
    local dir meta cert_file key_file domain issuer cert_type expire_at auto_renew last_renew_at remaining
    dir="$(cert_dir "${name}")"
    meta="$(cert_meta_file "${name}")"
    cert_file="${dir}/fullchain.pem"
    key_file="${dir}/privkey.pem"
    domain="${name}"
    issuer="manual"
    cert_type="imported"
    expire_at="0"
    auto_renew="false"
    last_renew_at="0"
    if [[ -f "${meta}" ]]; then
      domain="$(cert_meta_value "${meta}" domain)"
      issuer="$(cert_meta_value "${meta}" issuer)"
      cert_type="$(cert_meta_value "${meta}" type)"
      cert_file="$(cert_meta_value "${meta}" certFile)"
      key_file="$(cert_meta_value "${meta}" keyFile)"
      expire_at="$(cert_meta_value "${meta}" expireAt)"
      auto_renew="$(cert_meta_value "${meta}" autoRenew)"
      last_renew_at="$(cert_meta_value "${meta}" lastRenewAt)"
    fi
    cert_type="${cert_type:-imported}"
    last_renew_at="${last_renew_at:-0}"
    remaining="$(days_remaining "${expire_at}")"
    printf 'name: %s\n' "${name}"
    printf 'domain: %s\n' "${domain}"
    printf 'issuer: %s\n' "${issuer}"
    printf 'type: %s\n' "${cert_type}"
    printf 'certFile: %s\n' "${cert_file}"
    printf 'keyFile: %s\n' "${key_file}"
    printf 'expireAt: %s\n' "${expire_at}"
    printf 'daysRemaining: %s\n' "${remaining}"
    printf 'lastRenewAt: %s\n' "${last_renew_at}"
    printf 'autoRenew: %s\n' "${auto_renew}"
    if [[ -f "${dir}/fullchain.pem" ]]; then
      local enddate
      enddate="$(openssl_enddate "${dir}/fullchain.pem")"
      if [[ -n "${enddate}" ]]; then
        printf 'opensslExpire: %s\n' "${enddate}"
      fi
    fi
    printf '\n'
  done < <(cert_list_names)
  if [[ "${found}" -eq 0 ]]; then
    printf 'No certificates found in %s\n' "${CERTS_DIR}"
  fi
}

cmd_cert_import() {
  ensure_certs_dir
  local name fullchain_source key_source target_dir backup_dir domain
  read -r -p "Certificate name: " name
  [[ -n "${name}" ]] || fail "certificate name is required"
  read -r -p "fullchain.pem path: " fullchain_source
  read -r -p "privkey.pem path: " key_source
  [[ -f "${fullchain_source}" ]] || fail "certificate file not found: ${fullchain_source}"
  [[ -f "${key_source}" ]] || fail "private key file not found: ${key_source}"
  command -v openssl >/dev/null 2>&1 || fail "openssl is required to validate certificate and private key"

  local cert_mod key_mod
  cert_mod="$(openssl x509 -noout -modulus -in "${fullchain_source}" 2>/dev/null || true)"
  key_mod="$(openssl rsa -noout -modulus -in "${key_source}" 2>/dev/null || true)"
  if [[ -z "${cert_mod}" || -z "${key_mod}" || "${cert_mod}" != "${key_mod}" ]]; then
    fail "证书与私钥不匹配"
  fi

  target_dir="$(cert_dir "${name}")"
  if [[ -d "${target_dir}" ]]; then
    local overwrite
    read -r -p "Certificate exists. Overwrite? [y/N]: " overwrite
    if [[ ! "${overwrite}" =~ ^[Yy]$ ]]; then
      printf 'Import cancelled\n'
      return 0
    fi
    backup_dir="$(backup_path "${CERTS_DIR}/${name}.bak")"
    mv "${target_dir}" "${backup_dir}"
  fi

  mkdir -p "${target_dir}"
  cp "${fullchain_source}" "${target_dir}/fullchain.pem"
  cp "${key_source}" "${target_dir}/privkey.pem"
  chmod 0644 "${target_dir}/fullchain.pem"
  chmod 0600 "${target_dir}/privkey.pem"

  domain="$(openssl x509 -noout -subject -in "${target_dir}/fullchain.pem" 2>/dev/null | sed -n 's/.*CN *= *//p' | sed 's#/$##' | head -n1)"
  domain="${domain:-${name}}"
  write_cert_meta "${target_dir}/meta.json" "${name}" "${domain}" "manual" "${target_dir}/fullchain.pem" "${target_dir}/privkey.pem" "$(date +%s)" "$(openssl_expire_epoch "${target_dir}/fullchain.pem")" "false" "imported" "0"
  chmod 700 "${target_dir}"
  printf 'Certificate imported: %s\n' "${target_dir}"
}

cmd_cert_delete() {
  ensure_certs_dir
  local name
  name="$(choose_certificate)" || return 0
  local target_dir backup_dir confirm
  target_dir="$(cert_dir "${name}")"
  read -r -p "Delete certificate ${name}? [y/N]: " confirm
  if [[ ! "${confirm}" =~ ^[Yy]$ ]]; then
    printf 'Delete cancelled\n'
    return 0
  fi
  backup_dir="$(backup_path "${CERTS_DIR}/deleted.${name}")"
  mv "${target_dir}" "${backup_dir}"
  printf 'Certificate moved to backup: %s\n' "${backup_dir}"
}

cmd_cert_set_panel_https() {
  require_installed
  ensure_certs_dir
  local name dir cert_file key_file
  name="$(choose_certificate)" || return 0
  dir="$(cert_dir "${name}")"
  cert_file="${dir}/fullchain.pem"
  key_file="${dir}/privkey.pem"
  [[ -f "${cert_file}" ]] || fail "certificate file missing: ${cert_file}"
  [[ -f "${key_file}" ]] || fail "key file missing: ${key_file}"

  sqlite3 "${DB_PATH}" "delete from settings where key='webCertFile'; insert into settings(key, value) values('webCertFile', '${cert_file}');" >/dev/null
  sqlite3 "${DB_PATH}" "delete from settings where key='webKeyFile'; insert into settings(key, value) values('webKeyFile', '${key_file}');" >/dev/null

  printf 'Panel HTTPS certificate set:\n'
  printf 'webCertFile=%s\n' "${cert_file}"
  printf 'webKeyFile=%s\n' "${key_file}"
  local confirm
  read -r -p "Restart panel now? [y/N]: " confirm
  if [[ "${confirm}" =~ ^[Yy]$ ]]; then
    systemctl restart "${SERVICE_NAME}"
    systemctl is-active --quiet "${SERVICE_NAME}"
    printf 'x-ui restarted\n'
  else
    printf 'Restart required: systemctl restart x-ui\n'
  fi
}

cmd_cert_issue_acme() {
  ensure_certs_dir
  local domain="${DEFAULT_ACME_DOMAIN}"
  read -r -p "Domain [${DEFAULT_ACME_DOMAIN}]: " input_domain
  domain="${input_domain:-${DEFAULT_ACME_DOMAIN}}"
  [[ -n "${domain}" ]] || fail "domain is required"

  local public_ip dns_ips match
  public_ip="$(server_ip)"
  dns_ips="$(domain_ipv4s "${domain}" | paste -sd, -)"
  [[ -n "${dns_ips}" ]] || fail "domain ${domain} has no A record"
  match=0
  IFS=',' read -ra dns_array <<< "${dns_ips}"
  for ip in "${dns_array[@]}"; do
    if [[ "${ip}" == "${public_ip}" ]]; then
      match=1
      break
    fi
  done
  [[ "${match}" -eq 1 ]] || fail "domain ${domain} resolves to ${dns_ips}, current server public IP is ${public_ip}"
  printf 'Domain resolves to this server: %s -> %s\n' "${domain}" "${public_ip}"

  if port_in_use 80; then
    fail "port 80 is occupied; standalone HTTP-01 cannot continue"
  fi
  printf 'Port 80 is available\n'

  local acme
  acme="$(ensure_acme_installed)"
  printf 'Using acme.sh: %s\n' "${acme}"

  "${acme}" --set-default-ca --server letsencrypt >/dev/null
  "${acme}" --issue --standalone -d "${domain}"

  local target_dir backup_dir cert_file key_file
  target_dir="$(cert_dir "${domain}")"
  if [[ -d "${target_dir}" ]]; then
    backup_dir="$(backup_path "${CERTS_DIR}/${domain}.bak")"
    mv "${target_dir}" "${backup_dir}"
  fi
  mkdir -p "${target_dir}"
  chmod 700 "${target_dir}"

  cert_file="${target_dir}/fullchain.pem"
  key_file="${target_dir}/privkey.pem"
  "${acme}" --install-cert -d "${domain}" \
    --fullchain-file "${cert_file}" \
    --key-file "${key_file}" \
    --reloadcmd "systemctl reload ${SERVICE_NAME} >/dev/null 2>&1 || true"

  chmod 0644 "${cert_file}"
  chmod 0600 "${key_file}"
  write_cert_meta "${target_dir}/meta.json" "${domain}" "${domain}" "acme.sh" "${cert_file}" "${key_file}" "$(date +%s)" "$(openssl_expire_epoch "${cert_file}")" "true" "domain" "0"
  install_cert_renew_timer
  printf 'Certificate issued and installed: %s\n' "${target_dir}"
}

cmd_cert_issue_ip() {
  ensure_certs_dir
  local ip acme
  ip="$(server_ip)"
  [[ -n "${ip}" ]] || fail "failed to detect public IP"
  printf 'Public IP: %s\n' "${ip}"
  acme="$(ensure_acme_installed)"
  printf 'Using acme.sh: %s\n' "${acme}"
  "${acme}" --set-default-ca --server letsencrypt >/dev/null
  if "${acme}" --issue --standalone -d "${ip}" --yes-I-know-dns-manual-mode-enough-go-ahead-please >/tmp/x-ui-ip-cert.log 2>&1; then
    local name target_dir cert_file key_file
    name="ip-${ip}"
    target_dir="$(cert_dir "${name}")"
    mkdir -p "${target_dir}"
    chmod 700 "${target_dir}"
    cert_file="${target_dir}/fullchain.pem"
    key_file="${target_dir}/privkey.pem"
    "${acme}" --install-cert -d "${ip}" --fullchain-file "${cert_file}" --key-file "${key_file}" --reloadcmd "systemctl reload ${SERVICE_NAME} >/dev/null 2>&1 || true"
    chmod 0644 "${cert_file}"
    chmod 0600 "${key_file}"
    write_cert_meta "${target_dir}/meta.json" "${name}" "${ip}" "acme.sh" "${cert_file}" "${key_file}" "$(date +%s)" "$(openssl_expire_epoch "${cert_file}")" "true" "ip" "0"
    install_cert_renew_timer
    printf 'IP certificate issued and installed: %s\n' "${target_dir}"
  else
    cat /tmp/x-ui-ip-cert.log >&2 || true
    fail "当前 CA 不支持 IP 证书申请。建议使用域名：${DEFAULT_ACME_DOMAIN}"
  fi
}

cmd_cert_status() {
  ensure_certs_dir
  local found=0
  while IFS= read -r name; do
    [[ -n "${name}" ]] || continue
    normalize_cert_meta "${name}"
    found=1
    local meta domain issuer cert_type expire_at auto_renew remaining
    meta="$(cert_meta_file "${name}")"
    domain="${name}"
    issuer="manual"
    cert_type="imported"
    expire_at="0"
    auto_renew="false"
    if [[ -f "${meta}" ]]; then
      domain="$(cert_meta_value "${meta}" domain)"
      issuer="$(cert_meta_value "${meta}" issuer)"
      cert_type="$(cert_meta_value "${meta}" type)"
      expire_at="$(cert_meta_value "${meta}" expireAt)"
      auto_renew="$(cert_meta_value "${meta}" autoRenew)"
    fi
    cert_type="${cert_type:-imported}"
    remaining="$(days_remaining "${expire_at}")"
    printf '%s\n' "${name}"
    printf 'Type: %s\n' "${cert_type}"
    printf 'Issuer: %s\n' "${issuer}"
    printf 'Domain: %s\n' "${domain}"
    printf 'Days Remaining: %s days\n' "${remaining}"
    if [[ "${auto_renew}" == "true" ]]; then
      printf 'Auto Renew: ON\n\n'
    else
      printf 'Auto Renew: OFF\n\n'
    fi
  done < <(cert_list_names)
  [[ "${found}" -eq 1 ]] || printf 'No certificates found in %s\n' "${CERTS_DIR}"
  if systemctl list-timers --all x-ui-cert-renew.timer >/dev/null 2>&1; then
    systemctl list-timers --all x-ui-cert-renew.timer --no-pager || true
  fi
}

renew_one_certificate() {
  local name="$1"
  local meta dir cert_file key_file domain cert_type expire_at auto_renew remaining acme changed=0
  dir="$(cert_dir "${name}")"
  meta="$(cert_meta_file "${name}")"
  [[ -f "${meta}" ]] || return 0
  auto_renew="$(cert_meta_value "${meta}" autoRenew)"
  [[ "${auto_renew}" == "true" ]] || return 0
  expire_at="$(cert_meta_value "${meta}" expireAt)"
  remaining="$(days_remaining "${expire_at}")"
  [[ "${remaining}" -le 30 ]] || {
    printf 'Skip %s: %s days remaining\n' "${name}" "${remaining}"
    return 0
  }
  domain="$(cert_meta_value "${meta}" domain)"
  cert_type="$(cert_meta_value "${meta}" type)"
  cert_file="$(cert_meta_value "${meta}" certFile)"
  key_file="$(cert_meta_value "${meta}" keyFile)"
  cert_file="${cert_file:-${dir}/fullchain.pem}"
  key_file="${key_file:-${dir}/privkey.pem}"
  acme="$(ensure_acme_installed)"
  printf 'Renewing %s...\n' "${name}"
  if [[ "${cert_type}" == "ip" ]]; then
    if ! "${acme}" --renew -d "${domain}" 2>&1 | tee /tmp/x-ui-cert-renew.log; then
      if grep -q "Skipping. Next renewal time" /tmp/x-ui-cert-renew.log; then
        printf 'Skip %s: CA renewal window not reached\n' "${name}"
        return 0
      fi
      return 1
    fi
    "${acme}" --install-cert -d "${domain}" --fullchain-file "${cert_file}" --key-file "${key_file}" --reloadcmd "systemctl reload ${SERVICE_NAME} >/dev/null 2>&1 || true"
  else
    if ! "${acme}" --renew -d "${domain}" 2>&1 | tee /tmp/x-ui-cert-renew.log; then
      if grep -q "Skipping. Next renewal time" /tmp/x-ui-cert-renew.log; then
        printf 'Skip %s: CA renewal window not reached\n' "${name}"
        return 0
      fi
      return 1
    fi
    "${acme}" --install-cert -d "${domain}" --fullchain-file "${cert_file}" --key-file "${key_file}" --reloadcmd "systemctl reload ${SERVICE_NAME} >/dev/null 2>&1 || true"
  fi
  chmod 0644 "${cert_file}"
  chmod 0600 "${key_file}"
  cert_meta_set "${meta}" expireAt "$(openssl_expire_epoch "${cert_file}")"
  cert_meta_set "${meta}" lastRenewAt "$(date +%s)"
  changed=1
  if [[ "${changed}" -eq 1 ]]; then
    reload_after_cert_change "${cert_file}" "${key_file}"
  fi
}

cmd_cert_renew() {
  ensure_certs_dir
  install_cert_renew_timer
  local name
  while IFS= read -r name; do
    [[ -n "${name}" ]] || continue
    normalize_cert_meta "${name}"
    renew_one_certificate "${name}"
  done < <(cert_list_names)
}

cmd_cert_autorenew() {
  ensure_certs_dir
  local name state meta cert_type
  name="$(choose_certificate)" || return 0
  read -r -p "Auto renew for ${name} [on/off]: " state
  case "${state}" in
    on|ON|On|1|true|TRUE) state="true" ;;
    off|OFF|Off|0|false|FALSE) state="false" ;;
    *) fail "invalid auto renew state: ${state}" ;;
  esac
  meta="$(cert_meta_file "${name}")"
  [[ -f "${meta}" ]] || fail "meta.json not found for ${name}"
  normalize_cert_meta "${name}"
  cert_type="$(cert_meta_value "${meta}" type)"
  if [[ "${state}" == "true" && "${cert_type}" == "imported" ]]; then
    fail "Imported certificates do not support ACME renewal."
  fi
  cert_meta_set "${meta}" autoRenew "${state}"
  if [[ "${state}" == "true" ]]; then
    install_cert_renew_timer
  fi
  printf 'autoRenew for %s set to %s\n' "${name}" "${state}"
}

cmd_cert_discover_import() {
  ensure_certs_dir
  command -v openssl >/dev/null 2>&1 || fail "openssl is required"
  local candidates=()
  local certs=()
  local keys=()
  local sources=()

  discover_acme_dir() {
    local base="$1" src="$2"
    [[ -d "${base}" ]] || return 0
    local dir name cert key
    for dir in "${base}"/*/; do
      [[ -d "${dir}" ]] || continue
      name="$(basename "${dir}")"
      # letsencrypt
      if [[ "${src}" == "letsencrypt" ]]; then
        cert="${dir}fullchain.pem"
        key="${dir}privkey.pem"
      else
        # acme.sh ECC or RSA
        if [[ "${name}" == *_ecc ]]; then
          cert="${dir}fullchain.cer"
          key="${dir}${name%.ecc_}.key"
          key="${dir}${name%_ecc}.key"
        else
          cert="${dir}fullchain.cer"
          key="${dir}${name}.key"
        fi
      fi
      [[ -f "${cert}" && -f "${key}" ]] || continue
      # validate pair
      local cmod kmod
      cmod="$(openssl x509 -noout -modulus -in "${cert}" 2>/dev/null || true)"
      kmod="$(openssl rsa -noout -modulus -in "${key}" 2>/dev/null ||
              openssl ec  -noout -no_public -in "${key}" 2>/dev/null || true)"
      [[ -n "${cmod}" && "${cmod}" == "${kmod}" ]] || continue
      candidates+=("${name}")
      certs+=("${cert}")
      keys+=("${key}")
      sources+=("${src}")
    done
  }

  discover_acme_dir "/etc/letsencrypt/live"  "letsencrypt"
  discover_acme_dir "/root/.acme.sh"         "acme.sh"
  local home_dir
  for home_dir in /home/*/; do
    [[ -d "${home_dir}.acme.sh" ]] || continue
    discover_acme_dir "${home_dir}.acme.sh" "acme.sh"
  done

  if [[ "${#candidates[@]}" -eq 0 ]]; then
    printf '未发现系统证书（/etc/letsencrypt/live 或 ~/.acme.sh）\n'
    return 0
  fi

  printf '发现以下系统证书：\n'
  local i
  for i in "${!candidates[@]}"; do
    local expiry
    expiry="$(openssl x509 -enddate -noout -in "${certs[${i}]}" 2>/dev/null | sed 's/notAfter=//' || true)"
    printf '%d. [%s] %s (到期: %s)\n' "$((i+1))" "${sources[${i}]}" "${candidates[${i}]}" "${expiry:-unknown}"
  done

  local choice
  read -r -p "选择要导入的证书编号（0 取消）: " choice
  [[ "${choice}" =~ ^[0-9]+$ ]] || { printf '输入无效\n'; return 1; }
  [[ "${choice}" -eq 0 ]] && return 0
  [[ "${choice}" -ge 1 && "${choice}" -le "${#candidates[@]}" ]] || { printf '编号超出范围\n'; return 1; }

  local idx=$(( choice - 1 ))
  local src_cert="${certs[${idx}]}"
  local src_key="${keys[${idx}]}"
  local default_name="${candidates[${idx}]}"

  local name
  read -r -p "证书名称 [${default_name}]: " name
  name="${name:-${default_name}}"
  # sanitize: only keep a-z0-9._-
  name="$(printf '%s' "${name}" | tr -cd 'a-zA-Z0-9._-' | tr '[:upper:]' '[:lower:]' | sed 's/^\.\+//;s/\.\+$//;s/\.\{2,\}/./g')"
  [[ -n "${name}" ]] || fail "证书名称无效"

  local target_dir
  target_dir="$(cert_dir "${name}")"
  if [[ -d "${target_dir}" ]]; then
    local overwrite
    read -r -p "证书 ${name} 已存在，覆盖？[y/N]: " overwrite
    [[ "${overwrite}" =~ ^[Yy]$ ]] || { printf '已取消\n'; return 0; }
    local bak
    bak="$(backup_path "${CERTS_DIR}/${name}.bak")"
    mv "${target_dir}" "${bak}"
  fi
  mkdir -p "${target_dir}"
  chmod 700 "${target_dir}"
  cp "${src_cert}" "${target_dir}/fullchain.pem"
  cp "${src_key}"  "${target_dir}/privkey.pem"
  chmod 0644 "${target_dir}/fullchain.pem"
  chmod 0600 "${target_dir}/privkey.pem"

  local domain expire_at
  domain="$(openssl x509 -noout -subject -in "${target_dir}/fullchain.pem" 2>/dev/null \
            | sed -n 's/.*CN *= *//p' | sed 's#/$##' | head -n1)"
  domain="${domain:-${name}}"
  expire_at="$(openssl_expire_epoch "${target_dir}/fullchain.pem")"
  write_cert_meta "${target_dir}/meta.json" "${name}" "${domain}" \
    "${sources[${idx}]}" "${target_dir}/fullchain.pem" "${target_dir}/privkey.pem" \
    "$(date +%s)" "${expire_at}" "false" "imported" "0"
  printf '证书已导入: %s\n' "${target_dir}"
}

cmd_cert_not_implemented() {
  printf '即将支持 / Not implemented yet\n'
}

cmd_cert_manager() {
  ensure_certs_dir
  local subcmd="${1:-}"
  case "${subcmd}" in
    status) cmd_cert_status; return 0 ;;
    renew) cmd_cert_renew; return 0 ;;
    autorenew) cmd_cert_autorenew; return 0 ;;
    issue-domain) cmd_cert_issue_acme; return 0 ;;
    issue-ip) cmd_cert_issue_ip; return 0 ;;
    discover-import) cmd_cert_discover_import; return 0 ;;
  esac
  while true; do
    cat <<'EOF'
Certificate Manager
1. 查看证书
2. 导入证书
3. 删除证书
4. 设置面板 HTTPS 证书
5. 申请域名证书（Let's Encrypt）
6. 申请 IP 证书（实验性）
7. 查看续期状态
8. 立即续期
9. 自动续期开关
10. 发现并导入系统证书
11. 返回
EOF
    local choice
    read -r -p "Select: " choice
    case "${choice}" in
      1) cmd_cert_list ;;
      2) cmd_cert_import ;;
      3) cmd_cert_delete ;;
      4) cmd_cert_set_panel_https ;;
      5) cmd_cert_issue_acme ;;
      6) cmd_cert_issue_ip ;;
      7) cmd_cert_status ;;
      8) cmd_cert_renew ;;
      9) cmd_cert_autorenew ;;
      10|0) return 0 ;;
      *) printf 'Invalid selection\n' ;;
    esac
  done
}

cmd_port_guard_status() {
  printf '端口保护状态\n'
  if command -v systemctl >/dev/null 2>&1; then
    local timer_state service_state
    timer_state="$(systemctl is-active zui-port-guard-sync.timer 2>/dev/null || true)"
    service_state="$(systemctl is-active zui-port-guard-sync.service 2>/dev/null || true)"
    printf '定时器: %s\n' "$(port_guard_systemd_state_text "${timer_state:-unknown}")"
    printf '同步服务: %s\n' "$(port_guard_systemd_state_text "${service_state:-unknown}")"
  fi
  if command -v nft >/dev/null 2>&1; then
    nft list table inet "${PORT_GUARD_TABLE}" 2>/dev/null || printf 'nftables 表不存在: inet %s\n' "${PORT_GUARD_TABLE}"
  else
    printf '未找到 nft 命令，端口保护不可用\n'
  fi
}

cmd_port_guard_sync() {
  [[ -x "${PORT_GUARD_SYNC}" ]] || fail "端口保护同步脚本不存在: ${PORT_GUARD_SYNC}"
  printf '正在同步端口保护规则...\n'
  "${PORT_GUARD_SYNC}"
}

cmd_port_guard_unban() {
  local port="${1:-}"
  [[ "${port}" =~ ^[0-9]+$ ]] || fail "用法: x-ui port-guard unban <端口>"
  if command -v nft >/dev/null 2>&1; then
    nft delete element inet "${PORT_GUARD_TABLE}" blocked_ports "{ ${port} }" 2>/dev/null || true
    nft flush set inet "${PORT_GUARD_TABLE}" "pg4_${port}" 2>/dev/null || true
  fi
  if [[ -f "${DB_PATH}" ]]; then
    sqlite3 "${DB_PATH}" "update inbounds set port_guard_banned_until=0, port_guard_last_trigger_ip='', port_guard_last_trigger_at=0 where port=${port};" >/dev/null 2>&1 || true
  fi
  mkdir -p "$(dirname "${PORT_GUARD_LOG}")"
  python3 - "${port}" >>"${PORT_GUARD_LOG}" <<'PY'
import json, sys, time
print(json.dumps({
    "time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "event": "manual_unban",
    "message": "手动解除端口保护封禁",
    "port": int(sys.argv[1]),
    "operator": "x-ui.sh",
}, separators=(",", ":"), ensure_ascii=False))
PY
  printf '已解除端口 %s 的端口保护封禁\n' "${port}"
}

cmd_port_guard_logs() {
  if [[ -f "${PORT_GUARD_LOG}" ]]; then
    tail -n "${1:-100}" "${PORT_GUARD_LOG}"
  else
    printf '端口保护日志不存在: %s\n' "${PORT_GUARD_LOG}"
  fi
}

port_guard_systemd_state_text() {
  case "$1" in
    active) printf '运行中' ;;
    inactive) printf '未运行' ;;
    activating) printf '启动中' ;;
    failed) printf '失败' ;;
    unknown) printf '未知' ;;
    *) printf '%s' "$1" ;;
  esac
}

cmd_port_guard() {
  local subcmd="${1:-status}"
  case "${subcmd}" in
    status) cmd_port_guard_status ;;
    sync) cmd_port_guard_sync ;;
    unban) shift; cmd_port_guard_unban "$@" ;;
    logs) shift; cmd_port_guard_logs "$@" ;;
    *) fail "用法: x-ui port-guard {status|sync|unban <端口>|logs}" ;;
  esac
}

menu_service() {
  while true; do
    cat <<'EOF'
========================
服务管理
====

1. 启动服务
2. 停止服务
3. 重启服务
4. 查看状态
5. 查看日志
6. 开机启动
7. 取消开机启动
8. 返回
EOF
    local choice
    read -r -p "请输入选择：" choice || return 0
    case "${choice}" in
      "") continue ;;
      1) cmd_start; pause_return ;;
      2) cmd_stop; pause_return ;;
      3) cmd_restart; pause_return ;;
      4) cmd_status; pause_return ;;
      5) cmd_log ;;
      6) cmd_enable; pause_return ;;
      7) cmd_disable; pause_return ;;
      8|0) return 0 ;;
      *) printf '无效选择，请重新输入。\n' ;;
    esac
  done
}

menu_port_guard() {
  while true; do
    cat <<'EOF'
========================
端口保护
====

1. 查看状态
2. 立即同步
3. 查看日志
4. 手动解除封禁
5. 返回
EOF
    local choice port
    read -r -p "请输入选择：" choice || return 0
    case "${choice}" in
      "") continue ;;
      1) cmd_port_guard_status; pause_return ;;
      2) cmd_port_guard_sync; pause_return ;;
      3) cmd_port_guard_logs; pause_return ;;
      4)
        read -r -p "请输入要解除封禁的端口：" port || port=""
        if [[ -z "${port}" ]]; then
          printf '端口不能为空。\n'
        else
          cmd_port_guard_unban "${port}"
        fi
        pause_return
        ;;
      5|0) return 0 ;;
      *) printf '无效选择，请重新输入。\n' ;;
    esac
  done
}

menu_user_manager() {
  while true; do
    cat <<'EOF'
面板账号密码管理

1. 自定义重置账号密码
2. 随机重置账号密码
0. 返回主菜单

EOF
    local choice username password
    read -r -p "请输入选择：" choice || return 0
    case "${choice}" in
      "")
        continue
        ;;
      1)
        read -r -p "请输入新用户名：" username || username=""
        if [[ -z "${username}" ]]; then
          printf '用户名不能为空。\n'
          pause_return
          continue
        fi
        read -r -p "请输入新密码：" password || password=""
        if [[ -z "${password}" ]]; then
          printf '密码不能为空。\n'
          pause_return
          continue
        fi
        apply_panel_user "${username}" "${password}"
        pause_return
        ;;
      2)
        cmd_reset_user
        pause_return
        ;;
      0)
        return 0
        ;;
      *)
        printf '无效选择，请重新输入。\n'
        ;;
    esac
  done
}

menu_port_manager() {
  while true; do
    cat <<'EOF'
面板端口管理

1. 重置自定义端口
2. 重置随机端口（范围10000-59999）
0. 返回主菜单

EOF
    local choice port
    read -r -p "请输入选择：" choice || return 0
    case "${choice}" in
      "")
        continue
        ;;
      1)
        read -r -p "请输入新面板端口：" port || port=""
        if ! validate_panel_port "${port}"; then
          pause_return
          continue
        fi
        apply_panel_port "${port}"
        pause_return
        ;;
      2)
        cmd_reset_port
        pause_return
        ;;
      0)
        return 0
        ;;
      *)
        printf '无效选择，请重新输入。\n'
        ;;
    esac
  done
}

menu_update() {
  cat <<'EOF'
警告：
更新功能可能覆盖当前安装内容。

确认继续？
请输入 YES 继续：
EOF
  local confirm
  read -r confirm || confirm=""
  if [[ "${confirm}" == "YES" ]]; then
    cmd_update
  else
    printf '已取消更新。\n'
    pause_return
  fi
}

menu_uninstall() {
  cat <<'EOF'
警告：
将卸载 z-ui 服务和程序文件。

数据库与证书默认保留。

请输入：
UNINSTALL
继续：
EOF
  local confirm
  read -r confirm || confirm=""
  if [[ "${confirm}" == "UNINSTALL" ]]; then
    cmd_uninstall_confirmed
  else
    printf '已取消卸载。\n'
    pause_return
  fi
}

print_menu_separator() {
  printf '──────────────────────\n'
}

print_menu_header() {
  print_menu_separator
  printf 'z-ui 管理面板\n'
  print_menu_separator
}

print_menu_status() {
  local status panel_status xray_status port
  status="$(service_status)"
  panel_status="$(service_status_text "${status}")"
  xray_status="$(service_status_text "${status}")"
  port="$(current_port)"
  print_menu_separator
  printf '系统状态\n\n'
  printf '面板状态 : %s\n' "${panel_status}"
  printf 'Xray状态 : %s\n' "${xray_status}"
  printf '端口     : %s\n' "${port}"
  print_menu_separator
}

print_main_menu() {
  print_menu_header
  cat <<'EOF'

──────── 服务相关 ────────
1. 服务管理

──────── 面板相关 ────────
2. 面板信息
3. 面板账号密码管理
4. 面板端口管理

──────── 系统功能 ────────
5. 证书管理
6. 端口保护
7. 更新系统
8. 卸载系统

──────── 退出 ────────
9. 退出

EOF
  print_menu_status
}

menu_main() {
  trap 'printf "\n已退出菜单。\n"; exit 130' INT
  while true; do
    print_main_menu
    local choice
    read -r -p "请输入选择：" choice || return 0
    case "${choice}" in
      "") continue ;;
      1) menu_service ;;
      2) cmd_panel_info_cn; pause_return ;;
      3) menu_user_manager ;;
      4) menu_port_manager ;;
      5) cmd_cert_manager ;;
      6) menu_port_guard ;;
      7) menu_update ;;
      8) menu_uninstall ;;
      9|0|q|Q) return 0 ;;
      *) printf '无效选择，请重新输入。\n' ;;
    esac
  done
}

cmd_update() {
  require_root
  local arch
  case "$(uname -m)" in
    x86_64)  arch="amd64" ;;
    aarch64) arch="arm64" ;;
    *)       fail "unsupported architecture: $(uname -m)" ;;
  esac

  # Fetch target version from GitHub install.sh
  local raw_url="https://raw.githubusercontent.com/${REPO}/${BRANCH}/install.sh"
  local target_version
  target_version="$(curl -fsSL "${raw_url}" | grep '^XUI_RELEASE_VERSION=' | head -1 | cut -d'"' -f2)"
  [[ -n "${target_version}" ]] || fail "failed to determine target version from ${raw_url}"

  local current_version
  current_version="$("${INSTALL_DIR}/x-ui" version 2>/dev/null | sed -n 's/^Version: //p' | head -1 || true)"
  printf '当前版本：%s\n目标版本：%s\n' "${current_version:-unknown}" "${target_version}"

  # Download release package to temp dir
  local tmp_dir
  tmp_dir="$(mktemp -d /tmp/z-ui-update-XXXXXX)"
  # No RETURN trap: bash may read the replaced script file when trap fires.
  # Clean up tmp_dir explicitly before systemctl (before the file is replaced).

  local pkg="${tmp_dir}/x-ui.tar.gz"
  local url="https://github.com/${REPO}/releases/download/${target_version}/z-ui-linux-${arch}.tar.gz"
  printf '下载 %s ...\n' "${url}"
  curl -fL --retry 3 --connect-timeout 15 -o "${pkg}" "${url}"
  [[ -s "${pkg}" ]] || fail "downloaded package is empty"
  tar -tzf "${pkg}" >/dev/null

  # Extract and validate
  local extract_dir="${tmp_dir}/extract"
  mkdir -p "${extract_dir}"
  tar -xzf "${pkg}" -C "${extract_dir}"
  local source_dir="${extract_dir}/x-ui"
  [[ -d "${source_dir}" ]] || source_dir="${extract_dir}"
  [[ -f "${source_dir}/x-ui" ]] || fail "package missing x-ui binary"
  [[ -f "${source_dir}/x-ui.sh" ]] || fail "package missing x-ui.sh"

  # Backup current install dir
  local backup_dir="${INSTALL_DIR}.backup.$(date -u +%Y%m%d%H%M%S)"
  cp -a "${INSTALL_DIR}" "${backup_dir}"

  # Stop service before replacing files
  systemctl stop "${SERVICE_NAME}" >/dev/null 2>&1 || true

  # Replace program files — never touch /etc/x-ui
  rm -rf "${INSTALL_DIR}"
  mkdir -p "${INSTALL_DIR}"
  cp -a "${source_dir}/." "${INSTALL_DIR}/"
  chmod 755 "${INSTALL_DIR}/x-ui" "${INSTALL_DIR}/x-ui.sh"
  [[ -f "${INSTALL_DIR}/bin/xray-linux-${arch}" ]] && chmod 755 "${INSTALL_DIR}/bin/xray-linux-${arch}"

  install -m 0755 "${INSTALL_DIR}/x-ui.sh" /usr/bin/x-ui
  install -m 0644 "${INSTALL_DIR}/x-ui.service" "/etc/systemd/system/${SERVICE_NAME}.service"
  [[ -f "${INSTALL_DIR}/scripts/zui-port-guard-sync" ]] &&     install -m 0755 "${INSTALL_DIR}/scripts/zui-port-guard-sync" "${PORT_GUARD_SYNC}"
  [[ -f "${INSTALL_DIR}/zui-port-guard-sync.service" ]] &&     install -m 0644 "${INSTALL_DIR}/zui-port-guard-sync.service" "${PORT_GUARD_SERVICE}"
  [[ -f "${INSTALL_DIR}/zui-port-guard-sync.timer" ]] &&     install -m 0644 "${INSTALL_DIR}/zui-port-guard-sync.timer" "${PORT_GUARD_TIMER}"
  local lr_src="${INSTALL_DIR}/packaging/logrotate/x-ui-xray-access"
  [[ -f "${lr_src}" ]] && install -m 0644 "${lr_src}" /etc/logrotate.d/x-ui-xray-access

  rm -rf "${tmp_dir}"

  systemctl daemon-reload
  systemctl enable "${SERVICE_NAME}" >/dev/null 2>&1 || true
  systemctl start "${SERVICE_NAME}"

  # Verify or rollback
  if systemctl is-active --quiet "${SERVICE_NAME}"; then
    local new_version
    new_version="$("${INSTALL_DIR}/x-ui" version 2>/dev/null | sed -n 's/^Version: //p' | head -1 || true)"
    printf 'z-ui 更新成功：%s → %s\n备份保留在：%s\n' "${current_version:-unknown}" "${new_version:-unknown}" "${backup_dir}"
  else
    printf 'ERROR: 服务启动失败，正在回滚...\n' >&2
    rm -rf "${INSTALL_DIR}"
    mv "${backup_dir}" "${INSTALL_DIR}"
    install -m 0755 "${INSTALL_DIR}/x-ui.sh" /usr/bin/x-ui
    systemctl daemon-reload
    systemctl start "${SERVICE_NAME}" || true
    fail "更新失败，已回滚到旧版本"
  fi
}

cmd_uninstall_confirmed() {
  require_root
  systemctl stop "${SERVICE_NAME}" >/dev/null 2>&1 || true
  systemctl disable "${SERVICE_NAME}" >/dev/null 2>&1 || true
  systemctl stop zui-port-guard-sync.timer >/dev/null 2>&1 || true
  systemctl disable zui-port-guard-sync.timer >/dev/null 2>&1 || true
  systemctl stop zui-port-guard-sync.service >/dev/null 2>&1 || true
  rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
  rm -f "${PORT_GUARD_TIMER}" "${PORT_GUARD_SERVICE}" "${PORT_GUARD_SYNC}"
  rm -f /etc/logrotate.d/x-ui-xray-access
  systemctl daemon-reload
  if command -v nft >/dev/null 2>&1; then
    nft delete table inet "${PORT_GUARD_TABLE}" >/dev/null 2>&1 || true
  fi
  rm -rf /var/lib/z-ui/port-guard
  rm -rf "${INSTALL_DIR}"
  rm -f "/usr/bin/x-ui"
  printf 'z-ui 已卸载。数据库保留在：%s\n' "${DB_PATH}"
}

cmd_uninstall() {
  require_root
  local confirm
  read -r -p "请输入 UNINSTALL 确认卸载 z-ui：" confirm || confirm=""
  if [[ "${confirm}" != "UNINSTALL" ]]; then
    printf '已取消卸载。\n'
    return 0
  fi
  cmd_uninstall_confirmed
}

usage() {
  cat <<'EOF'
x-ui 命令用法：
  x-ui              进入中文交互菜单
  x-ui start        启动服务
  x-ui stop         停止服务
  x-ui restart      重启服务
  x-ui status       查看状态
  x-ui enable       开机启动
  x-ui disable      取消开机启动
  x-ui log          查看日志
  x-ui reset-user   随机重置账号密码
  x-ui reset-port   随机重置面板端口（范围10000-59999）
  x-ui set-user <用户名> <密码>
                    自定义重置账号密码
  x-ui set-port <端口>
                    自定义重置面板端口
  x-ui info         查看面板地址、用户名和当前密码
  x-ui cert         证书管理
  x-ui cert status  查看证书续期状态
  x-ui cert renew   续期 30 天内到期的证书
  x-ui cert autorenew
                    切换证书自动续期
  x-ui port-guard status
                    显示端口保护 nftables 状态
  x-ui port-guard sync
                    立即同步端口保护规则
  x-ui port-guard unban <port>
                    解除指定端口的端口保护封禁
  x-ui port-guard logs
                    显示端口保护日志
  x-ui update       更新系统
  x-ui uninstall    卸载服务和程序文件，保留数据库
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
    set-user) shift; cmd_set_user "$@" ;;
    set-port) shift; cmd_set_port "$@" ;;
    info) cmd_info ;;
    cert) shift; cmd_cert_manager "$@" ;;
    port-guard) shift; cmd_port_guard "$@" ;;
    update) cmd_update ;;
    uninstall) cmd_uninstall ;;
    "") menu_main ;;
    help | -h | --help) usage ;;
    *) usage; fail "unknown command: ${cmd}" ;;
  esac
}

main "$@"
