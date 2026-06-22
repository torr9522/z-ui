# Build Metadata 实现记录

## 修改原因

变更前，`x-ui info` 的构建提交显示为：

```text
当前版本：0.3.2
当前 Commit：unknown
```

这无法准确判断当前服务器运行的二进制来自哪个分支、哪个提交、什么时候构建，不利于回滚、审计和远程排障。

本阶段只增强 Build Metadata 与 Project History，不修改密码逻辑、不修改数据库、不修改登录逻辑、不修改 Port Guard。

## 设计方案

在 `config` 包中新增构建元信息：

- `config.BuildCommit`
- `config.BuildBranch`
- `config.BuildTime`

构建时通过 `-ldflags` 注入：

```bash
COMMIT=$(git rev-parse --short HEAD)
BRANCH=$(git branch --show-current)
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
go build -o x-ui -ldflags "\
-X 'x-ui/config.BuildCommit=${COMMIT}' \
-X 'x-ui/config.BuildBranch=${BRANCH}' \
-X 'x-ui/config.BuildTime=${BUILD_TIME}'" .
```

同时保留内部兼容变量：

- `config.commit`
- `config.branch`
- `config.buildTime`

用于兼容旧示例或旧构建命令。

Commit 还支持 Go build info 中的 `vcs.revision` fallback。普通在 git 工作区内执行 `go build` 时，如果 Go 工具链写入 VCS 信息，也可以显示短提交。

## 修改文件

- `config/config.go`
- `main.go`
- `x-ui.sh`
- `README.md`
- `PROJECT_HISTORY/008_BUILD_METADATA.md`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

## 最终行为

Go 二进制新增：

```bash
x-ui version
```

输出：

```text
Version: 0.3.2
Commit: <commit 或 未知>
Branch: <branch 或 未知>
BuildTime: <build time 或 未知>
```

`x-ui info` 显示：

```text
当前版本：0.3.2
当前提交：94647bd
当前分支：z-ui
构建时间：2026-06-22 03:11:25
```

如果无法获取，显示：

```text
未知
```

不再显示：

```text
unknown
```

中文菜单“面板信息”同步显示：

- 当前版本
- 当前提交
- 当前分支
- 构建时间

## 验证结果

本阶段要求验证：

- `go build ./...`
- `go test ./...`
- `bash -n x-ui.sh`
- `x-ui info` 显示版本、提交、分支、构建时间
- 菜单“2. 面板信息”同步显示版本、提交、分支、构建时间
- `sha256sum -c PROJECT_HISTORY/MANIFEST_SHA256.txt`

## 相关 Commit

待提交。

## 风险说明

- 没有修改数据库结构。
- 没有修改登录校验。
- 没有修改密码保存逻辑。
- 没有修改 Port Guard。
- 未注入 ldflags 的手工构建会显示 `未知`，这是预期降级行为。
