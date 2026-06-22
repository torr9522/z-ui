# 015 Xray access.log 标准轮转

日期：2026-06-22

关联提交：

- 当前 Xray access.log logrotate 工作区版本

## 功能背景

访问来源分析会只读读取 Xray `access.log`。在 30G 小盘 VPS 上，如果 access.log 长期不轮转，可能造成磁盘占用持续增长。

本阶段只实现 OS 层日志生命周期管理，不修改 Xray 运行逻辑，不修改访问来源分析逻辑，不增加任何访问控制能力。

## 设计目标

- 防止 `/var/log/xray/access.log` 长期增长。
- 使用系统标准 `logrotate`。
- 每天轮转。
- 最多保留 7 份历史日志。
- 压缩旧日志。
- 使用 `copytruncate` 避免影响 Xray 持续写入。
- 不引入 cron 删除日志。
- 不引入 Go 后台清理任务。
- 不引入任何网络控制逻辑。

## 配置文件

仓库文件：

- `packaging/logrotate/x-ui-xray-access`

安装路径：

- `/etc/logrotate.d/x-ui-xray-access`

配置内容：

```text
/var/log/xray/access.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
```

## 安装器变化

`install.sh` 新增：

- 安装 `logrotate` 依赖。
- 创建 `/var/log/xray` 目录。
- 安装 `/etc/logrotate.d/x-ui-xray-access`。
- 如果系统存在 `logrotate.timer`，执行 `systemctl enable --now logrotate.timer`。

不新增：

- cron 清理。
- 删除日志脚本。
- Go 后台任务。
- Xray 重启逻辑。

## 卸载变化

`x-ui uninstall` 会删除：

- `/etc/logrotate.d/x-ui-xray-access`

不会删除：

- `/var/log/xray/access.log`
- 已轮转历史日志

## access-source 影响

access-source 模块仍然只读取当前 `access.log`。

`copytruncate` 行为会在轮转时复制旧文件并清空原文件，Xray 仍继续写入同一路径。access-source 的增量 offset 在日志变小时会重置内存聚合状态并继续读取当前文件。

## Xray 影响

不影响 Xray 运行：

- 不修改 Xray 配置。
- 不重启 Xray。
- 不改变 Xray 写日志路径。
- 不改变 Xray 进程文件句柄写入方式。

## 修改文件

新增：

- `packaging/logrotate/x-ui-xray-access`
- `scripts/logrotate_config_check.sh`
- `PROJECT_HISTORY/015_XRAY_ACCESS_LOGROTATE.md`

修改：

- `install.sh`
- `x-ui.sh`
- `README.md`
- `README_EN.md`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

## 测试结果

本地验证：

```text
bash -n install.sh
bash -n x-ui.sh
bash -n scripts/logrotate_config_check.sh
scripts/logrotate_config_check.sh
logrotate -d packaging/logrotate/x-ui-xray-access
go build ./...
go test ./...
sha256sum -c PROJECT_HISTORY/MANIFEST_SHA256.txt
```

结果：

- 全部通过。

## 风险说明

- `copytruncate` 是避免重启 Xray 的标准方案，但轮转瞬间仍可能存在系统日志轮转常见的极少量写入竞态。
- logrotate 执行周期由操作系统 `logrotate.timer` 或发行版默认计划任务负责。
- 本阶段不处理 error.log，只按需求处理 access.log。

## 最终行为

日志生命周期由 OS logrotate 负责，Xray 继续写当前 access.log，z-ui 继续只读分析当前 access.log。该方案无网络控制能力，无常驻进程，无额外 CPU 扫描任务。
