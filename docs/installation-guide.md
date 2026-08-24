# Xray Panel 安装指南

## 目录结构

安装后，所有文件统一存放在 `/opt/xray-panel` 目录下：

```
/opt/xray-panel/
├── panel              # 主程序二进制文件
├── conf/              # 配置文件目录
│   └── config.yaml    # 主配置文件
├── data/              # 数据目录
│   └── panel.db       # SQLite 数据库
├── logs/              # 日志目录
│   └── panel.log      # 面板日志
└── web/               # Web 静态文件 (可选)
    ├── static/
    └── templates/
```

## 安装方式

### 方式一：在线安装（推荐）

适用于有网络连接的服务器，直接从 GitHub 下载最新版本。

```bash
# 安装最新版本
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install.sh) install

# 安装指定版本
PANEL_VERSION="v1.0.0" bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install.sh) install

# 从自定义仓库安装
GITHUB_REPO="username/repo" bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install.sh) install
```

### 方式二：本地安装

适用于无网络或网络受限的服务器。

1. **下载 Release 包**

   访问 [Releases 页面](https://github.com/yourusername/xray-panel/releases) 下载对应系统的压缩包：
   
   - Linux AMD64: `xray-panel-vX.X.X-linux-amd64.tar.gz`
   - Linux ARM64: `xray-panel-vX.X.X-linux-arm64.tar.gz`

2. **上传到服务器**

   ```bash
   scp xray-panel-v1.0.0-linux-amd64.tar.gz root@your-server:/root/
   ```

3. **解压并安装**

   ```bash
   # 解压
   tar xzf xray-panel-v1.0.0-linux-amd64.tar.gz
   cd xray-panel-v1.0.0-linux-amd64
   
   # 运行安装脚本
   bash scripts/install.sh install
   ```

## 安装过程说明

安装脚本会自动完成以下操作：

1. ✅ 检测操作系统和架构
2. ✅ 安装必要依赖（Nginx、SQLite、Certbot）
3. ✅ 安装 Xray-core
4. ✅ 下载/复制面板程序到 `/opt/xray-panel`
5. ✅ 生成配置文件
6. ✅ 配置 Nginx（添加 stream 支持）
7. ✅ 创建 systemd 服务
8. ✅ 安装管理脚本 `xray-panel`

**注意：安装脚本不会自动启动服务**，这样可以让你先检查配置。

## 首次启动

### 1. 检查配置文件

```bash
# 查看配置文件
cat /opt/xray-panel/conf/config.yaml

# 根据需要修改配置
nano /opt/xray-panel/conf/config.yaml
```

主要配置项：

- `server.listen`: 监听地址（默认 `127.0.0.1:8082`）
- `database.path`: 数据库路径
- `xray.binary_path`: Xray 可执行文件路径
- `nginx.config_dir`: Nginx 配置目录

### 2. 启动服务

```bash
# 启动面板
systemctl start xray-panel

# 查看状态
systemctl status xray-panel

# 查看日志
journalctl -u xray-panel -f
```

### 3. 查看管理员账户

首次启动会自动生成管理员账户：

```bash
cd /opt/xray-panel
./panel admin
```

输出示例：

```
========================================
📋 管理员账户信息
========================================

账户 #1:
  用户名:   admin_k7m2p9
  创建时间: 2024-01-30 10:30:00
  更新时间: 2024-01-30 10:30:00

========================================
💡 提示:
  - 如需重置密码，使用: ./panel reset-password -username=<用户名> -password=<新密码>
  - 密码已加密存储，无法直接查看
========================================
```

**重要：请立即保存管理员凭据！**

## 便捷管理脚本

安装完成后，可以使用 `xray-panel` 命令打开管理菜单：

```bash
xray-panel
```

管理菜单功能：

```
0.  退出脚本
————————————————
1.  安装 Xray Panel
2.  更新 Xray Panel
3.  卸载 Xray Panel
————————————————
4.  启动 Xray Panel
5.  停止 Xray Panel
6.  重启 Xray Panel
7.  查看 Xray Panel 状态
8.  查看 Xray Panel 日志
————————————————
9.  设置 Xray Panel 开机自启
10. 取消 Xray Panel 开机自启
————————————————
11. 重置管理员账户
12. 查看管理员信息
13. 修改面板端口
————————————————
14. 启动 Xray
15. 停止 Xray
16. 重启 Xray
17. 查看 Xray 状态
18. 查看 Xray 日志
————————————————
19. 配置 Nginx 反向代理
20. 申请 SSL 证书
21. 申请通配符证书
22. 续期 SSL 证书
————————————————
23. 备份数据
24. 恢复数据
25. 清理日志
```

## 配置 Nginx 反向代理

### 方式一：使用管理脚本（推荐）

```bash
xray-panel
# 选择 19. 配置 Nginx 反向代理
```

### 方式二：手动配置

创建 Nginx 配置文件：

```bash
nano /etc/nginx/conf.d/xray-panel.conf
```

添加以下内容：

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://127.0.0.1:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

重载 Nginx：

```bash
nginx -t && systemctl reload nginx
```

## SSL 证书申请

### 普通证书

使用管理脚本：

```bash
xray-panel
# 选择 20. 申请 SSL 证书
```

或手动执行：

```bash
certbot --nginx -d your-domain.com --email your@email.com --agree-tos
```

### 通配符证书

通配符证书需要 DNS 验证，支持以下方式：

#### 1. Cloudflare DNS

```bash
# 安装插件
apt-get install -y python3-certbot-dns-cloudflare

# 创建凭据文件
mkdir -p /root/.secrets
cat > /root/.secrets/cloudflare.ini <<EOF
dns_cloudflare_api_token = YOUR_API_TOKEN
EOF
chmod 600 /root/.secrets/cloudflare.ini

# 申请证书
certbot certonly --dns-cloudflare \
    --dns-cloudflare-credentials /root/.secrets/cloudflare.ini \
    -d example.com -d *.example.com \
    --email your@email.com --agree-tos
```

#### 2. 手动 DNS 验证

```bash
certbot certonly --manual --preferred-challenges dns \
    -d example.com -d *.example.com \
    --email your@email.com --agree-tos
```

按提示添加 DNS TXT 记录。

#### 3. 使用管理脚本

```bash
xray-panel
# 选择 21. 申请通配符证书
```

支持的 DNS 提供商：
- Cloudflare
- 阿里云（需要额外插件）
- 腾讯云（需要额外插件）
- 手动验证

### 证书续期

Certbot 会自动配置续期任务，也可以手动续期：

```bash
# 使用管理脚本
xray-panel
# 选择 22. 续期 SSL 证书

# 或手动执行
certbot renew
```

## 常用命令

### 面板管理

```bash
# 启动
systemctl start xray-panel

# 停止
systemctl stop xray-panel

# 重启
systemctl restart xray-panel

# 状态
systemctl status xray-panel

# 查看日志（实时）
journalctl -u xray-panel -f

# 查看最近 100 行日志
journalctl -u xray-panel -n 100

# 查看日志文件（普通用户可读）
cat /opt/xray-panel/logs/panel.stdout.log  # 标准输出
cat /opt/xray-panel/logs/panel.stderr.log  # 错误输出
tail -f /opt/xray-panel/logs/panel.stdout.log  # 实时查看

# 开机自启
systemctl enable xray-panel

# 取消自启
systemctl disable xray-panel
```

### 日志文件说明

日志文件位于 `/opt/xray-panel/logs/`，权限设置为所有用户可读（644），方便在 WinSCP 等工具中查看：

- `panel.stdout.log` - 标准输出日志
- `panel.stderr.log` - 错误日志
- `panel.log` - 应用程序日志（如果配置了）

**在 WinSCP 中查看：**
1. 连接到服务器（普通用户即可）
2. 导航到 `/opt/xray-panel/logs/`
3. 双击日志文件即可查看

### 管理员操作

```bash
# 查看管理员信息
cd /opt/xray-panel
./panel admin

# 重置密码
./panel reset-password -username=admin_xxx -password=NewPassword123

# 查看版本
./panel version
```

### 数据备份

```bash
# 使用管理脚本
xray-panel
# 选择 23. 备份数据

# 或手动备份
tar czf xray-panel-backup-$(date +%Y%m%d).tar.gz \
    /opt/xray-panel/data \
    /opt/xray-panel/conf
```

### 数据恢复

```bash
# 使用管理脚本
xray-panel
# 选择 24. 恢复数据

# 或手动恢复
systemctl stop xray-panel
tar xzf xray-panel-backup-20240130.tar.gz -C /
systemctl start xray-panel
```

## 更新面板

### 使用管理脚本

```bash
xray-panel
# 选择 2. 更新 Xray Panel
```

### 手动更新

```bash
# 更新到最新版本
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install.sh) update

# 更新到指定版本
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install.sh) update v1.0.0
```

更新过程会自动：
1. 备份当前版本
2. 停止服务
3. 下载新版本
4. 启动服务
5. 验证更新

## 卸载面板

### 使用管理脚本

```bash
xray-panel
# 选择 3. 卸载 Xray Panel
```

### 手动卸载

```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install.sh) uninstall
```

卸载过程：
1. 自动备份数据
2. 停止服务
3. 删除 systemd 服务
4. 删除程序文件
5. 询问是否删除数据

**注意：卸载不会删除 Xray-core 和 Nginx**

## 故障排查

### 服务无法启动

```bash
# 查看详细日志
journalctl -u xray-panel -n 100 --no-pager

# 检查配置文件
/opt/xray-panel/panel -config /opt/xray-panel/conf/config.yaml

# 检查端口占用
netstat -tlnp | grep 8082
```

### 无法访问面板

1. 检查服务状态：`systemctl status xray-panel`
2. 检查防火墙：`ufw status` 或 `firewall-cmd --list-all`
3. 检查 Nginx 配置：`nginx -t`
4. 查看 Nginx 日志：`tail -f /var/log/nginx/error.log`

### 忘记管理员密码

```bash
cd /opt/xray-panel
./panel reset-password -username=admin_xxx -password=NewPassword123
```

### 数据库损坏

```bash
# 恢复备份
systemctl stop xray-panel
cp /root/xray-panel-backup-xxx/panel.db /opt/xray-panel/data/
systemctl start xray-panel
```

## 安全建议

1. ✅ 使用强密码
2. ✅ 启用 HTTPS
3. ✅ 配置防火墙
4. ✅ 定期备份数据
5. ✅ 定期更新系统和面板
6. ✅ 限制 SSH 访问
7. ✅ 使用非标准端口（可选）

## 支持的系统

- Ubuntu 20.04+
- Debian 10+
- CentOS 8+
- Rocky Linux 8+
- AlmaLinux 8+

## 系统要求

- 内存：≥ 512MB
- 磁盘：≥ 1GB
- 架构：AMD64 或 ARM64

## 相关文档

- [配置文件说明](configuration.md)
- [CLI 命令使用](cli-commands.md)
