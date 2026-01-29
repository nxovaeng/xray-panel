#!/bin/bash

# Xray Panel 反向代理快速配置脚本
# 用途: 自动配置 Nginx 反向代理和 SSL 证书

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查是否为 root
if [ "$EUID" -ne 0 ]; then 
    print_error "请使用 root 权限运行此脚本"
    exit 1
fi

echo "=================================="
echo "  Xray Panel 反向代理配置向导"
echo "=================================="
echo ""

# 1. 检查 Nginx
print_info "检查 Nginx..."
if ! command -v nginx &> /dev/null; then
    print_error "Nginx 未安装，请先安装 Nginx"
    exit 1
fi
print_info "✓ Nginx 已安装: $(nginx -v 2>&1 | grep -oP 'nginx/\K[0-9.]+')"

# 2. 检查 Certbot
print_info "检查 Certbot..."
if ! command -v certbot &> /dev/null; then
    print_warn "Certbot 未安装，正在安装..."
    if [ -f /etc/debian_version ]; then
        apt update && apt install -y certbot
    elif [ -f /etc/redhat-release ]; then
        yum install -y certbot
    else
        print_error "不支持的系统，请手动安装 Certbot"
        exit 1
    fi
fi
print_info "✓ Certbot 已安装"

# 3. 获取配置信息
echo ""
print_info "请输入配置信息:"
echo ""

read -p "面板域名 (如 panel.example.com): " PANEL_DOMAIN
if [ -z "$PANEL_DOMAIN" ]; then
    print_error "域名不能为空"
    exit 1
fi

read -p "面板监听地址 [127.0.0.1:8080]: " PANEL_LISTEN
PANEL_LISTEN=${PANEL_LISTEN:-127.0.0.1:8080}

read -p "是否配置 IP 白名单? (y/n) [n]: " SETUP_WHITELIST
SETUP_WHITELIST=${SETUP_WHITELIST:-n}

WHITELIST_IPS=""
if [ "$SETUP_WHITELIST" = "y" ]; then
    read -p "请输入允许访问的 IP (多个 IP 用空格分隔): " WHITELIST_IPS
fi

read -p "是否配置 HTTP 基本认证? (y/n) [n]: " SETUP_AUTH
SETUP_AUTH=${SETUP_AUTH:-n}

AUTH_USER=""
if [ "$SETUP_AUTH" = "y" ]; then
    read -p "认证用户名: " AUTH_USER
    if [ -z "$AUTH_USER" ]; then
        print_error "用户名不能为空"
        exit 1
    fi
fi

# 4. 确认配置
echo ""
print_info "配置信息确认:"
echo "  域名: $PANEL_DOMAIN"
echo "  Panel 地址: $PANEL_LISTEN"
echo "  IP 白名单: $([ "$SETUP_WHITELIST" = "y" ] && echo "是 ($WHITELIST_IPS)" || echo "否")"
echo "  HTTP 认证: $([ "$SETUP_AUTH" = "y" ] && echo "是 (用户: $AUTH_USER)" || echo "否")"
echo ""
read -p "确认配置? (y/n): " CONFIRM
if [ "$CONFIRM" != "y" ]; then
    print_info "已取消"
    exit 0
fi

# 5. 申请 SSL 证书
echo ""
print_info "申请 SSL 证书..."

# 停止 Nginx（如果运行）
if systemctl is-active --quiet nginx; then
    print_info "停止 Nginx..."
    systemctl stop nginx
fi

# 申请证书
if certbot certonly --standalone -d "$PANEL_DOMAIN" --non-interactive --agree-tos --register-unsafely-without-email; then
    print_info "✓ 证书申请成功"
else
    print_error "证书申请失败"
    systemctl start nginx
    exit 1
fi

# 6. 创建 Nginx 配置
print_info "创建 Nginx 配置..."

NGINX_CONF="/etc/nginx/conf.d/${PANEL_DOMAIN}.conf"

cat > "$NGINX_CONF" << EOF
# Xray Panel 反向代理配置
# 自动生成于: $(date)

# HTTP 重定向到 HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name ${PANEL_DOMAIN};
    return 301 https://\$server_name\$request_uri;
}

