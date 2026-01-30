#!/bin/bash

# Xray Panel 在线安装脚本
# 从 GitHub 下载并安装
# Version: 1.0.0
#
# 使用方法:
#   bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install-online.sh)
#
# 自定义仓库:
#   GITHUB_REPO="username/repo" bash <(curl -Ls ...)
#
# 自定义版本:
#   PANEL_VERSION="v1.0.0" bash <(curl -Ls ...)

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PLAIN='\033[0m'

# 配置
PANEL_VERSION="${PANEL_VERSION:-latest}"
INSTALL_DIR="/opt/xray-panel"
CONFIG_DIR="${INSTALL_DIR}/conf"
DATA_DIR="${INSTALL_DIR}/data"
LOG_DIR="${INSTALL_DIR}/logs"
SYSTEMD_SERVICE="/etc/systemd/system/xray-panel.service"

# 检测 GitHub 仓库
detect_github_repo() {
    local repo=""
    
    # 1. Check environment variable
    if [[ -n "$GITHUB_REPO" ]]; then
        repo="$GITHUB_REPO"
    # 2. Try to detect from git remote
    elif command -v git &> /dev/null && git rev-parse --git-dir > /dev/null 2>&1; then
        repo=$(git config --get remote.origin.url 2>/dev/null | sed -E 's#.*github\.com[:/]([^/]+/[^/]+)(\.git)?$#\1#')
    fi
    
    # 3. Fallback to default
    if [[ -z "$repo" ]]; then
        repo="nxovaeng/xray-panel"
    fi
    
    echo "$repo"
}


GITHUB_REPO=$(detect_github_repo)

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${PLAIN} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${PLAIN} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${PLAIN} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${PLAIN} $1"
}

# 检查 root 权限
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "此脚本必须以 root 权限运行"
        exit 1
    fi
}

# 检测操作系统
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    else
        log_error "无法检测操作系统"
        exit 1
    fi
    
    log_info "检测到操作系统: $OS $OS_VERSION"
}

# 检测架构
detect_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            log_error "不支持的架构: $ARCH"
            exit 1
            ;;
    esac
    log_info "检测到架构: $ARCH"
}

# 安装依赖
install_dependencies() {
    log_info "安装依赖..."
    
    case $OS in
        ubuntu|debian)
            apt-get update
            apt-get install -y curl wget unzip tar nginx sqlite3 certbot python3-certbot-nginx
            ;;
        centos|rhel|rocky|almalinux)
            yum install -y epel-release
            yum install -y curl wget unzip tar nginx sqlite certbot python3-certbot-nginx
            ;;
        *)
            log_error "不支持的操作系统: $OS"
            exit 1
            ;;
    esac
    
    log_success "依赖安装完成"
}

# 安装 Xray
install_xray() {
    log_info "安装 Xray-core..."
    
    if command -v xray &> /dev/null; then
        log_warning "Xray 已安装"
        xray version
    else
        bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/master/install-release.sh)" @ install
        log_success "Xray-core 安装完成"
    fi
}

# 下载面板
download_panel() {
    log_info "下载 Xray Panel..."
    
    # 创建目录
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"
    
    # 获取版本
    if [[ "$PANEL_VERSION" == "latest" ]]; then
        log_info "获取最新版本..."
        ACTUAL_VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep -oP '"tag_name": "\K(.*)(?=")')
        if [[ -z "$ACTUAL_VERSION" ]]; then
            log_error "获取最新版本失败"
            exit 1
        fi
        log_info "最新版本: $ACTUAL_VERSION"
    else
        ACTUAL_VERSION="$PANEL_VERSION"
    fi
    
    # 构建下载 URL
    DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/${ACTUAL_VERSION}/xray-panel-${ACTUAL_VERSION}-linux-${ARCH}.tar.gz"
    
    log_info "下载地址: $DOWNLOAD_URL"
    
    # 下载并解压
    cd /tmp
    if wget -q --show-progress "$DOWNLOAD_URL" -O xray-panel.tar.gz; then
        tar xzf xray-panel.tar.gz -C "$INSTALL_DIR"
        
        # 查找二进制文件
        BINARY=$(find "$INSTALL_DIR" -name "panel-linux-${ARCH}" -o -name "panel" | head -n 1)
        
        if [[ -n "$BINARY" ]]; then
            mv "$BINARY" "$INSTALL_DIR/panel"
            chmod +x "$INSTALL_DIR/panel"
            log_success "面板下载完成"
        else
            log_error "未找到二进制文件"
            exit 1
        fi
        
        rm -f xray-panel.tar.gz
    else
        log_error "下载失败"
        log_info "请检查网络连接或使用本地安装脚本"
        exit 1
    fi
}

# 生成配置文件
generate_config() {
    log_info "生成配置文件..."
    
    # 生成随机 JWT Secret
    JWT_SECRET=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 32)
    
    cat > "$CONFIG_DIR/config.yaml" <<EOF
# Xray Panel 配置文件
# 安装时间: $(date '+%Y-%m-%d %H:%M:%S')

server:
  listen: "127.0.0.1:8082"
  debug: false

database:
  path: "$DATA_DIR/panel.db"

jwt:
  secret: "$JWT_SECRET"
  expire_hour: 168  # 7 天

admin:
  username: ""  # 首次启动自动生成
  password: ""  # 首次启动自动生成
  email: ""

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
  file: "$LOG_DIR/panel.log"
  max_size: 100
  max_backups: 7
  max_age: 30
  compress: true
EOF
    
    log_success "配置文件已生成: $CONFIG_DIR/config.yaml"
}

