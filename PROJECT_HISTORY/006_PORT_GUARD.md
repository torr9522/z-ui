# 006 Port Guard MVP

## Phase Summary

Port Guard is the first audited phase recorded in the Project History System.

The feature adds IPv4-only window-period unique source IP protection for inbound ports. It temporarily bans an entire inbound port when the number of unique IPv4 source IPs seen within the configured time window exceeds the configured limit.

Example:

- port: `12568`
- window: `300` seconds
- unique IPv4 limit: `3`
- ban time: `300` seconds

Behavior:

1. nftables tracks unique IPv4 source IPs for the protected port during the window.
2. When the set limit is exceeded, the whole port is added to `blocked_ports`.
3. nftables timeout automatically removes the ban.
4. Xray Runtime and inbound configuration are not modified during the ban.

## 1. Design Goal

### Why Port Guard Was Added

The project needed a conservative first version of port protection that could reduce abusive multi-IP access on a single inbound port without changing Xray Runtime behavior or client state.

The MVP focuses on one precise behavior:

- window-period unique IPv4 source IP count
- temporary whole-port ban
- automatic nftables timeout recovery

### Why nftables

nftables was selected because it provides:

- kernel-level packet filtering
- dynamic timeout sets
- low-overhead per-port tracking
- automatic expiry without a long-running daemon
- a clean separation from Xray Runtime and ProtocolModule

The implemented table is:

```text
table inet zui_port_guard
```

The main sets are:

```text
blocked_ports
pg4_<port>
```

### Why Runtime Is Not Used

Runtime was intentionally not used for Port Guard MVP.

Reasons:

- banning through Runtime would require removing or mutating Xray inbounds
- Runtime changes have higher blast radius
- temporary bans should not rewrite Xray config
- timeout recovery is simpler and safer in nftables
- existing protocol behavior remains untouched

Port Guard does not call Runtime RemoveInbound and does not modify ProtocolModule.

### Why IPv6 Is Not Included

The MVP is IPv4-only.

IPv6 was deferred because:

- the audited n-ui implementation was IPv4-focused
- IPv6 needs separate source-address sets and validation
- IPv6 deployment and NAT behavior are different
- the first version should minimize firewall complexity

The implementation must not generate:

```text
pg6_<port>
ip6 saddr
```

### Why Speed Limit Is Not Included

Port speed limiting was excluded from MVP.

Reasons:

- nftables drop rules are not real bandwidth shaping
- real rate control needs tc/ifb design and interface detection
- upload/download direction handling is more complex
- unsafe speed-limit rules can affect unrelated traffic

No tc, nft speedlimit, or iptables hashlimit logic was added.

### Why It Is Not Called Online IP Limit

This feature is not a real-time online IP counter.

It counts unique IPv4 source IPs observed within a time window. Therefore UI wording uses:

- `端口保护`
- `窗口期 IP 限制`
- `窗口期唯一 IPv4 数`

The UI explicitly states:

```text
这是窗口期唯一 IPv4 来源 IP 数，不是实时在线 IP 数。
```

## 2. Database Changes

Explicit migration:

```text
202606210001_port_guard
```

New `inbounds` columns:

| Column | Type | Default | Purpose |
| --- | --- | --- | --- |
| `port_guard_enabled` | boolean | `false` | Enables Port Guard on the inbound |
| `port_guard_window_seconds` | integer | `300` | Unique IPv4 observation window |
| `port_guard_ip_count` | integer | `0` | Allowed unique IPv4 count; `0` disables |
| `port_guard_ban_seconds` | integer | `300` | Temporary whole-port ban duration |
| `port_guard_banned_until` | integer | `0` | Runtime ban status timestamp |
| `port_guard_last_trigger_ip` | text | `""` | Last observed trigger IP |
| `port_guard_last_trigger_at` | integer | `0` | Last trigger timestamp |

New settings defaults:

| Setting | Default |
| --- | --- |
| `portGuardEnabled` | `true` |
| `portGuardWhitelistPorts` | `[]` |
| `portGuardSyncIntervalSeconds` | `30` |
| `portGuardLogRetentionDays` | `7` |

AutoMigrate was deliberately limited through a legacy inbound model so Port Guard columns are added by explicit migration.

## 3. API Changes

