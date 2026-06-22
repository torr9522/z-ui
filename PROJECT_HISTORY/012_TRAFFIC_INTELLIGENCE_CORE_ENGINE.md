# 012 流量情报计算内核标准化

日期：2026-06-22

关联提交：

- 当前流量情报计算内核工作区版本

## 功能背景

流量情报数据层已经统一了 `AccessEvent`、`PortStat`、`IpStat`、`PortWindowStat` 和 `TrafficIntelligenceMeta`。本阶段继续标准化分析引擎，重点不是新增功能，而是统一时间窗口和统计口径，保证 API、UI、service 层展示完全来自同一计算结果。

## 设计目标

- 引入统一时间窗口引擎 `TimeWindowEngine`。
- 将统计入口统一为 `TrafficIntelligenceService.Compute()`。
- 窗口统计由 service 统一计算。
- UI 只展示 API 返回结果，不再兜底计算窗口值。
- controller 不做任何统计计算。
- 保持只读分析系统。

## 时间窗口引擎

新增：

- `TimeWindow`
- `TimeWindowEngine`

实现方法：

- `getWindow(start, end)`
- `getRollingWindow(duration)`
- `alignToMinute()`
- `alignToHour()`

窗口规则：

- rolling window 的结束点统一按分钟对齐到当前分钟上界。
- 窗口边界为闭区间。
- 1 小时、24 小时、7 天窗口都通过同一引擎生成。

## 统一统计入口

统计入口：

- `TrafficIntelligenceService.Compute(events, now)`

调用关系：

- `Analyze()` 负责读取 access log 和构造 meta。
- `Compute()` 负责所有聚合计算。
- `computeTrafficIntelligence()` 只接收事件和窗口引擎。
- controller 只返回 service 结果。
- UI 只展示 API 结果。

## 一致性规则

以下指标统一由 service 计算：

- `1h_unique_ips`
- `24h_unique_ips`
- `7d_unique_ips`
- `port.total_hits`
- `ip.first_seen`
- `ip.last_seen`
- `port.first_seen`
- `port.last_seen`

UI 不再使用旧字段做窗口兜底。

## 修改文件

修改：

- `web/service/access_source.go`
- `web/service/access_source_test.go`
- `web/html/xui/access_source.html`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

新增：

- `PROJECT_HISTORY/012_TRAFFIC_INTELLIGENCE_CORE_ENGINE.md`

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

- `TrafficIntelligenceService.Compute()` 聚合。
- `TimeWindowEngine.alignToMinute()`。
- `TimeWindowEngine.alignToHour()`。
- `TimeWindowEngine.getRollingWindow()`。
- 窗口边界包含规则。
- 1 小时、24 小时、7 天窗口统计。
- 端口总访问次数。
- IP 首次访问和最后访问。

## 只读边界

本阶段只改 access log 解析后的统计计算口径：

- 不写数据库。
- 不修改 Xray 配置。
- 不修改 Xray 运行逻辑。
- 不增加控制能力。
- 不触发任何自动动作。
- 不影响端口访问。

## 最终行为

流量情报层现在具备统一计算内核。API 和 UI 都只消费 `TrafficIntelligenceService.Compute()` 生成的结果，避免不同层各算一套，统计口径更可预测、可复现。
