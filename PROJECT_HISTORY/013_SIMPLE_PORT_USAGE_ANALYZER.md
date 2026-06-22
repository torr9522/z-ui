# 013 极简端口共享检测回滚

日期：2026-06-22

关联提交：

- 当前极简端口共享检测工作区版本

## 功能背景

访问来源分析曾升级为流量情报数据层和计算内核，但当前目标是回滚复杂分析系统，保留最小可用的端口共享检测能力。

本阶段将模块简化为：

```text
access.log
↓
解析 port / ip
↓
map[port]set[ip]
↓
统计
↓
API + UI 展示
```

## 设计目标

- 删除复杂时间维度分析。
- 删除多层统计模型。
- 删除统一计算内核。
- 只保留端口维度去重统计。
- API 保持 `GET /api/access-source/analytics`。
- 页面只展示端口、去重 IP 数、命中次数。
- 保持完全只读。

## 保留指标

每个端口仅保留：

- `port`
- `unique_ips`
- `total_hits`
- `last_seen`

## 删除内容

已从当前实现移除：

- 时间窗口引擎。
- 滚动窗口统计。
- 端口窗口统计结构。
- 复杂 IP 维度结构。
- 增量日志系统概念。
- 时间桶对齐逻辑。
- 趋势图和评分系统展示。

## API 变化

保留接口：

- `GET /api/access-source/analytics`

返回结构简化：

```json
{
  "ports": [
    {
      "port": 12345,
      "unique_ips": 100,
      "total_hits": 5000
    }
  ],
  "message": ""
}
```

## UI 变化

页面：

- `/xui/access-source`

标题：

- 端口共享检测

页面仅展示：

- 端口
- 去重 IP 数
- 命中次数

不再展示：

- 1 小时、24 小时、7 天。
- 时间周期统计。
- 趋势图。
- 评分系统。
- IP 维度表。

## 修改文件

修改：

- `web/service/access_source.go`
- `web/service/access_source_test.go`
- `web/html/xui/access_source.html`
- `scripts/access_source_dom_check.js`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

新增：

- `PROJECT_HISTORY/013_SIMPLE_PORT_USAGE_ANALYZER.md`

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

- IPv4 来源解析。
- IPv6 来源解析。
- loopback 来源过滤。
- 按端口去重 IP。
- 按端口统计命中次数。
- 最近访问时间保留。
- API 登录态保护。
- 页面 DOM 检查。

## 只读边界

本阶段只读取 `access.log` 并做内存聚合：

- 不写数据库。
- 不修改 Xray 配置。
- 不修改 Xray 运行逻辑。
- 不增加控制能力。
- 不触发任何自动动作。
- 不影响端口访问。

## 最终行为

访问来源分析已回滚为极简端口共享检测系统。它只统计每个端口出现过多少去重来源 IP 和命中次数，不包含时间分析系统，不包含复杂分析引擎，不具备任何控制能力。
