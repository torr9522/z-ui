# 017 v1.0 发布安装链路修复

日期：2026-06-22

关联提交：

- 当前 v1.0 发布安装链路修复工作区版本

## 问题背景

执行：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/torr9522/z-ui/z-ui/install.sh)
```

会安装旧版本 `0.3.2`。原因是 `install.sh` 通过 GitHub Releases latest API 解析最新 release，而当时 latest release 仍是旧的：

- `beta-v0.1.6-public-install`

因此 raw `install.sh` 虽然来自当前 `z-ui` 分支，但下载的 release asset 仍是旧包。

## 修复目标

- GitHub Release `v1.0.0` 成为唯一可信安装源。
- `install.sh` 固定下载 `v1.0.0`。
- 不再调用 GitHub latest release API。
- 不再 fallback 到 beta release。
- release asset 必须由当前 `z-ui` 分支 HEAD 构建。

## 安装器变化

修改：

- `install.sh`

行为变化：

- `XUI_RELEASE_VERSION` 固定为 `v1.0.0`。
- 下载 URL 固定为：

```text
https://github.com/torr9522/z-ui/releases/download/v1.0.0/z-ui-linux-<arch>.tar.gz
```

删除行为：

- 不再解析 `/releases/latest`。
- 不再根据 latest release 自动选择安装包。

## Release 资产要求

上传文件：

- `z-ui-linux-amd64.tar.gz`

包内必须包含：

- `x-ui` 二进制。
- `config/version = 1.0.0`。
- `PROJECT_HISTORY/016_RELEASE_STABLE.md`。
- `PROJECT_HISTORY/017_RELEASE_INSTALL_CHAIN_FIX.md`。
- `packaging/logrotate/x-ui-xray-access`。
- access-source 后端与 UI 文件。
- 完整仓库源码审计文件。

## 验证结果

验证项：

```text
bash -n install.sh
go build ./...
go test ./...
sha256sum -c PROJECT_HISTORY/MANIFEST_SHA256.txt
gh release view v1.0.0
bash <(curl -Ls https://raw.githubusercontent.com/torr9522/z-ui/z-ui/install.sh)
```

预期结果：

- 安装版本为 `1.0.0`。
- Commit 为当前 `z-ui` HEAD。
- 不出现 `0.3.2`。
- 不出现 `unknown commit`。
- 不拉取 `beta-v0.1.6-public-install`。

## 风险说明

- 本阶段只修复安装源，不修改业务逻辑。
- 本阶段不修改 access-source 统计逻辑。
- 本阶段不修改 Port Guard。
- 本阶段不修改网络控制逻辑。

## 最终行为

用户执行 raw `install.sh` 时，将固定安装 GitHub Release `v1.0.0` 中的当前稳定包，不再依赖 latest release 解析结果。
