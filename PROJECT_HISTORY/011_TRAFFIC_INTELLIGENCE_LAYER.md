# 011 流量情报数据层重构

日期：2026-06-22

关联提交：

- 当前流量情报数据层工作区版本

## 功能背景

访问来源分析初版已经可以基于 `access.log` 做端口和 IP 维度统计。本阶段不是新增控制能力，而是将原有单一分析服务重构为统一的“流量情报数据层”，让后续 ASN、趋势、图表等展示扩展都基于同一套结构化数据模型。

## 设计目标

- 保留 `GET /api/access-source/analytics` 接口路径。
- 将内部入口升级为 `TrafficIntelligenceService`。
- 将日志行解析为统一 `AccessEvent`。
- 生成统一的端口统计、IP 统计和端口窗口统计。
- API 返回新增 `meta` 和 `windows`。
- 页面展示“流量情报分析层”说明。

## 统一数据模型

新增核心结构：

- `AccessEvent`：原始访问事件，包含 `port`、`ip`、`timestamp`、`protocol`、`bytes_in`、`bytes_out`。
- `PortStat`：端口统计，包含 `port`、`unique_ip_count`、`total_hits`、`first_seen`、`last_seen`。
- `IpStat`：IP 统计，包含 `ip`、`total_hits`、`first_seen`、`last_seen`、`ports`。
- `PortWindowStat`：窗口统计，包含 `port`、`1h_unique_ips`、`24h_unique_ips`、`7d_unique_ips`、`growth_rate`。
- `TrafficIntelligenceMeta`：元信息，包含 `log_source`、`log_status`、`last_read_time` 等。

## API 变化

保留接口：

- `GET /api/access-source/analytics`

返回结构升级：

- `meta`
- `ports`
- `ips`
- `windows`

`meta` 包含：

- `log_source`
- `log_configured`
- `log_status`
- `timezone`
- `generated_at`
- `last_read_time`
- `log_modified_at`
- `truncated`
- `message`
- `global_unique_ip_count`
- `global_total_hits`

## UI 变化

页面：

- `/xui/access-source`

文案更新：

- “流量情报分析层”
- “本模块基于 Xray access.log 做只读结构化分析”
- “不代表真实在线用户”
- “不影响系统运行或网络配置”
- “不涉及任何封禁或限制功能”

页面继续展示：

- 端口概览
- 端口统计
- IP 维度统计
- 时间窗口统计
- access log 读取状态

## 只读边界

本阶段只读取 `access.log` 并做结构化分析：

- 不写数据库。
- 不修改日志配置。
- 不重启服务。
- 不修改 Xray 运行逻辑。
- 不增加控制能力。
- 不做风险判断。
- 不做异常标记。
- 不触发任何自动动作。

## 修改文件

修改：

- `web/service/access_source.go`
- `web/service/access_source_test.go`
- `web/html/xui/access_source.html`
- `scripts/access_source_dom_check.js`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

新增：

- `PROJECT_HISTORY/011_TRAFFIC_INTELLIGENCE_LAYER.md`

## 测试结果

本地验证：

```text
node --check scripts/access_source_dom_check.js
go build ./...
go test ./...
bash -n x-ui.sh
bash -n install.sh
```

结果：

- 全部通过。

测试覆盖：

- `AccessEvent` 解析。
- IPv4 / IPv6 来源解析。
- loopback 来源过滤。
- `PortStat` 聚合。
- `IpStat` 聚合。
- `PortWindowStat` 聚合。
- `log_status` 计算。
- API 登录态保护。
- 页面 DOM 文案检查。

## 风险说明

- 这是日志分析层，不代表实时在线用户。
- 未配置 access log 时只能返回空报告。
- 最多读取日志末尾 `64MB`。
- 日志格式变化可能导致部分记录无法解析。
- `bytes_in` 和 `bytes_out` 当前预留，Xray access log 默认格式中未稳定提供时保持为空。

## 最终行为

访问来源分析已升级为统一流量情报数据层。该层只负责读取日志、解析事件、生成结构化统计结果，为后续展示和趋势扩展打基础，不具备任何控制能力。
