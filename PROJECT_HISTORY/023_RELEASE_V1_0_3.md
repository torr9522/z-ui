# 023 - v1.0.3 发布

日期：2026-06-23

## 背景

`x-ui.sh` 已新增面板账号密码管理与面板端口管理二级菜单。需要发布新的 GitHub Release，使安装脚本固定安装包含最新管理脚本的 release 包。

## 发布版本

- 版本号：`v1.0.3`
- 安装脚本固定下载：`v1.0.3`
- Release 资产：`z-ui-linux-amd64.tar.gz`
- 校验文件：`SHA256SUMS`

## 变更范围

修改：

- `x-ui.sh`
- `install.sh`
- `config/version`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

新增：

- `PROJECT_HISTORY/022_X_UI_SH_PANEL_MANAGEMENT_MENU.md`
- `PROJECT_HISTORY/023_RELEASE_V1_0_3.md`

## 功能摘要

- 主菜单新增面板账号密码管理入口。
- 主菜单新增面板端口管理入口。
- 支持自定义账号密码。
- 支持自定义面板端口。
- 随机端口范围固定为 `10000-59999`。
- 保持 `x-ui reset-user` 与 `x-ui reset-port` 命令兼容。

## 禁止修改范围确认

本次未修改：

- Go backend
- protocol schema engine
- validator
- runtime builder
- Port Guard
- nftables

## 验证

发布前执行：

- `bash -n x-ui.sh`
- `go build ./...`
- `go test ./...`
- `sha256sum -c PROJECT_HISTORY/MANIFEST_SHA256.txt`

发布后执行远程安装验证：

- 安装版本为 `1.0.3`
- 主菜单显示 `3. 面板账号密码管理` 和 `4. 面板端口管理`
- 账号密码管理二级菜单可进入
- 端口管理二级菜单可进入
- `x-ui set-port 21121` 可成功设置
- `x-ui reset-port` 随机端口范围为 `10000-59999`
- `x-ui set-user testuser TestPass123` 可成功设置
- `x-ui reset-user` 可随机生成账号密码
- 服务重启后保持 `active`
