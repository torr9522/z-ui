#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

go test ./...
go build -o /tmp/x-ui-rc .

DB_PATH="/etc/x-ui/x-ui.db"
ETC_DIR="/etc/x-ui"
BACKUP_DIR=""
HAD_ETC_DIR=0
WEB_PORT="54321"
BASE_URL=""

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
  BACKUP_DIR="$(mktemp -d /tmp/x-ui-rc-backup.XXXXXX)"
  rm -rf "$BACKUP_DIR"
  mv "$ETC_DIR" "$BACKUP_DIR"
fi
mkdir -p "$ETC_DIR"
chmod 700 "$ETC_DIR"
rm -f "$DB_PATH"
touch "$DB_PATH"
chmod 600 "$DB_PATH"
rm -f "$DB_PATH"

/tmp/x-ui-rc run >/tmp/x-ui-rc.log 2>&1 &
PANEL_PID=$!

for _ in $(seq 1 30); do
  WEB_PORT="$(sed -n 's/^  Panel port: //p' /tmp/x-ui-rc.log | tail -n1)"
  if [[ -z "$WEB_PORT" && -f "$DB_PATH" ]]; then
    WEB_PORT="$(python3 - <<'PY'
import sqlite3
try:
    conn = sqlite3.connect("/etc/x-ui/x-ui.db")
    cur = conn.cursor()
    cur.execute("select value from settings where key='webPort'")
    row = cur.fetchone()
    print(row[0] if row else "")
except Exception:
    print("")
PY
)"
  fi
  if [[ -n "$WEB_PORT" ]]; then
    BASE_URL="http://127.0.0.1:${WEB_PORT}"
    break
  fi
  sleep 1
done

if [[ -z "$BASE_URL" ]]; then
  echo "unable to determine panel port for rc smoke" >&2
  exit 1
fi

for _ in $(seq 1 30); do
  if curl -fsS "${BASE_URL}/" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

curl -fsS "${BASE_URL}/" >/dev/null

COOKIE=/tmp/x-ui-rc.cookie
rm -f "$COOKIE"
USERNAME="${XUI_SMOKE_USERNAME:-}"
PASSWORD="${XUI_SMOKE_PASSWORD:-}"
if [[ -z "$USERNAME" || -z "$PASSWORD" ]]; then
  USERNAME="$(sed -n 's/^  Username: //p' /tmp/x-ui-rc.log | tail -n1)"
  PASSWORD="$(sed -n 's/^  Password: //p' /tmp/x-ui-rc.log | tail -n1)"
fi
if [[ -z "$USERNAME" || -z "$PASSWORD" ]]; then
  echo "unable to determine login credentials for rc smoke" >&2
  exit 1
fi

curl -fsS -c "$COOKIE" \
  --data-urlencode "username=${USERNAME}" \
  --data-urlencode "password=${PASSWORD}" \
  "${BASE_URL}/login" >/tmp/x-ui-rc-login.json
python3 - <<'PY'
import json
with open("/tmp/x-ui-rc-login.json", "r", encoding="utf-8") as handle:
    obj = json.load(handle)
if not obj.get("success"):
    raise SystemExit(f"login failed: {obj.get('msg', '')}")
PY
curl -fsS -b "$COOKIE" "${BASE_URL}/xui/inbounds" >/tmp/x-ui-rc-inbounds.html
curl -fsS -b "$COOKIE" "${BASE_URL}/api/protocol/schema/vless" >/tmp/x-ui-rc-schema-vless.json
curl -fsS -b "$COOKIE" "${BASE_URL}/api/protocol/schema/mixed" >/tmp/x-ui-rc-schema-mixed.json

./bin/xray-linux-amd64 -test -c bin/config.json >/tmp/x-ui-rc-xray-test.log

echo "rc smoke passed"
