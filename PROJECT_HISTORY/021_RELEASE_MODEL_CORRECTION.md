# 021 发布模型现实约束修正

日期：2026-06-22

关联提交：

- 当前发布模型修正工作区版本

## 问题

之前的发布记录把以下对象描述成必须强制回写一致：

- Git commit
- Git tag
- GitHub Release
- `install.sh`
- `config/version`

这在真实 Git 发布链路里并不成立，因为：

- Git commit 是源码事实来源。
- GitHub Release 是从 commit/tag 构建出的产物。
- `install.sh` 只是 release 下载入口。
- `config/version` 是构建时写入包内的只读元信息。

release 不应该反向修改 Git 历史，也不应该要求安装入口与源码提交做双向绑定。

## 修正后的模型

单向链路：

```text
git commit
  -> build release from commit
  -> git tag
  -> GitHub Release
  -> install.sh download entry
  -> server install
```

约束说明：

- Git commit 是唯一 source of truth。
- tag 只标记 commit，不反向驱动源码修改。
- release 只承载构建产物，不反向要求改 Git 历史。
- install.sh 只负责下载 release asset。
- 服务器安装结果只记录已安装产物的 commit，不反向要求本地源码一致。

## config/version 定位

`config/version` 改为包内只读记录文件，用于标记构建来源：

```text
version: 1.0.1
commit: <build commit>
build_time: <build time>
```

运行时读取规则：

- `Version` 优先来自该文件。
- `Commit` 优先来自构建注入，其次来自该文件。
- `BuildTime` 优先来自构建注入，其次来自该文件。

该文件不参与 Git tag 反向修正。

## install.sh 调整

保留原则：

- 只下载固定 release asset。
- 不追 Git commit。
- 不回写源码版本文件。

新增行为：

- 安装完成后读取已安装二进制的 `Version / Commit / Branch / BuildTime`。
- 这些值全部来自 release 包本身，而不是本地源码状态。

## 一致性校验

新增：

- `scripts/verify_install.sh`

用途：

- 读取 installed commit。
- 读取 release commit。
- 读取 local git commit（如果当前目录是源码仓库）。

输出规则：

- `consistent`：installed commit = release commit。
- `drift detected`：两者不同，或者本地源码提交已领先。

`drift detected` 只表示状态差异，不代表安装失败。
