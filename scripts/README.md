# Xray Panel 脚本说明

## 脚本职责

| 文件 | 用途 |
|------|------|
| `scripts/install.sh` | **首次安装 & 生命周期管理**：install / update / uninstall / update-geo / status |
| `xray-panel.sh`（根目录）| **安装后交互式管理菜单**：服务控制、账户管理、Nginx、备份等 |

## 快速安装

### 在线安装

```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install.sh) install
```

指定版本 / 仓库：

```bash
PANEL_VERSION=v1.2.0 bash <(curl -Ls .../install.sh) install
GITHUB_REPO=myorg/xray-panel bash <(curl -Ls .../install.sh) install
```

### 本地安装（解压包内运行）

```bash
tar xzf xray-panel-v1.0.0-linux-amd64.tar.gz
cd xray-panel-v1.0.0-linux-amd64
bash scripts/install.sh install
```

## install.sh 命令参考

```
install.sh install              在线安装（最新版）
install.sh install <pkg.tar.gz> 本地安装（指定压缩包）
install.sh update [version]     更新面板（默认 latest）
install.sh uninstall            卸载面板
install.sh update-geo           更新 geoip.dat / geosite.dat
install.sh status               查看面板及 Geo 文件状态
```

### 定期更新 Geo 文件（cron）

使用 [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat) 增强版规则：

```bash
# 每周一凌晨 3 点自动更新
0 3 * * 1 bash /opt/xray-panel/scripts/install.sh update-geo >> /var/log/geo-update.log 2>&1
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GITHUB_REPO` | `nxovaeng/xray-panel` | GitHub 仓库 |
| `PANEL_VERSION` | `latest` | 安装/更新版本 |

## 系统要求

- Ubuntu 20.04+ / Debian 10+ / CentOS 8+
- AMD64 或 ARM64
- Root 权限
