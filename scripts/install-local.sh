#!/bin/bash

# Xray Panel 本地安装脚本
# 从本地压缩包安装
# Version: 1.0.0
#
# 使用方法:
#   1. 下载 release 压缩包到服务器
#   2. 解压: tar xzf xray-panel-vX.X.X-linux-amd64.tar.gz
#   3. 进入目录: cd xray-panel-vX.X.X-linux-amd64
#   4. 运行安装: bash scripts/install-local.sh

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PLAIN='\033[0m'

# 配置
INSTALL_DIR="/opt/xray-panel"
CONFIG_DIR="${INSTALL_DIR}/conf"
DATA_DIR="${INSTALL_DIR}/data"
LOG_DIR="${INSTALL_DIR}/logs"
SYSTEMD_SERVICE="/etc/systemd/system/xray-panel.service"

# 当前目录
CURRENT_DIR=$(pwd)

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

# 检查本地文件
check_local_files() {
    log_info "检查本地文件..."
    
    # 查找二进制文件
    BINARY=$(find "$CURRENT_DIR" -name "panel-linux-${ARCH}" -o -name "panel" | head -n 1)
    
    if [[ -z "$BINARY" ]]; then
        log_error "未找到面板二进制文件"
        log_error "请确保在解压后的目录中运行此脚本"
        exit 1
    fi
    
    log_success "找到二进制文件: $BINARY"
}

# 安装依赖
install_dependencies() {
    log_info "安装依赖..."
    
    case $OS in
        ubuntu|debian)
            apt-get update
            apt-get install -y curl wget nginx sqlite3 certbot python3-certbot-nginx
            ;;
        centos|rhel|rocky|almalinux)
            yum install -y epel-release
            yum install -y curl wget nginx sqlite certbot python3-certbot-nginx
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

# 安装面板
install_panel() {
    log_info "安装 Xray Panel..."
    
    # 创建目录
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"
    
    # 复制二进制文件
    cp "$BINARY" "$INSTALL_DIR/panel"
    chmod +x "$INSTALL_DIR/panel"
    
    # 复制配置文件示例（如果存在）
    if [[ -d "$CURRENT_DIR/conf" ]]; then
        cp -r "$CURRENT_DIR/conf"/* "$CONFIG_DIR/" 2>/dev/null || true
    fi
    
    # 复制 web 文件（如果存在）
    if [[ -d "$CURRENT_DIR/web" ]]; then
        cp -r "$CURRENT_DIR/web" "$INSTALL_DIR/" 2>/dev/null || true
    fi
    
    log_success "面板安装完成"
}

# 生成配置文件
generate_config() {
    log_info "生成配置文件..."
    
    # 如果配置文件已存在，跳过
    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        log_warning "配置文件已存在，跳过生成"
        return
    fi
    
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
Documentation=https://github.com/nxovaeng/xray-panel
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

# 安装管理脚本
install_management_script() {
    log_info "安装管理脚本..."
    
    # 如果本地有管理脚本，复制它
    if [[ -f "$CURRENT_DIR/xray-panel.sh" ]]; then
        cp "$CURRENT_DIR/xray-panel.sh" /usr/local/bin/xray-panel.sh
        chmod +x /usr/local/bin/xray-panel.sh
        ln -sf /usr/local/bin/xray-panel.sh /usr/bin/xray-panel
        log_success "管理脚本已安装"
    else
        log_warning "未找到管理脚本，可从 GitHub 下载"
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
    echo -e "${CYAN}本地安装脚本 v1.0.0${PLAIN}"
    echo ""
    
    log_info "开始安装..."
    
    check_root
    detect_os
    detect_arch
    check_local_files
    install_dependencies
    install_xray
    install_panel
    generate_config
    configure_nginx
    create_systemd_service
    install_management_script
    
    show_complete
}

# 运行主程序
main
