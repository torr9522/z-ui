# Modern x-ui

Modern x-ui 是基于 x-ui 的现代化维护分支，目标是在保持原项目轻量结构的前提下升级到 Xray-core 26.x，并补齐安全、迁移、协议模块化、Runtime API、证书管理和安装脚本能力。

本仓库适合作为长期维护、二次开发和审计基线。项目仍然保持 Go + SQLite + 现有 Vue/Ant Design 模板的轻量架构，避免引入企业级复杂度。

## 功能列表

- Go 1.26 构建基线
- Xray-core 26.6.1 基线
- SQLite 持久化
- `schema_migrations` 数据库迁移
- `password_hash` 登录安全迁移
- `inbound_clients` 客户端拆分表
- ProtocolModule 接管协议配置生成
- Runtime API 同步与完整重启 fallback
- CSRF 与安全 Cookie
- 随机首次启动账号、密码、端口
- 安全安装脚本和 `x-ui` 管理命令
- 证书管理、ACME 域名证书、自动续期
- TLS 证书选择器
- HTTP + TLS 模式
- REALITY / Vision / XHTTP / mixed 支持

## 支持协议

- VMess
- VLESS
- Trojan
- Shadowsocks
- Dokodemo-door
- SOCKS
- HTTP
- mixed
- tunnel

不实现 TUIC，除非未来 Xray-core 源码明确提供原生 inbound 支持。

## 安装方式

安装脚本支持 Debian、Ubuntu、CentOS、Rocky、Alma 的 amd64/arm64 环境。

```bash
bash <(curl -Ls https://raw.githubusercontent.com/<your-repo>/x-ui/master/install.sh)
```

安装流程会生成随机用户名、随机强密码和 10000-59999 范围内的随机面板端口。登录信息只输出到当前终端。

## 管理命令

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

证书命令：

```bash
x-ui cert
x-ui cert status
x-ui cert renew
x-ui cert autorenew
```

`x-ui uninstall` 需要输入 `UNINSTALL` 才会继续，避免误删服务。

## 证书管理

证书目录：

```text
/etc/x-ui/certs/<name>/
  fullchain.pem
  privkey.pem
  meta.json
```

支持：

- 查看证书
- 导入证书
- 删除证书并备份
- 设置面板 HTTPS 证书
- 申请域名证书（acme.sh standalone HTTP-01）
- 查看续期状态
- 立即续期
- 自动续期开关

API：

```http
GET /api/certificates
```

API 只返回元数据和路径，不返回私钥内容。

## HTTP + TLS

项目不新增 `https` 协议。HTTPS 入站通过：

```text
protocol = http
streamSettings.security = tls
```

也就是：HTTP + TLS = HTTPS。

HTTP TLS 模式必须提供有效 `certificateFile` 和 `keyFile`。

## REALITY

REALITY 是独立安全层，不使用 TLS 证书。REALITY 字段包括：

- dest
- serverNames
- fingerprint
- privateKey
- publicKey
- shortIds
- spiderX

证书管理系统不参与 REALITY 配置。

## 开发环境

推荐：

- Go 1.26+
- Node.js，用于 Playwright DOM 检查
- systemd 环境，用于安装脚本和服务验证

准备依赖：

```bash
go mod download
```

## 构建命令

```bash
go build ./...
go build -o x-ui .
```

## 测试命令

```bash
go test ./...
bash -n install.sh
bash -n x-ui.sh
scripts/rc_smoke.sh
scripts/rc_upgrade_smoke.sh
node scripts/share_link_validate.js
```

DOM 自动化需要运行中的面板：

```bash
XUI_BASE_URL=http://127.0.0.1:PORT \
XUI_USERNAME=USERNAME \
XUI_PASSWORD=PASSWORD \
node scripts/ui_dom_check.js
```

## 目录说明

- `protocol/`：协议模块、验证、迁移、FormSchema、配置生成
- `runtime/`：Runtime API、Planner、Builder、Reconciler、fallback
- `database/`：SQLite 初始化和迁移
- `web/`：Gin 服务、Controller、Service、模板、前端资源
- `scripts/`：烟测、升级测试、DOM 检查、分享链接验证
- `xray/`：Xray 进程和配置包装
- `install.sh`：一键安装脚本
- `x-ui.sh`：管理命令脚本
- `knowledge-base/`：项目知识库和二开指南

## 安全说明

上传远程仓库前不得包含：

- 真实密码、Token、Cookie、Session
- SSH Key、私钥、证书私钥
- 真实数据库文件
- `node_modules/`
- `backup/`、`archive/`、`combined-backup/`
- `*.tar.gz`、`*.sha256`

`.gitignore` 已默认排除这些本地文件。

## 远程上传说明

推荐上传前检查：

```bash
git status --short
git ls-files | grep -E 'node_modules|backup|archive|combined-backup|\.tar\.gz|\.sha256|\.pem|\.key|\.crt|\.db' || true
go build ./...
go test ./...
bash -n install.sh
bash -n x-ui.sh
```

上传前建议阅读：

- `knowledge-base/00-project-overview.md`
- `knowledge-base/02-source-skill-tree.md`
- `knowledge-base/03-development-guide.md`
- `knowledge-base/11-lessons-learned.md`

## 归档说明

本地归档包不进入 Git 仓库。需要长期保存时，使用独立目录保存：

- Project Archive
- Combined Backup
- 对应 `.sha256`

归档内容应保存在仓库外部或被 `.gitignore` 排除。
