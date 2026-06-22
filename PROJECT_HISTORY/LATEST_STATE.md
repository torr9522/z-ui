# 最新项目状态

## 项目

- 项目名称：`x-ui`
- 当前版本：`1.0.0`
- 当前分支：`z-ui`
- 最新审计阶段：`017_RELEASE_INSTALL_CHAIN_FIX`

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
- Port Guard 折叠面板
- 访问来源分析
- 流量情报数据层
- 流量情报计算内核
- 极简端口共享检测
- IP 画像增强层
- Xray access.log 标准轮转
- v1.0 稳定收尾
- v1.0 发布安装链路修复
- n-ui 风格账号密码显示兼容
- Build Metadata 构建元信息

## v1.0 稳定状态

当前系统定位：

- z-ui 是轻量级 Xray 观测与日志分析系统。
- 仅做日志分析与展示。
- 不做流量控制。
- 不做封禁或限速。
- 不做时间窗口风险分析。
- 不做复杂统计建模。
- 不修改系统网络规则。

职责边界：

- OS logrotate 负责日志生命周期。
- Xray 负责写日志。
- z-ui 只读 access.log 并展示端口访问来源概览。
- IP 画像为展示层，不参与任何控制。
- Port Guard 为独立端口保护能力，本阶段未修改。

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
- 入站表单中以高级功能折叠面板显示
- 新增入站默认收起
- 编辑已启用端口保护的入站时自动展开

端口保护未实现：

- IPv6
- 端口限速
- tc 整形
- nftables speedlimit
- iptables shim
- Runtime RemoveInbound
- 单客户端限制
- 设备限制

## 访问来源分析状态

访问来源分析状态：

- 已实现。
- 页面路径：`/xui/access-source`。
- API：`GET /api/access-source/analytics`。
- 数据来源：当前 Xray 配置或 settings template 中的 `access.log` 路径；如 `/var/log/xray/access.log` 存在则作为 fallback。
- 统计范围：首次读取大文件时最多读取 access log 末尾 `64MB`。
- 当前形态：极简端口共享检测 + IP 画像增强层。
- 内部逻辑：`access.log` -> 解析 `port/ip` -> `map[port]set[ip]` -> IP Profile Layer -> 聚合展示。
- 端口维度：端口、去重 IP 数、命中次数。
- 可选字段：最近访问时间、IP 画像列表。
- IP 画像字段：Geo、ASN、ISP、设备类型。
- Geo / ASN 来源：本机 MaxMind GeoLite2 City / ASN 数据库。
- IP 画像缓存：内存缓存，TTL 24 小时。
- 日志读取：保存 offset，只读取新增日志；日志轮转或截断时重置内存聚合。
- 批处理：30 秒内复用分析快照。
- API 返回：`ports`、`message`。
- UI 展示端口、去重 IP 数、命中次数、IP 画像。
- 不包含时间周期统计。
- 不包含趋势图。
- 不包含评分系统。
- 不包含风控判断。

访问来源分析安全边界：

- 只读读取和统计。
- 不写数据库。
- 不修改 Xray 配置。
- 不重启服务。
- 不调用端口保护模块。
- 不调用系统包过滤规则工具。
- 不修改防火墙或系统网络规则。
- 不影响端口访问。

## Xray access.log 轮转状态

Xray access.log 生命周期管理：

- 已实现。
- 仓库配置：`packaging/logrotate/x-ui-xray-access`。
- 安装路径：`/etc/logrotate.d/x-ui-xray-access`。
- 管理对象：`/var/log/xray/access.log`。
- 轮转周期：每天。
- 保留数量：7 份历史日志。
- 压缩：启用。
- 延迟压缩：启用。
- 空文件不处理：启用。
- 文件不存在不报错：启用。
- 写入不中断：使用 `copytruncate`。

职责边界：

- OS logrotate 负责日志生命周期。
- Xray 继续写当前 access.log。
- z-ui access-source 继续只读当前 access.log。
- 不修改 Xray 配置。
- 不重启 Xray。
- 不增加 cron 删除日志。
- 不增加 Go 后台日志清理任务。
- 不增加任何网络控制能力。

## 发布安装链路状态

安装链路：

