# 安装文档

## 系统要求

### 最低配置
- CPU: 1 核心
- 内存: 512 MB
- 磁盘: 10 GB
- 系统: Linux (Ubuntu 20.04+, Debian 10+, CentOS 8+)

### 推荐配置
- CPU: 2 核心
- 内存: 2 GB
- 磁盘: 20 GB
- 系统: Ubuntu 22.04 LTS

### 支持的架构
- AMD64 (x86_64)
- ARM64 (aarch64)

## 一键安装（推荐）

### 使用安装脚本

**默认安装**（使用 nxovaeng/xray-panel 仓库）:
```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/main/scripts/install.sh)
```

**自定义仓库**:
```bash
# 如果你 fork 了项目
GITHUB_REPO="your-username/xray-panel" bash <(curl -Ls https://raw.githubusercontent.com/your-username/xray-panel/main/scripts/install.sh)
```

**指定版本**:
```bash
PANEL_VERSION="v1.0.0" bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/main/scripts/install.sh)
```

**同时指定仓库和版本**:
```bash
GITHUB_REPO="your-username/xray-panel" PANEL_VERSION="v1.0.0" bash <(curl -Ls https://raw.githubusercontent.com/your-username/xray-panel/main/scripts/install.sh)
```

安装脚本会自动：
1. 检测操作系统和架构
2. 安装必要的依赖
3. 安装 Xray-core
4. 下载并安装 Panel
5. 配置 Nginx
6. 创建 systemd 服务
7. 启动所有服务

### 查看管理员凭据

安装完成后，运行以下命令查看自动生成的管理员凭据：

```bash
xray-panel -show-admin
```

## 手动安装

### 1. 安装依赖

#### Ubuntu/Debian
```bash
apt-get update
apt-get install -y curl wget unzip tar nginx sqlite3 certbot python3-certbot-nginx
```

#### CentOS/RHEL
```bash
yum install -y epel-release
yum install -y curl wget unzip tar nginx sqlite certbot python3-certbot-nginx
```

### 2. 安装 Xray-core

```bash
bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install
```

### 3. 下载 Panel

```bash
# 创建目录
mkdir -p /opt/xray-panel
cd /opt/xray-panel

# 下载最新版本（替换为实际的架构）
ARCH="amd64"  # 或 arm64
wget https://github.com/yourusername/xray-panel/releases/latest/download/xray-panel-linux-${ARCH}.tar.gz

# 解压
tar xzf xray-panel-linux-${ARCH}.tar.gz

# 创建符号链接
ln -s /opt/xray-panel/panel-linux-${ARCH} /usr/local/bin/xray-panel
chmod +x /usr/local/bin/xray-panel
```

### 4. 创建配置文件

```bash
# 创建目录
mkdir -p /etc/xray-panel/conf
mkdir -p /var/lib/xray-panel
mkdir -p /var/log/xray-panel

# 创建配置文件
cat > /etc/xray-panel/conf/config.yaml <<EOF
server:
  listen: "127.0.0.1:8082"
  debug: false

database:
  path: "/var/lib/xray-panel/panel.db"

jwt:
  secret: "$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 32)"
  expire_hour: 168

admin:
  username: ""
  password: ""

xray:
  binary_path: "/usr/local/bin/xray"
  config_path: "/usr/local/etc/xray/config.json"
  assets_path: "/usr/local/share/xray"
  api_port: 10085

nginx:
  config_dir: "/etc/nginx/conf.d"
  stream_dir: "/etc/nginx/stream.d"
  reload_cmd: "systemctl reload nginx"
  cert_dir: "/etc/letsencrypt/live"

log:
  level: "info"
  file: "/var/log/xray-panel/panel.log"
  max_size: 100
  max_backups: 7
  max_age: 30
  compress: true
EOF
```

### 5. 配置 Nginx

```bash
# 创建 stream 目录
mkdir -p /etc/nginx/stream.d

# 添加 stream 支持到 nginx.conf
cat >> /etc/nginx/nginx.conf <<EOF

# Stream configuration for SNI routing
stream {
    include /etc/nginx/stream.d/*.conf;
}
EOF

# 创建 Panel 代理配置
cat > /etc/nginx/conf.d/xray-panel.conf <<EOF
server {
    listen 80;
    server_name _;
    
    location / {
        proxy_pass http://127.0.0.1:8082;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
EOF

# 测试并重载 Nginx
nginx -t && systemctl reload nginx
```

### 6. 创建 Systemd 服务

```bash
cat > /etc/systemd/system/xray-panel.service <<EOF
[Unit]
Description=Xray Panel Service
Documentation=https://github.com/yourusername/xray-panel
After=network.target nss-lookup.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/xray-panel
ExecStart=/usr/local/bin/xray-panel -config /etc/xray-panel/conf/config.yaml
Restart=on-failure
RestartSec=10
LimitNOFILE=65536

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/xray-panel /var/log/xray-panel /etc/xray-panel

[Install]
WantedBy=multi-user.target
EOF

# 重载 systemd
systemctl daemon-reload
systemctl enable xray-panel
```

### 7. 启动服务

```bash
# 启动 Xray
systemctl enable xray
systemctl start xray

# 启动 Panel
systemctl start xray-panel

# 启动 Nginx
systemctl enable nginx
systemctl start nginx
```

### 8. 查看管理员凭据

```bash
xray-panel -show-admin
```

## Docker 安装（开发中）

