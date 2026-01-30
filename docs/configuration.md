# 配置文件说明

## 配置文件自动检测

Xray Panel 支持根据操作系统自动选择配置文件，无需手动指定。

### 检测顺序

程序启动时按以下顺序查找配置文件：

1. **命令行指定** - 如果使用 `-config` 参数，使用指定的文件
2. **OS 特定配置** - 根据操作系统查找对应的配置文件：
   - Windows: `conf/config.windows.yaml`
   - Linux: `conf/config.linux.yaml`
   - macOS: `conf/config.darwin.yaml`
3. **通用配置** - `conf/config.yaml`
4. **示例配置** - `conf/config.yaml.example`（会显示警告）
5. **默认配置** - 使用内置的默认值

### 使用方法

#### 方法 1: 自动检测（推荐）

```bash
# 直接运行，自动检测操作系统并加载对应配置
./panel
```

**Windows**:
```powershell
.\panel.exe
```

程序会自动：
1. 检测当前操作系统
2. 查找对应的配置文件
3. 输出使用的配置文件路径

**示例输出**:
```
Using config file: conf/config.windows.yaml
Starting xray-panel on 0.0.0.0:8082
```

#### 方法 2: 手动指定

```bash
# 使用自定义配置文件
./panel -config /path/to/custom.yaml
```

---

## 配置文件模板

### Windows 配置 (`config.windows.yaml`)

**特点**:
- 数据库使用相对路径 `data/panel.db`
- Xray 路径指向 `C:/Program Files/Xray/`
- 适合开发和测试环境
- 调试模式默认开启

**适用场景**:
- 本地开发
- Windows 服务器部署
- 测试环境

**路径说明**:
```yaml
database:
  path: "data/panel.db"  # 相对路径，方便开发

xray:
  binary_path: "C:/Program Files/Xray/xray.exe"
  config_path: "C:/Program Files/Xray/config.json"
```

---

### Linux 配置 (`config.linux.yaml`)

**特点**:
- 使用 Linux 标准路径
- 数据库位于 `/var/lib/xray-panel/`
- 配置文件位于 `/etc/xray-panel/`
- 适合生产环境
- 调试模式默认关闭

**适用场景**:
- 生产服务器
- VPS 部署
- Docker 容器

**路径说明**:
```yaml
database:
  path: "/var/lib/xray-panel/panel.db"  # 标准数据目录

xray:
  binary_path: "/usr/local/bin/xray"  # 官方安装脚本路径
  config_path: "/usr/local/etc/xray/config.json"
```

**systemd 服务**:
```bash
# 启动服务
systemctl start xray-panel

# 查看日志
journalctl -u xray-panel -f

# 查看管理员凭据
journalctl -u xray-panel | grep "初始管理员"
```

---

### macOS 配置 (`config.darwin.yaml`)

**特点**:
- 使用 macOS 标准路径
- 数据库位于用户 Library 目录
- 支持 Homebrew 安装的 Xray
- 适合开发环境

**适用场景**:
- macOS 开发
- 本地测试
- Homebrew 环境

**路径说明**:
```yaml
database:
  path: "~/Library/Application Support/xray-panel/panel.db"

xray:
  binary_path: "/usr/local/bin/xray"  # Homebrew 路径
```

**Homebrew 安装**:
```bash
# 安装 Xray
brew install xray

# 安装 Nginx（可选）
brew install nginx
```

---

## 配置项说明

### Server 配置

```yaml
server:
  listen: "0.0.0.0:8082"  # 监听地址和端口
  debug: true              # 调试模式
```

**listen**:
- `0.0.0.0:8082` - 监听所有网卡
- `127.0.0.1:8082` - 只监听本地
- `:8082` - 等同于 `0.0.0.0:8082`

**debug**:
- `true` - 开发模式，详细日志
- `false` - 生产模式，简洁日志

---

### Database 配置

```yaml
database:
  path: "data/panel.db"  # 数据库文件路径
```