# 配置 Nginx
configure_nginx() {
    log_info "配置 Nginx..."
    
    # 测试配置
    if nginx -t; then
        log_success "Nginx 配置正确"
    else
        log_error "Nginx 配置测试失败"
        exit 1
    fi
}

# 创建 systemd 服务
create_systemd_service() {
    log_info "创建 systemd 服务..."
    
    # 设置日志目录权限（755 = rwxr-xr-x，所有用户可读）
    chmod 755 "$LOG_DIR"
    
    cat > "$SYSTEMD_SERVICE" <<EOF
[Unit]
Description=Xray Panel Service
Documentation=https://github.com/$GITHUB_REPO
After=network.target nss-lookup.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/panel -config $CONFIG_DIR/config.yaml
Restart=on-failure
RestartSec=10
LimitNOFILE=65536

# Logging
StandardOutput=append:$LOG_DIR/panel.stdout.log
StandardError=append:$LOG_DIR/panel.stderr.log

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR $LOG_DIR $CONFIG_DIR

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable xray-panel
    
    # 设置日志文件权限（644 = rw-r--r--，所有用户可读）
    touch "$LOG_DIR/panel.stdout.log" "$LOG_DIR/panel.stderr.log"
    chmod 644 "$LOG_DIR/panel.stdout.log" "$LOG_DIR/panel.stderr.log"
    
    log_success "Systemd 服务已创建并启用"
    log_info "日志文件权限已设置为所有用户可读"
}

# 下载管理脚本
download_management_script() {
    log_info "下载管理脚本..."
    
    SCRIPT_URL="https://raw.githubusercontent.com/$GITHUB_REPO/master/xray-panel.sh"
    
    if wget -q "$SCRIPT_URL" -O /usr/local/bin/xray-panel.sh; then
        chmod +x /usr/local/bin/xray-panel.sh
        ln -sf /usr/local/bin/xray-panel.sh /usr/bin/xray-panel
        log_success "管理脚本已安装"
    else
        log_warning "管理脚本下载失败，可手动下载"
    fi
}

# 显示完成信息
show_complete() {
    echo ""
    echo -e "${GREEN}========================================${PLAIN}"
    echo -e "${GREEN}  Xray Panel 安装完成！${PLAIN}"
    echo -e "${GREEN}========================================${PLAIN}"
    echo ""
    echo -e "${CYAN}安装目录:${PLAIN} $INSTALL_DIR"
    echo -e "${CYAN}配置文件:${PLAIN} $CONFIG_DIR/config.yaml"
    echo -e "${CYAN}数据目录:${PLAIN} $DATA_DIR"
    echo -e "${CYAN}日志目录:${PLAIN} $LOG_DIR"
    echo ""
    echo -e "${YELLOW}⚠️  重要提示:${PLAIN}"
    echo -e "  1. 服务已安装但${RED}未启动${PLAIN}"
    echo -e "  2. 请先检查配置文件: ${GREEN}$CONFIG_DIR/config.yaml${PLAIN}"
    echo -e "  3. 确认配置无误后启动服务: ${GREEN}systemctl start xray-panel${PLAIN}"
    echo -e "  4. 首次启动会自动生成管理员账户"
    echo -e "  5. 查看管理员信息: ${GREEN}cd $INSTALL_DIR && ./panel -show-admin${PLAIN}"
    echo ""
    echo -e "${CYAN}常用命令:${PLAIN}"
    echo -e "  启动面板:     ${GREEN}systemctl start xray-panel${PLAIN}"
    echo -e "  停止面板:     ${GREEN}systemctl stop xray-panel${PLAIN}"
    echo -e "  重启面板:     ${GREEN}systemctl restart xray-panel${PLAIN}"
    echo -e "  查看状态:     ${GREEN}systemctl status xray-panel${PLAIN}"
    echo -e "  查看日志:     ${GREEN}journalctl -u xray-panel -f${PLAIN}"
    echo -e "  管理脚本:     ${GREEN}xray-panel${PLAIN}"
    echo ""
    echo -e "${CYAN}下一步操作:${PLAIN}"
    echo -e "  1. 启动服务: ${GREEN}systemctl start xray-panel${PLAIN}"
    echo -e "  2. 查看管理员账户: ${GREEN}cd $INSTALL_DIR && ./panel -show-admin${PLAIN}"
    echo -e "  3. 配置 Nginx 反向代理 (可选)"
    echo -e "  4. 申请 SSL 证书 (推荐)"
    echo ""
    echo -e "${CYAN}使用管理脚本:${PLAIN}"
    echo -e "  运行 ${GREEN}xray-panel${PLAIN} 打开便捷管理菜单"
    echo ""
    echo -e "${GREEN}========================================${PLAIN}"
    echo ""
}

# 主安装流程
main() {
    clear
    echo -e "${BLUE}"
    cat << "EOF"
 __   __                   ____                  _ 
 \ \ / /_ __ __ _ _   _   |  _ \ __ _ _ __   ___| |
  \ V / '__/ _` | | | |  | |_) / _` | '_ \ / _ \ |
   | || | | (_| | |_| |  |  __/ (_| | | | |  __/ |
   |_||_|  \__,_|\__, |  |_|   \__,_|_| |_|\___|_|
                 |___/                             
EOF
    echo -e "${PLAIN}"
    echo -e "${CYAN}在线安装脚本 v1.0.0${PLAIN}"
    echo -e "${CYAN}仓库: https://github.com/$GITHUB_REPO${PLAIN}"
    echo ""
    
    log_info "开始安装..."
    log_info "使用仓库: $GITHUB_REPO"
    
    check_root
    detect_os
    detect_arch
    install_dependencies
    install_xray
    download_panel
    generate_config
    configure_nginx
    create_systemd_service
    download_management_script
    
    show_complete
}

# 运行主程序
main
