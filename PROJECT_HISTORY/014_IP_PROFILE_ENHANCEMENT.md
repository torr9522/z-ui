# 014 IP 画像增强层

日期：2026-06-22

关联提交：

- 当前 IP 画像增强工作区版本

## 功能背景

极简端口共享检测已经回滚为 `access.log -> port/ip 统计 -> API + UI` 的只读结构。当前阶段在不改变统计模型的前提下，为已统计出的来源 IP 增加轻量画像信息，帮助运维人员观察访问来源分布。

本阶段保持核心结构：

```text
access.log
↓
port/ip 统计
↓
IP Profile Layer
↓
API + UI 展示
```

## 设计目标

- 保持极简 `port -> ip set -> count` 统计结构。
- 不恢复时间窗口、滚动窗口或复杂分析引擎。
- 不写数据库。
- 不修改 Xray 配置。
- 不重启 Xray。
- 不调用 Port Guard。
- 不调用任何系统网络控制能力。
- 基于本机 MaxMind GeoLite2 数据库补充 IP 画像。
- 使用内存缓存降低 GeoIP / ASN 查询成本。
- 使用日志 offset 增量读取，避免每次请求全量扫描。
- 使用 30 秒批处理间隔，避免实时逐行分析。

## IP 画像字段

每个来源 IP 新增：

- `ip`
- `geo`
- `country_code`
- `city`
- `asn`
- `isp`
- `device_type`

`geo` 来源：

- `GeoLite2-City.mmdb`

`asn` 和 `isp` 来源：

- `GeoLite2-ASN.mmdb`

本机默认查找路径：

- `/usr/share/GeoIP/GeoLite2-City.mmdb`
- `/usr/local/share/GeoIP/GeoLite2-City.mmdb`
- `/var/lib/GeoIP/GeoLite2-City.mmdb`
- `./GeoLite2-City.mmdb`
- `/usr/share/GeoIP/GeoLite2-ASN.mmdb`
- `/usr/local/share/GeoIP/GeoLite2-ASN.mmdb`
- `/var/lib/GeoIP/GeoLite2-ASN.mmdb`
- `./GeoLite2-ASN.mmdb`

如果数据库不存在，画像字段使用未知值，不影响端口统计。

## ISP 与设备类型

ISP 清洗规则：

- `China Mobile`
- `China Telecom`
- `China Unicom`
- `AWS`
- `Google`
- `Oracle`
- `Alibaba`
- 其他组织名截断为前三个词
- 空值显示为 `未知`

设备类型：

- `CLOUD_SERVER`
- `HOME_BROADBAND`
- `MOBILE`
- `DATACENTER`
- `UNKNOWN`

UI 中文显示：

- 云服务器
- 家庭宽带
- 手机网络
- 数据中心
- 未知

## 性能优化

IP 画像缓存：

- 内存 `map[ip]IPProfile`。
- TTL：24 小时。
- 命中缓存时不重复查询 MaxMind 数据库。
- 进程重启后缓存自动丢失，不做持久化。

增量日志读取：

- 服务保存当前 access.log offset。
- 后续分析只读取新增日志内容。
- 日志截断或轮转时重置内存聚合状态。
- 首次读取大文件时最多读取末尾 `64MB`。

批处理：

- 分析结果 30 秒内复用内存快照。
- 超过 30 秒才读取新增日志并更新聚合。
- 不启动后台任务。
- 不实时逐行分析。

## API 变化

保留接口：

- `GET /api/access-source/analytics`

返回结构保持 `ports` 为主：

```json
{
  "ports": [
    {
      "port": 12345,
      "unique_ips": 100,
      "total_hits": 5000,
      "ips": [
        {
          "ip": "1.2.3.4",
          "geo": "CN/Guangdong/Shenzhen",
          "asn": 9808,
          "isp": "China Mobile",
          "device_type": "MOBILE"
        }
      ]
    }
  ],
  "message": ""
}
```

接口仍然是只读接口，不新增任何写操作。

## UI 变化

页面：

- `/xui/access-source`

新增展示：

- IP 画像区域。
- 每个端口最多展示前 8 个来源 IP。
- 每个 IP 显示 IP、Geo、ISP、设备类型。
- 支持常见国家或地区旗帜显示。

保留展示：

- 端口。
- 去重 IP 数。
- 命中次数。

不展示：

- 1 小时统计。
- 24 小时统计。
- 7 天统计。
- 趋势图。
- 评分系统。
- 风险标记。

## 修改文件

修改：

- `go.mod`
- `go.sum`
- `web/service/access_source.go`
- `web/service/access_source_test.go`
- `web/html/xui/access_source.html`
- `scripts/access_source_dom_check.js`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

新增：

- `PROJECT_HISTORY/014_IP_PROFILE_ENHANCEMENT.md`

## 测试结果

本地验证：

```text
node --check scripts/access_source_dom_check.js
go build ./...
go test ./...
bash -n install.sh
bash -n x-ui.sh
bash -n scripts/zui-port-guard-sync
```

结果：

- 全部通过。

测试覆盖：

- access.log 行解析。
- loopback 来源过滤。
- 端口去重 IP 统计。
- 命中次数统计。
- 最近访问时间统计。
- 增量日志读取。
- 30 秒批处理快照复用。
- IP 画像缓存命中。
- IP 画像缓存过期后重新查询。
- ISP 清洗。
- 设备类型推断。
- API 登录态保护。
- UI DOM 语法检查。

## 只读边界

本阶段没有引入任何控制能力：

- 不修改 Xray 配置。
- 不重启 Xray。
- 不写数据库。
- 不调用 Port Guard。
- 不调用系统防火墙。
- 不修改系统网络规则。
- 不影响端口访问。

## 风险说明

- GeoIP / ASN 结果依赖本机 MaxMind 数据库质量和更新时间。
- MaxMind 数据库不存在时显示未知，不影响统计。
- 设备类型仅基于 ASN 组织名字符串推断，不代表精确设备识别。
- 进程重启后内存缓存和 offset 会丢失，下一次请求会重新从 access.log 读取数据。
- 首次读取超大 access.log 时仅扫描末尾 `64MB`，这是为了适配低内存 VPS。

## 最终行为

访问来源分析保持极简端口共享检测系统，同时增加只读 IP 画像增强层。系统仍然只读取 access.log 并做内存统计，不包含时间分析系统，不包含风控系统，不具备任何封禁、限速或网络控制能力。
