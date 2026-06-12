#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

go test ./...
go build -o /tmp/x-ui-rc .

DB_PATH="/etc/x-ui/x-ui.db"
WEB_PORT="54321"
if [[ -f "$DB_PATH" ]]; then
  DB_PORT="$(sqlite3 "$DB_PATH" "select value from settings where key='webPort' limit 1;" 2>/dev/null || true)"
  if [[ -n "$DB_PORT" ]]; then
    WEB_PORT="$DB_PORT"
  fi
fi
BASE_URL="http://127.0.0.1:${WEB_PORT}"

cleanup() {
  if [[ -n "${PANEL_PID:-}" ]]; then
    kill "$PANEL_PID" >/dev/null 2>&1 || true
    wait "$PANEL_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

/tmp/x-ui-rc run >/tmp/x-ui-rc.log 2>&1 &
PANEL_PID=$!

for _ in $(seq 1 30); do
  if curl -fsS "${BASE_URL}/" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

curl -fsS "${BASE_URL}/" >/dev/null

COOKIE=/tmp/x-ui-rc.cookie
rm -f "$COOKIE"
curl -fsS -c "$COOKIE" -d 'username=admin&password=admin' "${BASE_URL}/login" >/tmp/x-ui-rc-login.json
curl -fsS -b "$COOKIE" "${BASE_URL}/xui/inbounds" >/tmp/x-ui-rc-inbounds.html
curl -fsS -b "$COOKIE" "${BASE_URL}/api/protocol/schema/vless" >/tmp/x-ui-rc-schema-vless.json
curl -fsS -b "$COOKIE" "${BASE_URL}/api/protocol/schema/mixed" >/tmp/x-ui-rc-schema-mixed.json

./bin/xray-linux-amd64 -test -c bin/config.json >/tmp/x-ui-rc-xray-test.log

echo "rc smoke passed"