**路径类型**:
- 相对路径: `data/panel.db` - 相对于程序运行目录
- 绝对路径: `/var/lib/xray-panel/panel.db` - 完整路径
- 用户目录: `~/data/panel.db` - 展开为用户主目录

**注意**:
- 程序会自动创建目录
- 确保有写入权限
- 定期备份数据库

---

### JWT 配置

```yaml
jwt:
  secret: "change-me-in-production"  # JWT 密钥
  expire_hour: 168                    # 过期时间（小时）
```

**secret**:
- 生产环境必须修改
- 建议使用 32+ 字符的随机字符串
- 可以使用命令生成: `openssl rand -base64 32`

**expire_hour**:
- 24 = 1 天
- 168 = 7 天
- 720 = 30 天

---

### Admin 配置

```yaml
admin:
  username: ""  # 留空自动生成
  password: ""  # 留空自动生成
  email: ""     # 可选
```

**自动生成（推荐）**:
- 留空所有字段
- 首次启动时自动生成
- 凭据输出到日志

**手动指定**:
```yaml
admin:
  username: "myadmin"
  password: "MyStrongPassword123!"
  email: "admin@example.com"
```

---

### Xray 配置

```yaml
xray:
  binary_path: "/usr/local/bin/xray"
  config_path: "/usr/local/etc/xray/config.json"
  assets_path: "/usr/local/share/xray"
  api_port: 10085
```

**binary_path**: Xray 可执行文件
**config_path**: Xray 配置文件（面板生成）
**assets_path**: geoip.dat 和 geosite.dat 位置
**api_port**: Xray API 监听端口

---

### Nginx 配置

```yaml
nginx:
  config_dir: "/etc/nginx/conf.d"
  stream_dir: "/etc/nginx/stream.d"
  reload_cmd: "systemctl reload nginx"
  cert_dir: "/etc/letsencrypt/live"
```

**config_dir**: HTTP 配置目录
**stream_dir**: Stream 配置目录（SNI 路由）
**reload_cmd**: 重载命令
**cert_dir**: SSL 证书目录

---

## 最佳实践

### 开发环境

1. 使用 OS 特定配置文件
2. 启用调试模式
3. 使用相对路径
4. 自动生成管理员凭据

```yaml
server:
  debug: true
database:
  path: "data/panel.db"
admin:
  username: ""
  password: ""
```

### 生产环境

1. 关闭调试模式
2. 使用绝对路径
3. 修改 JWT Secret
4. 配置 HTTPS
5. 限制访问 IP

```yaml
server:
  listen: "127.0.0.1:8082"  # 只监听本地
  debug: false
database:
  path: "/var/lib/xray-panel/panel.db"
jwt:
  secret: "your-random-secret-here"
```


## 配置文件优先级

1. 命令行参数 `-config`
2. OS 特定配置文件
3. 通用配置文件
4. 示例配置文件
5. 内置默认值

---

## 故障排查

### 配置文件未找到

**现象**: 使用默认配置启动

**解决**:
1. 检查配置文件是否存在
2. 检查文件名是否正确
3. 使用 `-config` 明确指定

### 路径权限问题

**现象**: 无法创建数据库或写入文件

**解决**:
```bash
# 创建目录
mkdir -p /var/lib/xray-panel

# 设置权限
chown xray-panel:xray-panel /var/lib/xray-panel
chmod 755 /var/lib/xray-panel
```

### 配置语法错误

**现象**: 启动失败，YAML 解析错误

**解决**:
1. 检查 YAML 语法（缩进、冒号）
2. 使用在线 YAML 验证器
3. 参考示例配置文件

---

## 总结

- ✅ 支持多操作系统自动检测
- ✅ 提供 OS 特定的配置模板
- ✅ 灵活的配置文件查找机制
- ✅ 支持相对路径和绝对路径
- ✅ 自动生成管理员凭据
- ✅ 适配开发和生产环境
