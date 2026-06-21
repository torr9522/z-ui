# 006 端口保护 MVP

## 阶段摘要

端口保护是项目历史系统记录的第一个完整审计阶段。

该功能为入站端口增加仅 IPv4 的窗口期唯一来源 IP 保护。当配置窗口内访问某个入站端口的唯一 IPv4 来源 IP 数超过限制时，系统会临时封禁整个端口。

示例：

- 端口：`12568`
- 窗口期：`300` 秒
- 唯一 IPv4 限制：`3`
- 封禁时间：`300` 秒

行为：

1. nftables 在窗口期内跟踪受保护端口的唯一 IPv4 来源 IP。
2. 当 set 超过限制时，整个端口会被加入 `blocked_ports`。
3. nftables timeout 到期后自动解除封禁。
4. 封禁期间不修改 Xray Runtime，也不修改 inbound 配置。

## 1. 设计目标

### 为什么加入端口保护

项目需要一个保守的第一版端口保护能力，用于降低单个入站端口被多 IP 滥用的风险，同时不改变 Xray Runtime 行为和客户端状态。

MVP 只聚焦一个明确行为：

- 窗口期唯一 IPv4 来源 IP 统计
- 整个端口临时封禁
- nftables timeout 自动恢复

### 为什么选择 nftables

选择 nftables 的原因：

- 支持内核级包过滤
- 支持动态 timeout set
- 适合低开销的按端口跟踪
- 不需要常驻守护进程即可自动过期
- 能与 Xray Runtime 和 ProtocolModule 保持清晰隔离

实现使用的表：

```text
table inet zui_port_guard
```

主要 set：

```text
blocked_ports
pg4_<port>
```

### 为什么不使用 Runtime

端口保护 MVP 明确不使用 Runtime。

原因：

- 通过 Runtime 封禁会要求删除或修改 Xray inbound
- Runtime 改动影响面更大
- 临时封禁不应该重写 Xray 配置
- timeout 自动恢复交给 nftables 更简单、更安全
- 现有协议行为保持不变

端口保护不会调用 Runtime RemoveInbound，也不会修改 ProtocolModule。

### 为什么不做 IPv6

MVP 仅支持 IPv4。

IPv6 延后的原因：

- 已审计的 n-ui 实现主要围绕 IPv4
- IPv6 需要独立来源地址 set 和验证策略
- IPv6 部署环境和 NAT 行为不同
- 第一版应尽量降低防火墙复杂度

实现禁止生成：

```text
pg6_<port>
ip6 saddr
```

### 为什么不做限速

端口限速不进入 MVP。

原因：

- nftables drop 规则不等于真正带宽整形
- 真正限速需要 tc/ifb、网卡识别和方向处理
- 上传和下载限速语义更复杂
- 不安全的限速规则可能影响其它端口流量

本阶段没有加入 tc、nft speedlimit 或 iptables hashlimit 逻辑。

### 为什么不叫在线 IP 限制

该功能不是实时在线 IP 统计。

它统计的是指定时间窗口内观察到的唯一 IPv4 来源 IP。因此 UI 文案使用：

- `端口保护`
- `窗口期 IP 限制`
- `窗口期唯一 IPv4 数`

UI 明确提示：

```text
这是窗口期唯一 IPv4 来源 IP 数，不是实时在线 IP 数。
```

## 2. 数据库变更

显式迁移版本：

```text
202606210001_port_guard
```

新增 `inbounds` 字段：

| 字段 | 类型 | 默认值 | 用途 |
| --- | --- | --- | --- |
| `port_guard_enabled` | boolean | `false` | 是否启用该 inbound 的端口保护 |
| `port_guard_window_seconds` | integer | `300` | 唯一 IPv4 观察窗口 |
| `port_guard_ip_count` | integer | `0` | 允许的唯一 IPv4 数；`0` 表示关闭 |
| `port_guard_ban_seconds` | integer | `300` | 整个端口临时封禁时长 |
| `port_guard_banned_until` | integer | `0` | 运行态封禁截止时间戳 |
| `port_guard_last_trigger_ip` | text | `""` | 最近触发封禁的 IP |
| `port_guard_last_trigger_at` | integer | `0` | 最近触发封禁时间戳 |

新增 settings 默认值：

| 设置项 | 默认值 |
| --- | --- |
| `portGuardEnabled` | `true` |
| `portGuardWhitelistPorts` | `[]` |
| `portGuardSyncIntervalSeconds` | `30` |
| `portGuardLogRetentionDays` | `7` |

为了确保字段通过显式迁移加入，`AutoMigrate` 被限制为旧 inbound 结构，不直接自动增加端口保护字段。

## 3. API 变更

