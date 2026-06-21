# Latest Project State

## Project

- Project name: `x-ui`
- Current version: `0.3.2`
- Current branch: `z-ui`
- Latest audited phase: `006_PORT_GUARD`

## Recent Commits

```text
ffee765 Localize Port Guard user-facing text
5571ccb Add Port Guard UI controls and status display
b5b1c46 Add Port Guard nftables sync script and systemd timer
9338a83 Add Port Guard database fields and API
2f3ca3a Fix release version handling in installer
da3b1b3 Use z-ui release asset names in installer
2038cb6 Prepare z-ui branch with self-hosted source links
339ed7f Prepare repository for remote upload with knowledge base
c73661d Harden x-ui command safety and certificate handling
8c12a8a Add certificate lifecycle management
```

## Current Feature Tree

Core areas:

- `protocol/`: protocol parsing and share-link compatibility
- `runtime/`: Xray runtime coordination
- `database/`: schema, models, explicit migrations
- `web/controller/`: HTTP controllers and API routes
- `web/service/`: application service layer
- `web/html/`: panel templates
- `web/assets/`: frontend model and static assets
- `scripts/`: smoke checks and operational scripts
- `install.sh`: installer
- `x-ui.sh`: management command

Implemented modern features:

- database migrations
- ProtocolModule cleanup
- certificate manager
- certificate lifecycle management
- auto renew
- HTTP+TLS
- REALITY / Vision / XHTTP compatibility
- asset cache buster
- repository self-hosted install links
- Port Guard MVP

## Protocol Status

Supported protocols:

- VMess
- VLESS
- Trojan
- Shadowsocks
- Dokodemo-door
- SOCKS
- HTTP
- mixed
- tunnel

Port Guard is protocol-agnostic because it protects inbound ports through nftables instead of protocol-specific runtime changes.

## Certificate Status

Certificate system status:

- certificate manager present
- certificate scanning present
- panel HTTPS support present
- ACME flow present
- auto-renew systemd timer present
- IP certificate experimental entry preserved

## Port Guard Status

Port Guard MVP status:

- implemented
- IPv4-only
- nftables table: `inet zui_port_guard`
- sync script: `/usr/local/bin/zui-port-guard-sync`
- log path: `/var/log/z-ui/port-guard.log`
- state dir: `/var/lib/z-ui/port-guard`
- timer: `zui-port-guard-sync.timer`
- service: `zui-port-guard-sync.service`

Port Guard does:

- window-period unique IPv4 source IP counting
- whole-port temporary ban
- automatic timeout recovery through nftables
- manual unban through API and `x-ui.sh`
- Chinese user-facing status text

Port Guard does not do:

- IPv6
- port speed limit
- tc shaping
- nftables speedlimit
- iptables shim
- Runtime RemoveInbound
- per-client limit
- device limit

## Current Validation Baseline

Latest Port Guard validation passed:

- `go build ./...`
- `go test ./...`
- `bash -n install.sh`
- `bash -n x-ui.sh`
- `bash -n scripts/zui-port-guard-sync`
- Playwright DOM check
- remote deployment check on `45.77.246.87`
- remote `x-ui port-guard status`
- remote `/api/port-guard/status`

## Future History Rule

Next major phase must create:

- `PROJECT_HISTORY/007_<FEATURE>.md`

Do not overwrite:

- `PROJECT_HISTORY/006_PORT_GUARD.md`

Update only:

- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`