- `install.sh` 固定下载 GitHub Release `v1.0.0`。
- 不再调用 GitHub Releases latest API。
- 不再自动选择 `beta-v0.1.6-public-install`。
- 不允许 fallback 到旧 release 包。

固定下载地址：

```text
https://github.com/torr9522/z-ui/releases/download/v1.0.0/z-ui-linux-<arch>.tar.gz
```

发布要求：

- `v1.0.0` release asset 必须由当前 `z-ui` HEAD 构建。
- `z-ui-linux-amd64.tar.gz` 必须包含完整仓库源码审计文件和当前构建产物。

## 当前验证基线

最近验证基线：

- `go build ./...`
- `go test ./...`
- `bash -n install.sh`
- `bash -n x-ui.sh`
- `bash -n scripts/zui-port-guard-sync`
- Playwright DOM 检查
- Port Guard 折叠面板 Playwright DOM 检查
- 远程部署检查：`45.77.246.87`
- 远程 `x-ui port-guard status`
- 远程 `/api/port-guard/status`
- 远程 Port Guard 折叠面板 DOM 检查
- 访问来源分析 service 单元测试
- 访问来源分析 API 登录态测试
- 访问来源分析 Playwright DOM 检查
- 流量情报数据层统一模型测试
- 流量情报时间窗口引擎测试
- 流量情报统一计算入口测试
- 极简端口共享检测聚合测试
- IP 画像缓存测试
- access.log 增量读取测试
- IP 画像 UI 展示检查
- logrotate 配置内容检查
- logrotate debug 解析检查
- v1.0 稳定收尾审计
- v1.0 release 安装链路校验

## 账号密码显示状态

n-ui 风格账号密码显示兼容状态：

- 已实现 `users.password` 明文副本。
- 已保留 `users.password_hash`。
- 登录校验优先使用 `users.password_hash`。
- `x-ui info` 显示当前用户名和当前密码。
- 中文菜单“面板信息”显示当前密码。
- `x-ui reset-user` 生成随机强账号密码并显示。
- 不恢复 `admin/admin`。
- 登录失败日志不记录用户输入的 password。
- session 仍只保存 userId / username / 登录状态。

## 当前构建信息系统

Build Metadata 状态：

- Version 来源：`config/version` 嵌入文件。
- Commit 来源：优先使用 `-ldflags -X x-ui/config.BuildCommit=...` 注入；未注入时尝试读取 Go build info 的 `vcs.revision`。
- Branch 来源：构建时通过 `-ldflags -X x-ui/config.BuildBranch=...` 注入。
- BuildTime 来源：构建时通过 `-ldflags -X x-ui/config.BuildTime=...` 注入。
- 未注入时显示：`未知`。
- `x-ui info` 会显示 Version、Commit、Branch、Build Time。
- 中文菜单“面板信息”会同步显示 Version、Commit、Branch、Build Time。

## 后续历史规则

最近新增阶段：

- `PROJECT_HISTORY/017_RELEASE_INSTALL_CHAIN_FIX.md`

下一个重要阶段必须创建：

- `PROJECT_HISTORY/018_<FEATURE>.md`

不要覆盖：

- `PROJECT_HISTORY/006_PORT_GUARD.md`
- `PROJECT_HISTORY/007_NUI_PASSWORD_DISPLAY.md`
- `PROJECT_HISTORY/008_BUILD_METADATA.md`
- `PROJECT_HISTORY/009_PORT_GUARD_COLLAPSE_UI.md`
- `PROJECT_HISTORY/010_ACCESS_SOURCE_ANALYTICS.md`
- `PROJECT_HISTORY/011_TRAFFIC_INTELLIGENCE_LAYER.md`
- `PROJECT_HISTORY/012_TRAFFIC_INTELLIGENCE_CORE_ENGINE.md`
- `PROJECT_HISTORY/013_SIMPLE_PORT_USAGE_ANALYZER.md`
- `PROJECT_HISTORY/014_IP_PROFILE_ENHANCEMENT.md`
- `PROJECT_HISTORY/015_XRAY_ACCESS_LOGROTATE.md`
- `PROJECT_HISTORY/016_RELEASE_STABLE.md`
- `PROJECT_HISTORY/017_RELEASE_INSTALL_CHAIN_FIX.md`

只更新：

- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`