新增需要登录态的 API：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/port-guard/status` | 返回端口保护状态 |
| `POST` | `/api/port-guard/sync` | 立即执行同步 |
| `POST` | `/api/port-guard/unban` | 手动解除端口封禁 |
| `GET` | `/api/port-guard/logs?limit=100` | 返回最近端口保护日志 |

POST 接口使用 CSRF 保护。

API 结构字段保持英文，避免破坏前端兼容性。

用户可见 API message 使用中文：

- `获取端口保护状态成功`
- `同步端口保护规则成功`
- `解除端口保护封禁成功`
- `获取端口保护日志成功`

内部状态枚举保持稳定英文值：

- `OFF`
- `ACTIVE`
- `BANNED`
- `WHITELISTED`
- `ERROR`

中文显示文本通过独立字段返回：

```json
"displayText": "正常"
```

## 4. UI 变更

Inbound 表单新增区域：

```text
端口保护
```

字段：

- `启用端口保护`
- `窗口期秒数`
- `窗口期唯一 IPv4 数`
- `超限封禁秒数`

用户说明：

```text
统计指定时间窗口内访问该端口的唯一 IPv4 来源 IP。超过限制后临时封禁整个端口。这是窗口期唯一 IPv4 来源 IP 数，不是实时在线 IP 数。
```

Inbound 列表新增：

- 端口保护状态列
- 同步按钮
- 日志按钮
- 已封禁状态下的手动解封按钮

中文状态显示：

| 内部状态 | 显示文本 |
| --- | --- |
| `ACTIVE` | `正常` |
| `BANNED` | `已封禁` |
| `OFF` | `关闭` |
| `WHITELISTED` | `白名单保护` |
| `ERROR` | `错误` |

## 5. 安装器变更

`install.sh` 在发布包中存在相关文件时安装端口保护运行文件：

- `scripts/zui-port-guard-sync` -> `/usr/local/bin/zui-port-guard-sync`
- `zui-port-guard-sync.service` -> `/etc/systemd/system/zui-port-guard-sync.service`
- `zui-port-guard-sync.timer` -> `/etc/systemd/system/zui-port-guard-sync.timer`

新增目录：

- `/var/lib/z-ui/port-guard`
- `/var/log/z-ui`

新增依赖：

- `nftables`
- `sqlite3`

安装后启动：

```text
systemctl enable --now zui-port-guard-sync.timer
systemctl start zui-port-guard-sync.service
```

`x-ui.sh` 卸载时清理：

- timer
- service
- sync 脚本
- nft 表 `inet zui_port_guard`
- `/var/lib/z-ui/port-guard`

日志默认保留。

## 6. 命令变更

新增管理入口：

```text
x-ui port-guard
```

子命令：

| 命令 | 用途 |
| --- | --- |
| `x-ui port-guard status` | 显示 nftables 和 timer 状态 |
| `x-ui port-guard sync` | 立即执行同步 |
| `x-ui port-guard unban <port>` | 解除临时封禁并清空该端口 IP set |
| `x-ui port-guard logs` | 显示最近端口保护日志 |

用户可见命令输出使用中文。

示例：

```text
端口保护状态
定时器: 运行中
同步服务: 未运行
```

```text
已解除端口 12568 的端口保护封禁
```

## 7. nftables 变更

表：

```text
table inet zui_port_guard
```

全局封禁端口 set：

```text
set blocked_ports {
  type inet_service
  flags timeout,dynamic
}
```

每个受保护端口一个 set：

```text
set pg4_<port> {
  type ipv4_addr
  flags timeout,dynamic
  timeout <window_seconds>s
  size <ip_count>
}
```

链：

```text
chain input {
  type filter hook input priority 0; policy accept;
}

chain output {
  type filter hook output priority 0; policy accept;
}
```

规则：

- 丢弃命中 `blocked_ports` 的 TCP/UDP 目标端口
- 丢弃命中 `blocked_ports` 的 TCP/UDP 源端口
- 放行已经记录在 `pg4_<port>` 的 IPv4 来源 IP
- 将新的 IPv4 来源 IP 加入 `pg4_<port>`
- 当端口 set 已满时，将整个端口加入 `blocked_ports`

白名单保护：

- SSH 端口
- 面板端口
- `80`
- `443`
- 用户自定义白名单端口

白名单端口不会生成对应的 nftables 端口 set。

## 8. 修改文件

修改文件：

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

新增文件：

- `database/port_guard_migration_test.go`
- `web/controller/port_guard.go`
- `web/controller/port_guard_test.go`
- `web/service/port_guard.go`
- `web/service/port_guard_test.go`
- `scripts/zui-port-guard-sync`
- `zui-port-guard-sync.service`
- `zui-port-guard-sync.timer`

删除文件：

- 无

## 9. 测试结果

本地验证已通过：

- `go build ./...`
- `go test ./...`
- `bash -n install.sh`
- `bash -n x-ui.sh`
- `bash -n scripts/zui-port-guard-sync`
- `git diff --check`
- `scripts/rc_smoke.sh`
- `scripts/rc_upgrade_smoke.sh`
- sync 脚本 dry-run nft 生成检查
- Playwright DOM 检查

远程服务器 `45.77.246.87` 验证已通过：

- `x-ui` active
- `zui-port-guard-sync.timer` active
- 面板 HTTP 返回 `200`
- nft 表 `inet zui_port_guard` 存在
- Xray 版本 `26.6.1`
- `/api/port-guard/status` 返回成功
- `x-ui port-guard status` 输出中文
- 日志包含中文 `message` 值
- 未生成 IPv6 nftables 规则
- 测试 inbound 已在验证后删除

已知注意点：

- smoke 脚本必须顺序执行，因为它们共享 `/etc/x-ui` 测试状态。

## 10. 已完成提交

端口保护阶段提交：

- `9338a83` Add Port Guard database fields and API
- `b5b1c46` Add Port Guard nftables sync script and systemd timer
- `5571ccb` Add Port Guard UI controls and status display
- `ffee765` Localize Port Guard user-facing text

## 11. 最终状态

端口保护 MVP 已实现并验证为仅 IPv4。

本阶段未实现：

- IPv6
- 端口限速
- tc
- nftables speedlimit
- iptables shim
- Runtime RemoveInbound
- 设备限制
- 单客户端限制
- 订阅系统

未来扩展必须记录在新的编号项目历史阶段中。
