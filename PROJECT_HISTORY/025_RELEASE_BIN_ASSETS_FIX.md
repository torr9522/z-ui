# 025 Release bin/ Assets Fix (v1.0.5)

## 故障现象

v1.0.4 安装后面板显示 Xray 状态 stop。

journalctl 报错：
1. failed to open geoip.dat > stat /usr/local/x-ui/geoip.dat: no such file or directory
2. 写入配置文件失败: open bin/config.json: no such file or directory

## 根因

release tar.gz 构建时未将 bin/ 目录打包进去，导致：
- bin/xray-linux-amd64 缺失 → xray 无法启动
- bin/geoip.dat / bin/geosite.dat 缺失 → routing config 报错
- bin/config.json 无法写入 → WorkingDirectory=/usr/local/x-ui 下 bin/ 不存在

## 修复

1. 新增 scripts/build_release.sh：强制将 bin/ 完整打包进 tar.gz
2. install.sh install_files()：增加 bin/ 完整性校验，缺失时 fail 终止
3. x-ui.sh cmd_update：增加新包 bin/ 完整性校验，缺失时拒绝更新
4. 版本升级至 v1.0.5

## 验证

- tar -tzf z-ui-linux-amd64.tar.gz | grep "x-ui/bin/" 确认 bin/ 存在
- 154.26.180.242 全新安装 v1.0.5 后 Xray 状态 running
- journalctl 无 geoip.dat / config.json 报错
