#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

go test ./...
go build -o /tmp/x-ui-rc .

ETC_DIR="/etc/x-ui"
DB_PATH="${ETC_DIR}/x-ui.db"
BASE_URL="http://127.0.0.1:54321"
COOKIE=/tmp/x-ui-rc-upgrade.cookie
BACKUP_DIR=""
HAD_ETC_DIR=0

cleanup() {
  if [[ -n "${PANEL_PID:-}" ]]; then
    kill "$PANEL_PID" >/dev/null 2>&1 || true
    wait "$PANEL_PID" >/dev/null 2>&1 || true
  fi
  if [[ -n "$BACKUP_DIR" ]]; then
    rm -rf "$ETC_DIR"
    if [[ -d "$BACKUP_DIR" ]]; then
      mv "$BACKUP_DIR" "$ETC_DIR"
    fi
  elif [[ "$HAD_ETC_DIR" -eq 0 ]]; then
    rm -rf "$ETC_DIR"
  fi
}
trap cleanup EXIT

if [[ -d "$ETC_DIR" ]]; then
  HAD_ETC_DIR=1
  BACKUP_DIR="$(mktemp -d /tmp/x-ui-rc-upgrade-backup.XXXXXX)"
  rm -rf "$BACKUP_DIR"
  mv "$ETC_DIR" "$BACKUP_DIR"
fi
mkdir -p "$ETC_DIR"
chmod 700 "$ETC_DIR"
rm -f "$DB_PATH"
touch "$DB_PATH"
chmod 600 "$DB_PATH"
rm -f "$DB_PATH"

sqlite3 "$DB_PATH" <<'SQL'
CREATE TABLE users (
  id integer primary key autoincrement,
  username text,
  password text
);
INSERT INTO users (username, password) VALUES ('legacy-admin', 'legacy-pass');

CREATE TABLE settings (
  id integer primary key autoincrement,
  key text,
  value text
);
INSERT INTO settings (key, value) VALUES ('webPort', '54321');

CREATE TABLE inbounds (
  id integer primary key autoincrement,
  user_id integer,
  up integer,
  down integer,
  total integer,
  remark text,
  enable numeric,
  expiry_time integer,
  listen text,
  port integer,
  protocol text,
  settings text,
  stream_settings text,
  tag text,
  sniffing text
);
INSERT INTO inbounds (
  user_id, up, down, total, remark, enable, expiry_time, listen, port, protocol, settings, stream_settings, tag, sniffing
) VALUES (
  1, 0, 0, 0, 'legacy-vless', 1, 0, '', 24443, 'vless',
  '{"clients":[{"id":"11111111-1111-1111-1111-111111111111","flow":"xtls-rprx-vision"}]}',
  '{"network":"ws","security":"tls","tlsSettings":{"serverName":"legacy.example.com"},"wsSettings":{"path":"/legacy","headers":{"Host":"legacy.example.com"}}}',
  'legacy-vless', '{"enabled":true,"destOverride":["http","tls"]}'
);
SQL

/tmp/x-ui-rc run >/tmp/x-ui-rc-upgrade.log 2>&1 &
PANEL_PID=$!

for _ in $(seq 1 30); do
  if curl -fsS "${BASE_URL}/" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

curl -fsS "${BASE_URL}/" >/dev/null
rm -f "$COOKIE"
curl -fsS -c "$COOKIE" \
  --data-urlencode "username=legacy-admin" \
  --data-urlencode "password=legacy-pass" \
  "${BASE_URL}/login" >/tmp/x-ui-rc-upgrade-login.json
python3 - <<'PY'
import json
with open("/tmp/x-ui-rc-upgrade-login.json", "r", encoding="utf-8") as handle:
    obj = json.load(handle)
if not obj.get("success"):
    raise SystemExit(f"login failed: {obj.get('msg', '')}")
PY

sqlite3 "$DB_PATH" "select password_hash from users where username='legacy-admin'" | grep -q .
test "$(sqlite3 "$DB_PATH" "select ifnull(password,'') from users where username='legacy-admin'")" = ""
sqlite3 "$DB_PATH" "select settings from inbounds where tag='legacy-vless'" | grep -q '"decryption":"none"'
sqlite3 "$DB_PATH" "select count(*) from inbound_clients where inbound_id=1" | grep -q '^1$'

curl -fsS -b "$COOKIE" "${BASE_URL}/xui/inbounds" >/tmp/x-ui-rc-upgrade-inbounds.html
./bin/xray-linux-amd64 -test -c bin/config.json >/tmp/x-ui-rc-upgrade-xray-test.log

echo "rc upgrade smoke passed"
