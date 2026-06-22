# 010 访问来源分析

日期：2026-06-21

关联提交：

- `94647bd` Restore Chinese interactive x-ui menu
- 当前访问来源分析工作区版本

## 功能背景

需要基于当前 `access.log` 或已有访问记录能力，提供一个只读的访问来源分析页面，用于观察入站端口的来源 IP 数、访问次数和访问时间趋势。

本阶段明确不是安全防护系统，不实现任何封禁、拦截、限速或网络规则变更。

## 设计目标

- 按端口统计去重来源 IP 数、总访问次数、首次访问时间、最近访问时间。
- 按 IP 统计访问次数、首次出现时间、最后出现时间和访问端口列表。
- 提供 1 小时、24 小时、7 天窗口内的端口去重 IP 数。
- 数据来源仅限 `access.log` 和已有 Xray 配置中的 access log 路径。
- 页面只读展示，不写数据库，不修改 Xray 配置，不重启服务。

## 不实现内容

- 不实现封禁、拦截、丢弃或限制行为。
- 不调用系统包过滤规则工具。
- 不调用传统包过滤规则工具。
- 不修改防火墙或系统网络规则。
- 不调用端口保护模块。
- 不调用运行时移除入站逻辑。
- 不使用连接跟踪。
- 不增加自动任务、后台聚合或清空记录功能。

## 数据来源

读取顺序：

1. 当前运行配置 `bin/config.json` 中的 `log.access`。
2. settings 中的 Xray config template 的 `log.access`。
3. 如果 `/var/log/xray/access.log` 存在，则作为 fallback 读取。

读取限制：

- 最多读取末尾 `64MB` 日志，避免大日志拖慢页面。
- 时间按服务器本地时区解析和展示。
- 无 access log 时返回空报告和中文提示。

## API 变化

新增只读 API：

- `GET /api/access-source/analytics`

登录要求：

- 必须登录态。

返回主要字段：

- `logPath`
- `logConfigured`
- `logExists`
- `timezone`
- `generatedAt`
- `truncated`
- `message`
- `uniqueIpCount`
- `totalHits`
- `ports`
- `ips`

## UI 变化

新增页面：

- `/xui/access-source`

新增菜单：

- 访问来源分析

页面展示：

- 只读分析说明。
- 日志路径。
- 日志状态。
- 去重来源 IP。
- 总访问次数。
- 端口概览卡片。
- 端口统计表。
- IP 维度统计表。

用户可见安全边界文案：

- 本页面仅读取 `access.log` 并统计访问来源，不执行封禁、不修改防火墙、不影响端口访问，也不会触发端口保护。

## 修改文件

新增：

- `web/service/access_source.go`
- `web/service/access_source_test.go`
- `web/controller/access_source.go`
- `web/controller/access_source_test.go`
- `web/html/xui/access_source.html`
- `scripts/access_source_dom_check.js`
- `PROJECT_HISTORY/010_ACCESS_SOURCE_ANALYTICS.md`

修改：

- `web/web.go`
- `web/controller/xui.go`
- `web/html/xui/common_sider.html`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

## 测试结果

本地验证：

```text
go build ./...
go test ./...
bash -n x-ui.sh
bash -n install.sh
```

结果：

- 全部通过。

测试覆盖：

- IPv4 access log 解析。
- IPv6 access log 解析。
- loopback 来源过滤。
- 端口维度聚合。
- IP 维度聚合。
- 1 小时、24 小时、7 天窗口统计。
- API 未登录访问保护。
- Playwright DOM 检查访问来源分析页面只读提示和统计区域。

## 风险说明

- 这是基于 access log 的历史访问统计，不代表实时在线连接数。
- 如果 Xray 未配置 access log，页面只能显示空报告。
- 大日志只读取末尾 `64MB`，较早记录不会进入本次统计。
- 日志格式变化可能导致部分记录无法识别。
- 不做 IP 风险判断，不做异常检测，不做自动动作。

## 最终行为

访问来源分析只读取日志并生成统计结果，用于运维观察端口使用情况、来源增长趋势和共享趋势。

该功能不会影响任何端口访问，不会修改系统网络规则，不会触发端口保护。
