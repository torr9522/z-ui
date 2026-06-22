# 018 nftables Bootstrap Engine

日期：2026-06-22

关联提交：

- 当前 nftables Bootstrap Engine 工作区版本

## 阶段目标

实现 z-ui 专用 nftables Bootstrap Engine，用于安装时初始化 z-ui 隔离防火墙子系统。

核心目标：

- 自动检测 nftables。
- 缺失时按系统包管理器安装 nftables。
- 启用 nftables systemd 服务。
- 只创建 z-ui 专用 table：`inet zui_port_guard`。
- 只初始化 z-ui 自己需要的 chain 和 set。
- 不修改系统防火墙默认策略。
- 不影响 SSH。
- 不接管系统 INPUT policy。

## 新增脚本

新增：

- `scripts/nft_check.sh`
- `scripts/nft_init_zui.sh`
- `scripts/nft_bootstrap.sh`
- `scripts/nft_bootstrap_check.sh`

职责：

- `nft_check.sh`：检测、安装并启用 nftables 服务。
- `nft_init_zui.sh`：幂等初始化 `inet zui_port_guard`。
- `nft_bootstrap.sh`：组合执行 check 和 init。
- `nft_bootstrap_check.sh`：dry-run 验证 bootstrap 输出不包含危险操作。

## nftables 结构

Bootstrap 只创建：

```text
table inet zui_port_guard
```

chains：

```text
chain input {
    type filter hook input priority 0;
    policy accept;
}

chain output {
    type filter hook output priority 0;
    policy accept;
}
```

set：

```text
set blocked_ports {
    type inet_service
    flags timeout,dynamic
}
```

## 安装器集成

`install.sh` 安装文件后执行：

```text
scripts/nft_check.sh
scripts/nft_init_zui.sh
```

行为：

- 可重复执行。
- table 已存在时不重复创建。
- chain 已存在时不重复创建。
- set 已存在时不重复创建。
- 不删除用户自定义 nftables。
- 不 flush 任何系统规则。

## 禁止行为

本阶段没有引入：

- iptables。
- ufw。
- firewalld。
- conntrack。
- 全局 INPUT policy 修改。
- SSH 端口规则。
- 系统 filter table 接管。
- 全局 network blocking 规则。
- delete / flush / drop / reject 规则。

## 与 Port Guard 的关系

Bootstrap Engine 只提供 z-ui 专用执行容器：

- table
- chain
- blocked_ports set

Port Guard sync 脚本后续仍负责根据数据库配置生成端口级规则。

本阶段不修改 Port Guard 业务逻辑。

## 修改文件

新增：

- `scripts/nft_check.sh`
- `scripts/nft_init_zui.sh`
- `scripts/nft_bootstrap.sh`
- `scripts/nft_bootstrap_check.sh`
- `PROJECT_HISTORY/018_NFT_BOOTSTRAP_ENGINE.md`

修改：

- `install.sh`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

## 测试结果

本地验证：

```text
bash -n scripts/nft_check.sh
bash -n scripts/nft_init_zui.sh
bash -n scripts/nft_bootstrap.sh
bash -n scripts/nft_bootstrap_check.sh
ZUI_NFT_DRY_RUN=1 scripts/nft_bootstrap.sh
scripts/nft_bootstrap_check.sh
bash -n install.sh
go build ./...
go test ./...
sha256sum -c PROJECT_HISTORY/MANIFEST_SHA256.txt
```

结果：

- 全部通过。

## 风险说明

- Bootstrap 依赖系统包管理器安装 nftables。
- 如果系统禁用 systemd，服务启用步骤会跳过或失败，但初始化仍以 nft 命令可用为准。
- Bootstrap 只初始化 z-ui 专用 table，不保证用户系统防火墙整体状态。

## 最终行为

z-ui 安装时会自动准备独立 nftables 子系统 `inet zui_port_guard`。该子系统只作为 Port Guard 执行容器，不作为系统防火墙，不接管全局策略，不锁 SSH。
