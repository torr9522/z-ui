# 009 Port Guard 折叠面板改造

## 日期

2026-06-21

## 关联提交

- `5571ccb Add Port Guard UI controls and status display`
- `ffee765 Localize Port Guard user-facing text`
- 当前折叠面板改造工作区版本

## 功能背景

Port Guard 在新增入站页面占用空间过大。

原问题：

- 默认展开。
- 表单高度明显增加。
- 移动端体验较差。
- 大多数用户不会使用 Port Guard。

本阶段目标：

- 将 Port Guard 改为高级功能折叠面板。
- 新增入站默认收起。
- 编辑已启用 Port Guard 的入站时自动展开。
- 不修改 API。
- 不修改数据库字段。
- 不修改 Port Guard 后端逻辑。

## 修改文件

- `web/html/xui/form/inbound.html`
- `web/html/xui/inbound_modal.html`
- `scripts/port_guard_collapse_dom_check.js`

## 实现内容

默认标题：

```text
端口保护（高级功能）
```

新增入站：

- 默认收起。
- 标题状态显示 `关闭`。

编辑已有入站：

- 如果 `portGuardEnabled=true`，自动展开。
- 标题状态显示 `已启用`。

支持行为：

- 点击标题展开。
- 再次点击标题收起。
- 展开后显示原有说明和字段。
- 保存仍使用原有字段，不改变前后端数据结构。

展开后字段：

- 说明文字。
- 启用端口保护开关。
- 窗口期秒数。
- 窗口期唯一 IPv4 数。
- 超限封禁秒数。

保留说明：

```text
这是窗口期唯一 IPv4 来源 IP 数，不是实时在线 IP 数。
```

## 验证结果

Playwright DOM 验证通过：

```text
defaultCollapsed=true
clickExpands=true
clickCollapses=true
savedRowVisible=true
editEnabledAutoExpands=true
cleanupRemoved=true
```

验证覆盖：

- 新增入站默认折叠。
- 点击展开。
- 点击收起。
- 启用 Port Guard 后保存。
- 编辑已启用 Port Guard 入站自动展开。
- 测试入站清理成功。

## 移动端验证

视口：

```text
390x844
```

验证通过：

- 折叠状态单行显示。
- 点击展开正常。
- 点击收起正常。
- 编辑已启用入站自动展开正常。
- 保存正常。

## 远程同步验证

服务器：

```text
45.77.246.87
```

同步时间：

```text
2026-06-21 19:24:52
```

远程运行版本：

```text
Version: 0.3.2
Commit: 94647bd
Branch: z-ui
BuildTime: 2026-06-21 19:24:52
```

Port Guard 状态：

- 正常。
- `zui-port-guard-sync.timer` 为 `active`。
- `x-ui port-guard status` 正常执行。

远程 DOM 验证：

- 通过。

远程测试入站残留：

```text
0
```

## SHA256

本阶段新增或更新以下文件到 `PROJECT_HISTORY/MANIFEST_SHA256.txt`：

- `web/html/xui/form/inbound.html`
- `web/html/xui/inbound_modal.html`
- `scripts/port_guard_collapse_dom_check.js`
- `PROJECT_HISTORY/009_PORT_GUARD_COLLAPSE_UI.md`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

校验命令：

```bash
sha256sum -c PROJECT_HISTORY/MANIFEST_SHA256.txt
```

校验结果：

```text
全部通过
```

## 最终状态

Port Guard 折叠面板改造完成并已同步到远程测试服务器。

本阶段没有修改：

- `PROJECT_HISTORY/006_PORT_GUARD.md`
- `PROJECT_HISTORY/007_NUI_PASSWORD_DISPLAY.md`
- `PROJECT_HISTORY/008_BUILD_METADATA.md`
