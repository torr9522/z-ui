# Modern x-ui

Modern x-ui is a maintained modernization branch of x-ui. It keeps the original lightweight Go + SQLite + server-rendered panel architecture while upgrading the project to Xray-core 26.x and adding safer authentication, migrations, protocol modules, Runtime API reconciliation, certificate management, and secure install scripts.

## Features

- Go 1.26 baseline
- Xray-core 26.6.1 baseline
- SQLite storage
- `schema_migrations`
- `password_hash` migration
- `inbound_clients` migration
- ProtocolModule-driven inbound config generation
- Runtime API sync with full restart fallback
- CSRF and hardened session cookies
- Random first-run username, password, and panel port
- Secure installer and `x-ui` management command
- Certificate manager, ACME domain certificates, auto renewal
- TLS certificate selector
- HTTP + TLS mode
- REALITY / Vision / XHTTP / mixed support

## Supported Protocols

- VMess
- VLESS
- Trojan
- Shadowsocks
- Dokodemo-door
- SOCKS
- HTTP
- mixed
- tunnel

TUIC is not implemented unless future Xray-core source code confirms native inbound support.

## Installation

The installer supports Debian, Ubuntu, CentOS, Rocky, and Alma on amd64/arm64.

```bash
bash <(curl -Ls https://raw.githubusercontent.com/torr9522/z-ui/z-ui/install.sh)
```

The installer generates a random username, strong random password, and random panel port in the 10000-59999 range. Credentials are printed only to the current terminal.

## Management Commands

Running `x-ui` opens the Chinese interactive menu:

```bash
x-ui
```

The menu provides service management, panel information, credential reset, port reset, certificate management, Port Guard, update, and uninstall entries.

Advanced users and automation can still use command mode:

```bash
x-ui start
x-ui stop
x-ui restart
x-ui status
x-ui enable
x-ui disable
x-ui log
x-ui reset-user
x-ui reset-port
x-ui info
x-ui update
x-ui uninstall
```

Certificate commands:

```bash
x-ui cert
x-ui cert status
x-ui cert renew
x-ui cert autorenew
```

Port Guard commands:

```bash
x-ui port-guard status
x-ui port-guard sync
x-ui port-guard unban <port>
x-ui port-guard logs
```

`x-ui uninstall` requires typing `UNINSTALL` before it proceeds.

## Certificate Management

Certificate layout:

```text
/etc/x-ui/certs/<name>/
  fullchain.pem
  privkey.pem
  meta.json
```

Supported operations:

- list certificates
- import certificate
- delete certificate with backup
- set panel HTTPS certificate
- issue domain certificate with acme.sh standalone HTTP-01
- view renewal status
- renew now
- toggle auto renewal

API:

```http
GET /api/certificates
```

The API returns metadata and paths only. It never returns private key contents.

## HTTP + TLS

The project does not add a fake `https` protocol. HTTPS inbound mode is represented as:

```text
protocol = http
streamSettings.security = tls
```

HTTP TLS mode requires valid `certificateFile` and `keyFile` paths.

## REALITY

REALITY is a separate security mode and does not use TLS certificates. Certificate management must not be mixed into REALITY settings.

## Development

Recommended tools:

- Go 1.26+
- Node.js for Playwright DOM checks
- systemd environment for install/service validation

Build:

```bash
go mod download
go build ./...
go build -o x-ui .
```

Test:

```bash
go test ./...
bash -n install.sh
bash -n x-ui.sh
scripts/rc_smoke.sh
scripts/rc_upgrade_smoke.sh
node scripts/share_link_validate.js
```

DOM checks require a running panel:

```bash
XUI_BASE_URL=http://127.0.0.1:PORT \
XUI_USERNAME=USERNAME \
XUI_PASSWORD=PASSWORD \
node scripts/ui_dom_check.js
```

## Directory Guide

- `protocol/`: protocol modules, validation, migration, FormSchema, config generation
- `runtime/`: Runtime API, planner, builder, reconciler, fallback
- `database/`: SQLite initialization and migrations
- `web/`: Gin server, controllers, services, templates, frontend assets
- `scripts/`: smoke tests, upgrade tests, DOM checks, share-link validation
- `xray/`: Xray process/config wrapper
- `install.sh`: unattended installer
- `x-ui.sh`: management command script
- `knowledge-base/`: project knowledge base and secondary development guide

## Security Notes

Do not upload local secrets or generated artifacts:

- passwords, tokens, cookies, sessions
- SSH keys, private keys, certificate private keys
- real database files
- `node_modules/`
- `backup/`, `archive/`, `combined-backup/`
- `*.tar.gz`, `*.sha256`

The repository `.gitignore` excludes these local files by default.

## Remote Upload Checklist

```bash
git status --short
git ls-files | grep -E 'node_modules|backup|archive|combined-backup|\.tar\.gz|\.sha256|\.pem|\.key|\.crt|\.db' || true
go build ./...
go test ./...
bash -n install.sh
bash -n x-ui.sh
```
