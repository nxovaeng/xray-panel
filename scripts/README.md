# Xray Panel 脚本说明

## 脚本列表

| 脚本 | 说明 |
|------|------|
| `install-online.sh` | 在线安装（从 GitHub 下载）|
| `install-local.sh` | 本地安装（从压缩包）|
| `update.sh` | 更新面板 |
| `uninstall.sh` | 卸载面板 |
| `reset-admin.sh` | 重置管理员密码 |

## 快速开始

### 在线安装

```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install-online.sh)
```

### 本地安装

```bash
tar xzf xray-panel-v1.0.0-linux-amd64.tar.gz
cd xray-panel-v1.0.0-linux-amd64
bash scripts/install-local.sh
```

### 更新

```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/update.sh)
```

### 卸载

```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/uninstall.sh)
```

## 环境变量

### install-online.sh

- `PANEL_VERSION` - 安装版本（默认: latest）
- `GITHUB_REPO` - GitHub 仓库（默认: nxovaeng/xray-panel）

示例：
```bash
PANEL_VERSION="v1.0.0" GITHUB_REPO="username/repo" bash install-online.sh
```

## 系统要求

- Ubuntu 20.04+ / Debian 10+ / CentOS 8+
- AMD64 或 ARM64 架构
- Root 权限

## 相关文档

- [快速开始](../QUICK_START.md)
- [安装指南](../docs/installation-guide.md)
- [配置说明](../docs/configuration.md)