# HTTPS 服务器
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ${PANEL_DOMAIN};
    
    # SSL 证书
    ssl_certificate /etc/letsencrypt/live/${PANEL_DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${PANEL_DOMAIN}/privkey.pem;
    
    # SSL 安全配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # 安全头
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
EOF

# 添加 IP 白名单
if [ "$SETUP_WHITELIST" = "y" ]; then
    cat >> "$NGINX_CONF" << EOF
    
    # IP 白名单
EOF
    for ip in $WHITELIST_IPS; do
        echo "    allow $ip;" >> "$NGINX_CONF"
    done
    echo "    deny all;" >> "$NGINX_CONF"
fi

# 添加 HTTP 基本认证
if [ "$SETUP_AUTH" = "y" ]; then
    cat >> "$NGINX_CONF" << EOF
    
    # HTTP 基本认证
    auth_basic "Xray Panel Access";
    auth_basic_user_file /etc/nginx/.htpasswd;
EOF
fi

# 添加其余配置
cat >> "$NGINX_CONF" << EOF
    
    # 日志
    access_log /var/log/nginx/panel.access.log;
    error_log /var/log/nginx/panel.error.log;
    
    # 反向代理
    location / {
        proxy_pass http://${PANEL_LISTEN};
        proxy_http_version 1.1;
        
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        proxy_buffering off;
    }
    
    # 静态文件缓存
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|svg|woff|woff2|ttf|eot)$ {
        proxy_pass http://${PANEL_LISTEN};
        expires 7d;
        add_header Cache-Control "public, immutable";
    }
}
EOF

print_info "✓ Nginx 配置已创建: $NGINX_CONF"

# 7. 创建 HTTP 基本认证
if [ "$SETUP_AUTH" = "y" ]; then
    print_info "配置 HTTP 基本认证..."
    
    # 安装 htpasswd
    if ! command -v htpasswd &> /dev/null; then
        if [ -f /etc/debian_version ]; then
            apt install -y apache2-utils
        elif [ -f /etc/redhat-release ]; then
            yum install -y httpd-tools
        fi
    fi
    
    # 创建密码文件
    htpasswd -c /etc/nginx/.htpasswd "$AUTH_USER"
    print_info "✓ HTTP 基本认证已配置"
fi

# 8. 测试 Nginx 配置
print_info "测试 Nginx 配置..."
if nginx -t; then
    print_info "✓ Nginx 配置测试通过"
else
    print_error "Nginx 配置测试失败"
    exit 1
fi

# 9. 启动 Nginx
print_info "启动 Nginx..."
systemctl start nginx
systemctl enable nginx
print_info "✓ Nginx 已启动"

# 10. 配置防火墙
print_info "配置防火墙..."
if command -v ufw &> /dev/null; then
    ufw allow 80/tcp
    ufw allow 443/tcp
    print_info "✓ 防火墙已配置 (ufw)"
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-service=http
    firewall-cmd --permanent --add-service=https
    firewall-cmd --reload
    print_info "✓ 防火墙已配置 (firewalld)"
else
    print_warn "未检测到防火墙，请手动开放 80 和 443 端口"
fi

# 11. 配置证书自动续期
print_info "配置证书自动续期..."
systemctl enable certbot.timer
systemctl start certbot.timer

# 创建续期钩子
mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh << 'EOF'
#!/bin/bash
nginx -s reload
EOF
chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh

print_info "✓ 证书自动续期已配置"

# 12. 完成
echo ""
echo "=================================="
print_info "配置完成！"
echo "=================================="
echo ""
echo "访问地址: https://${PANEL_DOMAIN}"
echo ""
echo "下一步:"
echo "  1. 确保 DNS 已正确配置"
echo "  2. 修改 Panel 配置为本地监听: ${PANEL_LISTEN}"
echo "  3. 重启 Panel 服务"
echo "  4. 访问 https://${PANEL_DOMAIN} 测试"
echo ""
echo "查看日志:"
echo "  访问日志: tail -f /var/log/nginx/panel.access.log"
echo "  错误日志: tail -f /var/log/nginx/panel.error.log"
echo ""
echo "证书续期:"
echo "  测试续期: certbot renew --dry-run"
echo "  手动续期: certbot renew"
echo ""

if [ "$SETUP_WHITELIST" = "y" ]; then
    print_warn "已配置 IP 白名单，只有以下 IP 可以访问:"
    for ip in $WHITELIST_IPS; do
        echo "  - $ip"
    done
    echo ""
fi

if [ "$SETUP_AUTH" = "y" ]; then
    print_warn "已配置 HTTP 基本认证，访问时需要输入用户名和密码"
    echo ""
fi

print_info "详细文档: PANEL_REVERSE_PROXY.md"