New login-protected API endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/port-guard/status` | Return Port Guard status |
| `POST` | `/api/port-guard/sync` | Run immediate sync |
| `POST` | `/api/port-guard/unban` | Manually unban a port |
| `GET` | `/api/port-guard/logs?limit=100` | Return recent Port Guard logs |

POST endpoints use CSRF protection.

API structure fields remain English for frontend compatibility.

User-visible API messages are Chinese:

- `获取端口保护状态成功`
- `同步端口保护规则成功`
- `解除端口保护封禁成功`
- `获取端口保护日志成功`

State values remain stable English enum values:

- `OFF`
- `ACTIVE`
- `BANNED`
- `WHITELISTED`
- `ERROR`

Display text is exposed separately through:

```json
"displayText": "正常"
```

## 4. UI Changes

Inbound form added a new section:

```text
端口保护
```

Fields:

- `启用端口保护`
- `窗口期秒数`
- `窗口期唯一 IPv4 数`
- `超限封禁秒数`

User-facing explanation:

```text
统计指定时间窗口内访问该端口的唯一 IPv4 来源 IP。超过限制后临时封禁整个端口。这是窗口期唯一 IPv4 来源 IP 数，不是实时在线 IP 数。
```

Inbound list added:

- Port Guard status column
- sync button
- log button
- manual unban button when state is banned

Chinese state display:

| Internal State | Display |
| --- | --- |
| `ACTIVE` | `正常` |
| `BANNED` | `已封禁` |
| `OFF` | `关闭` |
| `WHITELISTED` | `白名单保护` |
| `ERROR` | `错误` |

## 5. Installer Changes

`install.sh` now installs Port Guard runtime files when present in the release package:

- `scripts/zui-port-guard-sync` -> `/usr/local/bin/zui-port-guard-sync`
- `zui-port-guard-sync.service` -> `/etc/systemd/system/zui-port-guard-sync.service`
- `zui-port-guard-sync.timer` -> `/etc/systemd/system/zui-port-guard-sync.timer`

New directories:

- `/var/lib/z-ui/port-guard`
- `/var/log/z-ui`

New dependencies:

- `nftables`
- `sqlite3`

Install startup:

```text
systemctl enable --now zui-port-guard-sync.timer
systemctl start zui-port-guard-sync.service
```

Uninstall cleanup in `x-ui.sh` removes:

- timer
- service
- sync script
- nft table `inet zui_port_guard`
- `/var/lib/z-ui/port-guard`

Logs are preserved by default.

## 6. Command Changes

New management entry:

```text
x-ui port-guard
```

Subcommands:

| Command | Purpose |
| --- | --- |
| `x-ui port-guard status` | Show nftables and timer status |
| `x-ui port-guard sync` | Run immediate sync |
| `x-ui port-guard unban <port>` | Remove temporary ban and flush the port IP set |
| `x-ui port-guard logs` | Show recent Port Guard logs |

User-visible command output is Chinese.

Examples:

```text
端口保护状态
定时器: 运行中
同步服务: 未运行
```

```text
已解除端口 12568 的端口保护封禁
```

## 7. nftables Changes

Table:

```text
table inet zui_port_guard
```

Global blocked port set:

```text
set blocked_ports {
  type inet_service
  flags timeout,dynamic
}
```

Per protected port set:

```text
set pg4_<port> {
  type ipv4_addr
  flags timeout,dynamic
  timeout <window_seconds>s
  size <ip_count>
}
```

Chains:

```text
chain input {
  type filter hook input priority 0; policy accept;
}

chain output {
  type filter hook output priority 0; policy accept;
}
```

Rules:

- drop TCP/UDP destination port in `blocked_ports`
- drop TCP/UDP source port in `blocked_ports`
- accept already tracked IPv4 source IPs
- add new IPv4 source IPs to `pg4_<port>`
- when the per-port set is full, add the whole port to `blocked_ports`

Whitelist protection:

- SSH port
- panel port
- `80`
- `443`
- user configured whitelist ports

Whitelisted ports do not generate nftables per-port sets.

## 8. Files Changed

Modified files:

- `database/db.go`
- `database/migration.go`
- `database/model/model.go`
- `install.sh`
- `x-ui.sh`
- `web/assets/js/model/models.js`
- `web/controller/inbound.go`
- `web/html/xui/form/inbound.html`
- `web/html/xui/inbounds.html`
- `web/service/inbound.go`
- `web/service/setting.go`
- `web/web.go`
- `scripts/ui_dom_check.js`

Added files:

- `database/port_guard_migration_test.go`
- `web/controller/port_guard.go`
- `web/controller/port_guard_test.go`
- `web/service/port_guard.go`
- `web/service/port_guard_test.go`
- `scripts/zui-port-guard-sync`
- `zui-port-guard-sync.service`
- `zui-port-guard-sync.timer`

Deleted files:

- None

## 9. Test Results

Local validation passed:

- `go build ./...`
- `go test ./...`
- `bash -n install.sh`
- `bash -n x-ui.sh`
- `bash -n scripts/zui-port-guard-sync`
- `git diff --check`
- `scripts/rc_smoke.sh`
- `scripts/rc_upgrade_smoke.sh`
- sync script dry-run nft generation
- Playwright DOM check

Remote validation on `45.77.246.87` passed:

- `x-ui` active
- `zui-port-guard-sync.timer` active
- panel HTTP returned `200`
- nft table `inet zui_port_guard` exists
- Xray version `26.6.1`
- `/api/port-guard/status` returned success
- `x-ui port-guard status` printed Chinese output
- logs contain Chinese `message` values
- no IPv6 nftables rules were generated
- test inbounds were removed after verification

Known note:

- smoke scripts must run sequentially because they share `/etc/x-ui` test state.

## 10. Commits

Required Port Guard commits:

- `9338a83` Add Port Guard database fields and API
- `b5b1c46` Add Port Guard nftables sync script and systemd timer
- `5571ccb` Add Port Guard UI controls and status display
- `ffee765` Localize Port Guard user-facing text

## 11. Final State

Port Guard MVP is implemented and verified as IPv4-only.

It does not implement:

- IPv6
- port speed limit
- tc
- nftables speedlimit
- iptables shim
- Runtime RemoveInbound
- device limit
- per-client limit
- subscription system

Future expansion should be recorded in a new numbered Project History phase.
