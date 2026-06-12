#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ "${EUID}" -ne 0 ]]; then
  echo "install smoke must run as root" >&2
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64)
    ARCH="amd64"
    ;;
  aarch64 | arm64)
    ARCH="arm64"
    ;;
  *)
    echo "unsupported arch: $ARCH" >&2
    exit 1
    ;;
esac

TMP_DIR="$(mktemp -d)"
HTTP_PID=""
BACKUP_DIR=""

cleanup() {
  if [[ -n "$HTTP_PID" ]]; then
    kill "$HTTP_PID" >/dev/null 2>&1 || true
    wait "$HTTP_PID" >/dev/null 2>&1 || true
  fi
  timeout 10 systemctl stop x-ui >/dev/null 2>&1 || true
  timeout 10 systemctl disable x-ui >/dev/null 2>&1 || true
  rm -rf /usr/local/x-ui
  rm -f /usr/bin/x-ui
  rm -f /etc/systemd/system/x-ui.service
  systemctl daemon-reload >/dev/null 2>&1 || true
  if [[ -n "$BACKUP_DIR" ]]; then
    rm -rf /etc/x-ui
    if [[ -d "$BACKUP_DIR" ]]; then
      mv "$BACKUP_DIR" /etc/x-ui
    fi
  fi
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

go build -o "$TMP_DIR/x-ui-bin" .

PKG_DIR="$TMP_DIR/pkg/x-ui"
mkdir -p "$PKG_DIR/bin"
cp "$TMP_DIR/x-ui-bin" "$PKG_DIR/x-ui"
cp install.sh "$PKG_DIR/install.sh"
cp x-ui.sh "$PKG_DIR/x-ui.sh"
cp x-ui.service "$PKG_DIR/x-ui.service"
cp "bin/xray-linux-${ARCH}" "$PKG_DIR/bin/"
cp bin/geoip.dat "$PKG_DIR/bin/"
cp bin/geosite.dat "$PKG_DIR/bin/"
chmod 755 "$PKG_DIR/x-ui" "$PKG_DIR/x-ui.sh" "$PKG_DIR/bin/xray-linux-${ARCH}"

tar -czf "$TMP_DIR/x-ui-linux-${ARCH}.tar.gz" -C "$TMP_DIR/pkg" x-ui

python3 -m http.server 18080 --directory "$TMP_DIR" >/tmp/x-ui-install-smoke-http.log 2>&1 &
HTTP_PID=$!

if [[ -d /etc/x-ui ]]; then
  BACKUP_DIR="$(mktemp -d /tmp/x-ui-install-smoke-backup.XXXXXX)"
  rm -rf "$BACKUP_DIR"
  mv /etc/x-ui "$BACKUP_DIR"
fi

timeout 10 systemctl stop x-ui >/dev/null 2>&1 || true
timeout 10 systemctl disable x-ui >/dev/null 2>&1 || true
rm -rf /usr/local/x-ui
rm -f /usr/bin/x-ui
rm -f /etc/systemd/system/x-ui.service
systemctl daemon-reload

XUI_INSTALL_PACKAGE_URL="http://127.0.0.1:18080/x-ui-linux-${ARCH}.tar.gz" \
XUI_SKIP_DEP_INSTALL=1 \
  bash install.sh | tee /tmp/x-ui-install-smoke-output.txt

PORT="$(awk '/^Panel URL:/{getline; sub(/^.*:/, "", $0); print; exit}' /tmp/x-ui-install-smoke-output.txt)"
USERNAME="$(awk '/^Username:/{getline; print; exit}' /tmp/x-ui-install-smoke-output.txt)"
PASSWORD="$(awk '/^Password:/{getline; print; exit}' /tmp/x-ui-install-smoke-output.txt)"

[[ "$PORT" =~ ^[0-9]+$ ]] || { echo "invalid port: $PORT" >&2; exit 1; }
[[ "$PORT" -ge 10000 && "$PORT" -le 59999 ]] || { echo "port out of range: $PORT" >&2; exit 1; }
[[ "$PORT" -ne 54321 ]] || { echo "port must not be fixed default 54321" >&2; exit 1; }
[[ "$USERNAME" =~ ^xui_[0-9a-f]{6,8}$ ]] || { echo "invalid username: $USERNAME" >&2; exit 1; }
[[ "$USERNAME" != "admin" ]] || { echo "username must not be admin" >&2; exit 1; }
[[ "$PASSWORD" != "admin" ]] || { echo "password must not be admin" >&2; exit 1; }
[[ "${#PASSWORD}" -ge 24 ]] || { echo "password too short" >&2; exit 1; }

systemctl is-active --quiet x-ui
curl -fsS "http://127.0.0.1:${PORT}/" >/tmp/x-ui-install-smoke-index.html

COOKIE="$TMP_DIR/cookie"
curl -fsS -c "$COOKIE" \
  --data-urlencode "username=${USERNAME}" \
  --data-urlencode "password=${PASSWORD}" \
  "http://127.0.0.1:${PORT}/login" >/tmp/x-ui-install-smoke-login.json
python3 - <<'PY'
import json
with open("/tmp/x-ui-install-smoke-login.json", "r", encoding="utf-8") as handle:
    data = json.load(handle)
if not data.get("success"):
    raise SystemExit(f"login failed: {data.get('msg', '')}")
PY

curl -fsS -b "$COOKIE" -X POST "http://127.0.0.1:${PORT}/server/status" >/tmp/x-ui-install-smoke-server-status.json
python3 - <<'PY'
import json
with open("/tmp/x-ui-install-smoke-server-status.json", "r", encoding="utf-8") as handle:
    data = json.load(handle)
state = (((data.get("obj") or {}).get("xray") or {}).get("state"))
if state != "running":
    raise SystemExit(f"xray status is not running: {state}")
PY

test "$(stat -c %a /etc/x-ui)" = "700"
test "$(stat -c %a /etc/x-ui/x-ui.db)" = "600"
test "$(stat -c %a /usr/local/x-ui/bin/config.json)" = "600"

echo "install smoke passed"
