# 016 v1.0 稳定收尾

日期：2026-06-22

关联提交：

- 当前 v1.0 稳定收尾工作区版本

## 阶段目标

本阶段只做收尾与稳定化整理，不新增功能，不引入复杂分析系统，不修改 Port Guard 行为，不修改网络控制逻辑。

目标是将当前系统整理为稳定生产版本 v1.0。

## 当前系统定位

z-ui 是一个轻量级 Xray 观测与日志分析系统：

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

## 版本状态

当前稳定版本：

- `1.0.0`

版本来源：

- `config/version`

构建元信息：

- `BuildCommit`
- `BuildBranch`
- `BuildTime`

运行时可通过：

- `x-ui info`

查看版本、提交、分支和构建时间。

## 日志系统状态

Xray access.log 生命周期已由 logrotate 接管：

- 配置文件：`/etc/logrotate.d/x-ui-xray-access`
- 仓库文件：`packaging/logrotate/x-ui-xray-access`
- 策略：daily、rotate 7、compress、delaycompress、missingok、notifempty、copytruncate

本阶段未修改 logrotate 策略。

## access-source 状态

access-source 当前稳定形态：

- 只读读取 access.log。
- 极简端口共享检测。
- 统计端口、去重 IP 数、命中次数。
- IP 画像仅用于展示。
- 使用内存缓存降低 GeoIP / ASN 查询成本。
- 使用 offset 只读新增日志。

不包含：

- 时间窗口风险分析。
- 趋势评分。
- 异常检测。
- 访问控制。
- 自动封禁。
- 限速。

## IP 画像状态

IP 画像为展示层：

- Geo。
- ASN。
- ISP。
- DeviceType。

IP 画像不参与：

- 访问控制。
- 封禁。
- 限速。
- 风控判断。
- Port Guard 决策。

## Port Guard 状态

本阶段未修改 Port Guard。

Port Guard 当前状态保持：

- IPv4-only。
- 窗口期唯一 IPv4 来源 IP 数限制。
- nftables timeout 自动恢复。
- UI 折叠高级设置。
- 中文用户可见文本。

本阶段没有新增任何 Port Guard 能力。

## 代码整理结果

检查范围：

- `web/service/access_source.go`
- `web/controller/access_source.go`
- `web/html/xui/access_source.html`
- `scripts/access_source_dom_check.js`
- `packaging/logrotate/x-ui-xray-access`

结果：

- 未发现运行时代码中的临时调试输出。
- 未发现 access-source 中的网络控制调用。
- 未发现时间窗口风险分析系统。
- 未发现复杂统计模型新增。

`scripts/access_source_dom_check.js` 中保留验证结果输出，仅用于测试脚本，不属于运行时代码。

## 修改文件

修改：

- `config/version`
- `README.md`
- `README_EN.md`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

新增：

- `PROJECT_HISTORY/016_RELEASE_STABLE.md`

## 测试结果

本地验证：

```text
go build ./...
go test ./...
bash -n install.sh
bash -n x-ui.sh
bash -n scripts/logrotate_config_check.sh
bash -n scripts/zui-port-guard-sync
scripts/logrotate_config_check.sh
sha256sum -c PROJECT_HISTORY/MANIFEST_SHA256.txt
```

结果：

- 全部通过。

## 风险说明

- access-source 的 IP 画像准确性依赖本机 MaxMind 数据库。
- access-source 的内存状态在进程重启后会重新从日志读取。
- logrotate 的执行周期依赖操作系统标准 logrotate timer 或发行版默认调度。
- Port Guard 仍是独立端口保护能力，但本阶段未修改它。

## 最终结论

当前系统已收敛为 v1.0 稳定版本。日志生命周期、只读访问来源分析、IP 画像展示、Port Guard 独立保护能力和构建元信息均已形成明确边界。
