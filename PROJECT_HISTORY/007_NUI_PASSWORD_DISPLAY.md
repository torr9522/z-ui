# n-ui 风格账号密码显示记录

## 设计目标

本阶段按用户明确要求恢复 n-ui 风格的账号密码查看体验：本机 root 执行 `x-ui info` 或在中文菜单选择“面板信息”时，可以看到当前面板用户名和当前密码。

该行为用于兼容用户已有的服务器管理习惯，不用于绕过认证、不用于远程读取、不用于提权。

## 为什么恢复

用户希望面板管理脚本具备 n-ui 风格的本机运维体验：

- 首次安装后显示面板地址、用户名、密码和端口。
- `x-ui reset-user` 后显示新用户名和新密码。
- `x-ui info` 可查看当前用户名和当前密码。
- 中文菜单“面板信息”可查看当前密码。

## 安全影响

该模式会在数据库 `users.password` 中保存当前明文密码副本，因此拥有本机 root 权限的人可以读取当前面板密码。

保留的安全边界：

- 不恢复 `admin/admin`。
- `reset-user` 继续生成随机强账号密码。
- 登录校验优先使用 `users.password_hash`。
- session 仍只保存 userId / username / 登录状态。
- 登录失败日志不记录用户输入的密码。
- 不把密码写入普通登录失败日志。
- 服务进程非交互式启动时不向 systemd journal 输出首次随机密码。

## 当前折中方案

采用“双写、hash 优先校验”的折中方案：

- `users.password`：当前明文密码副本，仅用于本机 root 管理命令显示。
- `users.password_hash`：bcrypt 哈希，用于登录校验。

登录时：

- 优先校验 `users.password_hash`。
- 没有 hash 时兼容旧的 `users.password`。
- 不清空 `users.password`。

写入密码时：

- 同时写入 `users.password`。
- 同时写入 `users.password_hash`。

## 数据库变化

没有新增字段。

继续使用：

- `users.password`
- `users.password_hash`

行为变化：

- `users.password` 不再作为待清理旧字段。
- `users.password` 重新成为当前密码的明文显示副本。

## API 变化

无新增 API。

登录 API 行为保持不变：

- `/login`

安全变化：

- 登录失败日志不再输出用户提交的 password。

## UI 变化

Web 面板 UI 无新增页面。

中文交互菜单变化：

- “面板信息”显示当前密码。

## 安装器变化

`install.sh` 首次安装仍生成：

- 随机用户名
- 随机密码
- 随机端口

变化：

- 用户名生成调整为 10 位 `A-Za-z0-9`。
- 密码生成调整为 18 位 `A-Za-z0-9`。
- 调用 `x-ui setting -username ... -password ...` 后，Go 层会同时写入明文密码和 hash。

## 命令变化

`x-ui reset-user`：

- 生成随机用户名。
- 生成随机密码。
- 写入 `users.password`。
- 写入 `users.password_hash`。
- 终端显示新用户名和新密码。

`x-ui info`：

- 显示面板地址。
- 显示当前协议。
- 显示当前端口。
- 显示当前用户名。
- 显示当前密码。
- 显示服务状态。
- 显示当前版本。
- 显示当前 Commit。

如果 `users.password` 为空：

```text
当前密码：未保存明文，请使用 x-ui reset-user 重置
```

## 测试结果

本阶段要求验证：

- `go build ./...`
- `go test ./...`
- `bash -n install.sh`
- `bash -n x-ui.sh`
- `x-ui reset-user` 输出新用户名和新密码
- `x-ui info` 显示当前用户名和当前密码
- `users.password` 非空
- `users.password_hash` 非空
- 使用 `x-ui info` 显示的用户名密码可以登录
- 登录失败日志不包含用户输入的 password

## 最终提交

待提交。
