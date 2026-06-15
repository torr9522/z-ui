#!/usr/bin/env bash
set -euo pipefail

APP_NAME="x-ui"
INSTALL_DIR="/usr/local/x-ui"
BIN="${INSTALL_DIR}/x-ui"
CONFIG_DIR="/etc/x-ui"
DB_PATH="${CONFIG_DIR}/x-ui.db"
CERTS_DIR="${CONFIG_DIR}/certs"
SERVICE_NAME="x-ui"
REPO="${XUI_REPO:-FranzKafkaYu/x-ui}"
DEFAULT_ACME_DOMAIN="cshtps.527270.xyz"
RENEW_SERVICE="/etc/systemd/system/x-ui-cert-renew.service"
RENEW_TIMER="/etc/systemd/system/x-ui-cert-renew.timer"

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

ensure_certs_dir() {
  mkdir -p "${CERTS_DIR}"
  chmod 700 "${CERTS_DIR}"
}

cert_dir() {
  local name="$1"
  printf '%s/%s\n' "${CERTS_DIR}" "${name}"
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
  find "${CERTS_DIR}" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort
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
  local name state meta
  name="$(choose_certificate)" || return 0
  read -r -p "Auto renew for ${name} [on/off]: " state
  case "${state}" in
    on|ON|On|1|true|TRUE) state="true" ;;
    off|OFF|Off|0|false|FALSE) state="false" ;;
    *) fail "invalid auto renew state: ${state}" ;;
  esac
  meta="$(cert_meta_file "${name}")"
  [[ -f "${meta}" ]] || fail "meta.json not found for ${name}"
  cert_meta_set "${meta}" autoRenew "${state}"
  if [[ "${state}" == "true" ]]; then
    install_cert_renew_timer
  fi
  printf 'autoRenew for %s set to %s\n' "${name}" "${state}"
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
10. 返回
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
  x-ui cert         Certificate manager
  x-ui cert status  Show certificate renewal status
  x-ui cert renew   Renew certificates expiring within 30 days
  x-ui cert autorenew
                    Toggle certificate auto renewal
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
    cert) shift; cmd_cert_manager "$@" ;;
    update) cmd_update ;;
    uninstall) cmd_uninstall ;;
    "" | help | -h | --help) usage ;;
    *) usage; fail "unknown command: ${cmd}" ;;
  esac
}

main "$@"
