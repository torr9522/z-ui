# 最新项目状态

## 项目

- 项目名称：`x-ui`
- 当前版本：`1.0.6`
- 当前分支：`z-ui`
- 最新审计阶段：`026_RELEASE_COMMIT_PROVENANCE_FIX`

## 当前发布状态

- `v1.0.6` 修复了 release commit 溯源不一致问题。
- release 包内 `config/version` 由构建脚本在构建时生成。
- Go 二进制通过 `-ldflags` 注入 `BuildCommit` / `BuildBranch` / `BuildTime`。
- 构建阶段强制校验：
  - tag/source commit
  - package `config/version` commit
  - binary `x-ui version` commit
  必须一致，否则构建失败。

## 当前能力

- 统一协议 schema / validator 架构
- runtime dry-run 校验
- 系统证书自动发现与导入
- `x-ui update` 安全更新流程（保留 `/etc/x-ui`）
- release 包强制包含 `bin/xray-linux-{arch}`、`geoip.dat`、`geosite.dat`
- release provenance deterministic build

## 当前关键脚本

- `install.sh`：固定 release 版本安装，校验 `bin/` 完整性
- `x-ui.sh`：菜单管理、证书管理、安全更新
- `scripts/build_release.sh`：确定性 release 构建
- `scripts/verify_install.sh`：安装一致性检查
