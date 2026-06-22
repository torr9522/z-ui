# 019 install.sh 安装界面 UI 升级

日期：2026-06-22

关联提交：

- 当前 install.sh 安装 UI 升级工作区版本

## 阶段目标

将 `install.sh` 的线性文本输出整理为分阶段终端 UI，提升可读性和安装完成信息展示。

本阶段只修改安装输出结构，不修改业务逻辑。

## UI 结构

新增安装标题：

```text
╔════════════════════════════════════╗
║            z-ui Installer          ║
║     Lightweight · Stable · Secure  ║
╚════════════════════════════════════╝
```

安装流程分为：

- STEP 1：环境检测。
- STEP 2：下载资源。
- STEP 3：安装系统服务。
- STEP 4：初始化配置。
- STEP 5：安装完成。

安装完成后输出 box 风格信息面板：

- Panel URL。
- Username。
- Password。
- Port。
- Command。
- Config。

## 函数结构

新增或整理函数：

- `print_header()`
- `check_env()`
- `download_package()`
- `install_service()`
- `init_config()`
- `print_success_box()`
- `main()`

辅助输出函数：

- `step_title()`
- `ok_line()`
- `info_line()`
- `progress_line()`
- `box_row()`

## 安装流程保持

原流程保持：

1. root / OS / arch 检测。
2. 安装依赖。
3. 下载 release 包。
4. 备份旧数据库。
5. 安装文件。
6. 安装 logrotate 配置。
7. 初始化 nftables bootstrap。
8. 生成用户名、密码、端口。
9. 写入数据库配置。
10. 启动 x-ui。
11. 启动 Port Guard timer。
12. 输出安装成功信息。

## 禁止范围确认

本阶段未修改：

- Go backend。
- nftables 规则逻辑。
- x-ui 核心运行逻辑。
- Port Guard 业务逻辑。
- access-source 分析逻辑。
- release 下载版本。
- 系统依赖列表。

未引入：

- Python。
- Node.js。
- ncurses。
- dialog。
- whiptail。
- 新系统依赖。

## 修改文件

修改：

- `install.sh`
- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

新增：

- `PROJECT_HISTORY/019_INSTALLER_UI_REFRESH.md`

## 测试结果

本地验证：

```text
bash -n install.sh
go build ./...
go test ./...
sha256sum -c PROJECT_HISTORY/MANIFEST_SHA256.txt
```

结果：

- 全部通过。

## 最终行为

安装器以结构化终端 UI 展示安装过程和成功信息，但底层安装行为保持不变。