```bash
# 拉取镜像
docker pull yourusername/xray-panel:latest

# 运行容器
docker run -d \
  --name xray-panel \
  -p 8082:8082 \
  -v /etc/xray-panel:/etc/xray-panel \
  -v /var/lib/xray-panel:/var/lib/xray-panel \
  yourusername/xray-panel:latest
```

## 配置 SSL 证书

### 使用 Let's Encrypt

```bash
# 安装 certbot
apt-get install -y certbot python3-certbot-nginx

# 获取证书（替换为你的域名）
certbot --nginx -d yourdomain.com

# 自动续期
certbot renew --dry-run
```

### 手动配置 SSL

编辑 `/etc/nginx/conf.d/xray-panel.conf`:

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://127.0.0.1:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}

server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}
```

## 验证安装

### 检查服务状态

```bash
# Panel 状态
systemctl status xray-panel

# Xray 状态
systemctl status xray

# Nginx 状态
systemctl status nginx
```

### 查看日志

```bash
# Panel 日志
journalctl -u xray-panel -f

# 或查看日志文件
tail -f /var/log/xray-panel/panel.log

# Xray 日志
journalctl -u xray -f

# Nginx 日志
tail -f /var/log/nginx/error.log
```

### 测试访问

```bash
# 获取服务器 IP
curl ifconfig.me

# 访问 Web 界面
# http://YOUR_SERVER_IP
```

## 更新

### 使用更新脚本

**更新到最新版本**:
```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/main/scripts/update.sh)
```

**更新到指定版本**:
```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/main/scripts/update.sh) v1.0.0
```

**自定义仓库**:
```bash
GITHUB_REPO="your-username/xray-panel" bash <(curl -Ls https://raw.githubusercontent.com/your-username/xray-panel/main/scripts/update.sh)
```

### 手动更新

```bash
# 停止服务
systemctl stop xray-panel

# 备份
cp /usr/local/bin/xray-panel /usr/local/bin/xray-panel.backup

# 下载新版本
cd /tmp
wget https://github.com/yourusername/xray-panel/releases/latest/download/xray-panel-linux-amd64.tar.gz
tar xzf xray-panel-linux-amd64.tar.gz

# 替换二进制文件
mv panel-linux-amd64 /usr/local/bin/xray-panel
chmod +x /usr/local/bin/xray-panel

# 启动服务
systemctl start xray-panel
```

## 卸载

### 使用卸载脚本

```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/main/scripts/uninstall.sh)
```

### 手动卸载

```bash
# 停止服务
systemctl stop xray-panel
systemctl disable xray-panel

# 删除文件
rm -f /usr/local/bin/xray-panel
rm -f /etc/systemd/system/xray-panel.service
rm -rf /opt/xray-panel

# 删除配置和数据（可选）
rm -rf /etc/xray-panel
rm -rf /var/lib/xray-panel
rm -rf /var/log/xray-panel

# 删除 Nginx 配置
rm -f /etc/nginx/conf.d/xray-panel.conf

# 重载服务
systemctl daemon-reload
systemctl reload nginx
```

## 故障排查

### Panel 无法启动

```bash
# 查看详细日志
journalctl -u xray-panel -n 100 --no-pager

# 检查配置文件
xray-panel -config /etc/xray-panel/conf/config.yaml

# 检查端口占用
netstat -tlnp | grep 8082
```

### 无法访问 Web 界面

```bash
# 检查 Nginx 状态
systemctl status nginx

# 测试 Nginx 配置
nginx -t

# 检查防火墙
ufw status
iptables -L -n
```

### 数据库错误

```bash
# 检查数据库文件权限
ls -la /var/lib/xray-panel/panel.db

# 重置数据库（会丢失所有数据）
systemctl stop xray-panel
rm -f /var/lib/xray-panel/panel.db
systemctl start xray-panel
```

## 常见问题

### Q: 忘记管理员密码怎么办？

```bash
xray-panel -reset-password -username=admin_xxx -password=NewPassword123!
```

### Q: 如何更改监听端口？

编辑 `/etc/xray-panel/conf/config.yaml`，修改 `server.listen`，然后重启服务：

```bash
systemctl restart xray-panel
```

### Q: 如何启用调试模式？

编辑配置文件，设置 `server.debug: true` 和 `log.level: "debug"`，然后重启服务。

### Q: 如何备份数据？

```bash
# 备份数据库
cp /var/lib/xray-panel/panel.db /root/panel.db.backup

# 备份配置
cp /etc/xray-panel/conf/config.yaml /root/config.yaml.backup
```

## 安全建议

1. ✅ 立即修改默认管理员密码
2. ✅ 启用 HTTPS（使用 Let's Encrypt）
3. ✅ 配置防火墙，只开放必要端口
4. ✅ 定期备份数据库
5. ✅ 定期更新系统和软件
6. ✅ 使用强密码和 JWT Secret
7. ✅ 限制管理面板访问 IP（可选）

## 性能优化

### 系统优化

```bash
# 增加文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 优化内核参数
cat >> /etc/sysctl.conf <<EOF
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
net.ipv4.tcp_fastopen=3
EOF

sysctl -p
```

### Nginx 优化

编辑 `/etc/nginx/nginx.conf`:

```nginx
worker_processes auto;
worker_rlimit_nofile 65536;

events {
    worker_connections 4096;
    use epoll;
}
```

## 参考资源

- [官方文档](https://github.com/yourusername/xray-panel)
- [Xray-core 文档](https://xtls.github.io/)
- [Nginx 文档](https://nginx.org/en/docs/)
- [问题反馈](https://github.com/yourusername/xray-panel/issues)
