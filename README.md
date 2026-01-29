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

### 1. 下载或编译

#### 从 Release 下载（推荐）

访问 [Releases 页面](https://github.com/yourusername/xray-panel/releases) 下载适合你系统的版本：

- **Linux AMD64**: `xray-panel-vX.X.X-linux-amd64.tar.gz`
- **Linux ARM64**: `xray-panel-vX.X.X-linux-arm64.tar.gz`
- **Windows AMD64**: `xray-panel-vX.X.X-windows-amd64.zip`
- **Windows ARM64**: `xray-panel-vX.X.X-windows-arm64.zip`
- **macOS Intel**: `xray-panel-vX.X.X-darwin-amd64.tar.gz`
- **macOS Apple Silicon**: `xray-panel-vX.X.X-darwin-arm64.tar.gz`

#### 本地编译

```bash
# 当前平台
go build -o panel ./cmd/panel

# 所有平台（需要 Make）
make build-all VERSION=v1.0.0

# 或使用构建脚本
# Linux/macOS
./scripts/build-all.sh v1.0.0

# Windows
scripts\build-all.bat v1.0.0
```

详细构建说明请查看 [BUILD_GUIDE.md](BUILD_GUIDE.md)

### 2. 运行

```bash
# 自动检测操作系统并加载对应配置
./panel

# 或手动指定配置文件
./panel -config /path/to/config.yaml
```

### 3. 访问

```
http://localhost:8082
```

**首次启动**会自动生成管理员账户，凭据会输出到控制台：

```
========================================
🔐 初始管理员账户已创建
========================================
用户名: admin_k7m2p9
密码:   Xy9mK2pL4nQ8rT6v
========================================
⚠️  请立即登录并修改密码！
⚠️  请妥善保存这些凭据，它们不会再次显示！
========================================
```

## 配置文件

### 自动检测（推荐）

程序会根据操作系统自动选择配置文件：

- **Windows**: `conf/config.windows.yaml`
- **Linux**: `conf/config.linux.yaml`
- **macOS**: `conf/config.darwin.yaml`

### 配置示例

**Windows 开发环境**:
```yaml
server:
  listen: "0.0.0.0:8082"
  debug: true

database:
  path: "data/panel.db"  # 相对路径

admin:
  username: ""  # 留空自动生成
  password: ""  # 留空自动生成

xray:
  binary_path: "C:/Program Files/Xray/xray.exe"
  config_path: "C:/Program Files/Xray/config.json"
```

**Linux 生产环境**:
```yaml
server:
  listen: "127.0.0.1:8082"  # 只监听本地
  debug: false

database:
  path: "/var/lib/xray-panel/panel.db"

admin:
  username: ""  # 自动生成
  password: ""  # 自动生成

xray:
  binary_path: "/usr/local/bin/xray"
  config_path: "/usr/local/etc/xray/config.json"
  api_port: 10085

nginx:
  config_dir: "/etc/nginx/conf.d"
  reload_cmd: "systemctl reload nginx"
```

详细配置说明请查看 [配置文档](docs/configuration.md)

## 功能特性

### 用户管理
- ✅ 用户增删改查
- ✅ 流量限制和统计
- ✅ 到期时间管理
- ✅ 订阅链接生成（Base64/Clash/JSON）
- ✅ 二维码展示

### 入站管理
- ✅ VLESS 协议
- ✅ Trojan 协议
- ✅ 传输协议：WebSocket、gRPC、XHTTP
- ✅ 域名绑定
- ✅ 自动配置 Nginx 反向代理

### 出站管理
- ✅ SOCKS5 代理
- ✅ WireGuard (WARP)
- ✅ Trojan 落地
- ✅ 动态表单验证

### 路由规则
- ✅ 6 种规则类型（入站、域名、IP、GeoSite、GeoIP、协议）
- ✅ 优先级控制
- ✅ 快捷导入预设规则
- ✅ 实时生效

### 系统监控
- ✅ CPU 使用率
- ✅ 内存使用情况
- ✅ 磁盘使用情况
- ✅ 系统运行时间
- ✅ 用户统计

### 配置管理
- ✅ Xray 配置预览
- ✅ 配置文件生成
- ✅ 热更新支持（通过 API）
- ✅ Nginx 配置自动生成

## 架构说明

```
客户端 (VLESS/Trojan)
    ↓ TLS (443)
Nginx (反向代理 + TLS 终止)
    ↓ none (内部端口)
Xray (监听 127.0.0.1:10001+)
```

**优势**:
- 统一的 TLS 管理
- 支持 SNI 路由（多域名）
- 简化 Xray 配置
- 更好的性能和安全性

## 文档

- [快速构建指南](BUILD_GUIDE.md) - 如何构建项目
- [详细构建文档](docs/building.md) - 完整的构建说明
- [配置文件说明](docs/configuration.md)
- [日志系统文档](docs/logging.md)
- [CLI 命令使用](docs/cli-commands.md)
- [管理员账户安全](docs/admin-security.md)
- [用户与入站关系](docs/user-inbound-relationship.md)
- [订阅系统安全](docs/subscription-security.md)
- [配置生成与部署](docs/configuration.md)


## CLI 命令

```bash
# 显示版本
./panel -version

# 显示管理员账户信息
./panel -show-admin

# 重置管理员密码
./panel -reset-password -username=admin_xxx -password=NewPassword123!

# 指定配置文件
./panel -config /path/to/config.yaml
```

详细说明请查看 [CLI 命令文档](docs/cli-commands.md)

## 安全建议

1. ✅ 使用自动生成的强密码
2. ✅ 修改默认 JWT Secret
3. ✅ 启用 HTTPS
4. ✅ 配置 IP 白名单
5. ✅ 定期备份数据库
6. ✅ 定期更新系统

## 系统要求

- Go 1.21+ (编译)
- Xray-core (运行时)
- Nginx (可选，用于 TLS)
- SQLite (内置)

## 支持的平台

- ✅ Linux (AMD64, ARM64)
- ✅ Windows (AMD64, ARM64)
- ✅ macOS (Intel, Apple Silicon)

## 开发

```bash
# 克隆仓库
git clone https://github.com/yourusername/xray-panel.git
cd xray-panel

# 安装依赖
go mod download

# 运行（自动使用 Windows 配置）
go run ./cmd/panel

# 编译
go build -o panel.exe ./cmd/panel
```

## 许可证

MIT License

## 架构

```
客户端 (443)
    ↓
Nginx (TLS 终止)
    ↓
Xray (10001+)
```

## 技术栈

- **后端**: Go + Gin + GORM
- **前端**: htmx (14KB) + 60 行 JS
- **模板**: Go html/template


## 开发

```bash
# 安装依赖
go mod download

# 运行
go run cmd/panel/main.go

# 编译
make build
```

## License

MIT
