# 最新项目状态

## 项目

- 项目名称：`x-ui`
- 当前版本：`0.3.2`
- 当前分支：`z-ui`
- 最新审计阶段：`006_PORT_GUARD`

## 最近提交

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

## 当前功能树

核心目录：

- `protocol/`：协议解析和分享链接兼容逻辑
- `runtime/`：Xray 运行时协调
- `database/`：数据库结构、模型、显式迁移
- `web/controller/`：HTTP 控制器和 API 路由
- `web/service/`：应用服务层
- `web/html/`：面板模板
- `web/assets/`：前端模型和静态资源
- `scripts/`：冒烟测试和运维脚本
- `install.sh`：安装器
- `x-ui.sh`：管理命令

已实现的现代化能力：

- 数据库迁移
- ProtocolModule 收口
- 证书管理器
- 证书生命周期管理
- 自动续期
- HTTP+TLS
- REALITY / Vision / XHTTP 兼容
- 资源缓存版本
- 自托管仓库安装链接
- 端口保护 MVP

## 协议状态

支持协议：

- VMess
- VLESS
- Trojan
- Shadowsocks
- Dokodemo-door
- SOCKS
- HTTP
- mixed
- tunnel

端口保护与协议无关，因为它通过 nftables 保护入站端口，而不是改动协议运行时逻辑。

## 证书状态

证书系统状态：

- 证书管理器已存在
- 证书扫描已存在
- 面板 HTTPS 支持已存在
- ACME 流程已存在
- 自动续期 systemd timer 已存在
- IP 证书实验入口保留

## 端口保护状态

端口保护 MVP 状态：

- 已实现
- 仅 IPv4
- nftables 表：`inet zui_port_guard`
- 同步脚本：`/usr/local/bin/zui-port-guard-sync`
- 日志路径：`/var/log/z-ui/port-guard.log`
- 状态目录：`/var/lib/z-ui/port-guard`
- timer：`zui-port-guard-sync.timer`
- service：`zui-port-guard-sync.service`

端口保护已实现：

- 窗口期唯一 IPv4 来源 IP 统计
- 整个端口临时封禁
- 通过 nftables timeout 自动恢复
- 通过 API 和 `x-ui.sh` 手动解封
- 中文用户可见状态文本

端口保护未实现：

- IPv6
- 端口限速
- tc 整形
- nftables speedlimit
- iptables shim
- Runtime RemoveInbound
- 单客户端限制
- 设备限制

## 当前验证基线

最近端口保护验证已通过：

- `go build ./...`
- `go test ./...`
- `bash -n install.sh`
- `bash -n x-ui.sh`
- `bash -n scripts/zui-port-guard-sync`
- Playwright DOM 检查
- 远程部署检查：`45.77.246.87`
- 远程 `x-ui port-guard status`
- 远程 `/api/port-guard/status`

## 后续历史规则

下一个重要阶段必须创建：

- `PROJECT_HISTORY/007_<FEATURE>.md`

不要覆盖：

- `PROJECT_HISTORY/006_PORT_GUARD.md`

只更新：

- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`
