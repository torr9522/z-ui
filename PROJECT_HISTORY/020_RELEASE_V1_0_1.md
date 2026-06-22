# 020 v1.0.1 正式发布

日期：2026-06-22

关联提交：

- 当前 v1.0.1 发布工作区版本

## 背景

在 `v1.0.0` 阶段，安装脚本本身已经固定到 release 下载链路，但 GitHub Release 资产仍落后于当前 `z-ui` 分支最新提交，导致：

- 本地源码 HEAD
- GitHub Release 资产
- raw `install.sh` 安装结果

三者不一致。

## 目标

建立严格一致的生产发布链路：

- Git commit
- Git tag
- GitHub Release
- `install.sh` 下载地址
- 远程安装结果

全部指向同一稳定版本。

## 本阶段变更

修改：

- `install.sh`
- `config/version`
- `config/config.go`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

新增：

- `PROJECT_HISTORY/020_RELEASE_V1_0_1.md`

## 版本策略

发布版本：

- `v1.0.1`

安装脚本固定下载：

```text
https://github.com/torr9522/z-ui/releases/download/v1.0.1/z-ui-linux-<arch>.tar.gz
```

## 版本元信息

`config/version` 从单行版本号升级为结构化元信息格式：

```text
version=1.0.1
commit=<release commit>
build_time=<release build time>
```

运行时解析规则：

- `Version` 优先读取 `version=`
- `Commit` 优先读取构建时注入，其次读取 `commit=`
- `BuildTime` 优先读取构建时注入，其次读取 `build_time=`

这样即使构建时没有完整注入，也不会回退到错误的旧版本显示。

## 最终行为

用户执行：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/torr9522/z-ui/z-ui/install.sh)
```

将固定安装 GitHub Release `v1.0.1` 中的正式稳定包，不再落到旧 release 资产。
