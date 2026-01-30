# Xray Panel

轻量级 Xray 管理面板，基于 Go + HTMX，支持多操作系统。

## 特性

- 🚀 **轻量高效** - 单文件部署，资源占用低
- 🔐 **安全可靠** - 自动生成强密码，JWT 认证
- 🌍 **跨平台** - 支持 Windows、Linux、macOS
- 📊 **实时监控** - 系统资源、用户流量统计
- 🔄 **热更新** - 支持 Xray API 热更新（无需重启）
- 🎨 **现代界面** - 基于 HTMX，无需复杂前端框架

## 快速开始

### 方式一：一键安装（推荐）

适用于 Linux 服务器，自动安装所有依赖：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install-online.sh)
```

详细说明请查看 [快速开始](QUICK_START.md) 和 [安装指南](docs/installation-guide.md)。

### 方式二：本地安装 & 开发环境

请查看 [BUILD_GUIDE.md](BUILD_GUIDE.md)。

## 便捷管理

安装后可使用 `xray-panel` 命令打开管理菜单：

```bash
xray-panel
```

## 目录结构

安装后所有文件统一在 `/opt/xray-panel`：

```
/opt/xray-panel/
├── panel              # 主程序
├── conf/              # 配置文件
├── data/              # 数据库
└── logs/              # 日志
```

## 文档

- [快速开始](QUICK_START.md)
- [安装指南](docs/installation-guide.md)
- [配置文档](docs/configuration.md)
- [CLI 命令文档](docs/cli-commands.md)
- [日志系统文档](docs/logging.md)
- [构建指南](BUILD_GUIDE.md)

## 常用 CLI 命令

```bash
# 启动服务器
./panel server

# 显示版本
./panel version

# 显示管理员账户信息
./panel admin

# 重置管理员密码
./panel reset-password -username=admin_xxx -password=NewPassword123!

# 指定配置文件启动
./panel server -config /path/to/config.yaml
```

## 系统要求

- Go 1.21+ (编译)
- Xray-core (运行时)
- Nginx (可选，用于 TLS)
- SQLite (内置)

## 许可证

MIT License
