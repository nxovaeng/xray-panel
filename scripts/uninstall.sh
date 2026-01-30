#!/bin/bash

# Xray Panel Uninstallation Script
# Author: Xray Panel Team
# Version: 1.0.0

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PLAIN='\033[0m'

# Configuration
INSTALL_DIR="/opt/xray-panel"
CONFIG_DIR="${INSTALL_DIR}/conf"
DATA_DIR="${INSTALL_DIR}/data"
LOG_DIR="${INSTALL_DIR}/logs"
SYSTEMD_SERVICE="/etc/systemd/system/xray-panel.service"

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

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

confirm_uninstall() {
    echo ""
    echo -e "${RED}========================================${PLAIN}"
    echo -e "${RED}  WARNING: Uninstall Xray Panel${PLAIN}"
    echo -e "${RED}========================================${PLAIN}"
    echo ""
    echo -e "${YELLOW}This will remove:${PLAIN}"
    echo -e "  - Panel binary and files"
    echo -e "  - Systemd service"
    echo -e "  - Nginx configuration"
    echo ""
    echo -e "${YELLOW}This will NOT remove:${PLAIN}"
    echo -e "  - Database ($DATA_DIR)"
    echo -e "  - Configuration ($CONFIG_DIR)"
    echo -e "  - Logs ($LOG_DIR)"
    echo -e "  - Xray-core"
    echo -e "  - Nginx"
    echo ""
    read -p "Are you sure you want to continue? (yes/no): " -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_info "Uninstallation cancelled"
        exit 0
    fi
}

stop_services() {
    log_info "Stopping services..."
    
    if systemctl is-active --quiet xray-panel; then
        systemctl stop xray-panel
        log_success "Panel service stopped"
    fi
}

remove_systemd_service() {
    log_info "Removing systemd service..."
    
    if [[ -f "$SYSTEMD_SERVICE" ]]; then
        systemctl disable xray-panel 2>/dev/null || true
        rm -f "$SYSTEMD_SERVICE"
        systemctl daemon-reload
        log_success "Systemd service removed"
    fi
}

remove_binary() {
    log_info "移除面板文件..."
    
    # 移除管理脚本
    rm -f /usr/local/bin/xray-panel.sh
    rm -f /usr/bin/xray-panel
    
    log_success "面板文件已移除"
}

remove_nginx_config() {
    log_info "Removing Nginx configuration..."
    
    if [[ -f /etc/nginx/conf.d/xray-panel.conf ]]; then
        rm -f /etc/nginx/conf.d/xray-panel.conf
        log_success "Nginx panel configuration removed"
    fi
    
    # Reload nginx if running
    if systemctl is-active --quiet nginx; then
        nginx -t && systemctl reload nginx
        log_success "Nginx reloaded"
    fi
}

backup_data() {
    log_info "Creating backup..."
    
    BACKUP_DIR="/root/xray-panel-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # Backup database
    if [[ -f "$DATA_DIR/panel.db" ]]; then
        cp -r "$DATA_DIR" "$BACKUP_DIR/"
        log_success "Database backed up to: $BACKUP_DIR"
    fi
    
    # Backup configuration
    if [[ -d "$CONFIG_DIR" ]]; then
        cp -r "$CONFIG_DIR" "$BACKUP_DIR/"
        log_success "Configuration backed up to: $BACKUP_DIR"
    fi
    
    # Backup logs
    if [[ -d "$LOG_DIR" ]]; then
        cp -r "$LOG_DIR" "$BACKUP_DIR/"
        log_success "Logs backed up to: $BACKUP_DIR"
    fi
    
    echo ""
    echo -e "${GREEN}Backup created at: $BACKUP_DIR${PLAIN}"
    echo ""
}

clean_data() {
    read -p "是否删除所有数据和配置? (yes/no): " -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_warning "删除所有数据..."
        
        rm -rf "$INSTALL_DIR"
        
        log_success "所有数据已删除"
    else
        log_info "数据已保留在: $INSTALL_DIR"
        echo -e "  - 数据库: $DATA_DIR"
        echo -e "  - 配置: $CONFIG_DIR"
        echo -e "  - 日志: $LOG_DIR"
    fi
}

show_complete() {
    echo ""
    echo -e "${GREEN}========================================${PLAIN}"
    echo -e "${GREEN}  Uninstallation Complete!${PLAIN}"
    echo -e "${GREEN}========================================${PLAIN}"
    echo ""
    echo -e "${YELLOW}Remaining components:${PLAIN}"
    echo -e "  - Xray-core (use: bash -c \"\$(curl -L https://github.com/XTLS/Xray-install/raw/master/install-release.sh)\" @ remove)"
    echo -e "  - Nginx (use: apt remove nginx / yum remove nginx)"
    echo ""
}

main() {
    clear
    echo -e "${BLUE}"
    cat << "EOF"
 _   _       _           _        _ _ 
| | | |_ __ (_)_ __  ___| |_ __ _| | |
| | | | '_ \| | '_ \/ __| __/ _` | | |
| |_| | | | | | | | \__ \ || (_| | | |
 \___/|_| |_|_|_| |_|___/\__\__,_|_|_|
                                       
EOF
    echo -e "${PLAIN}"
    echo -e "${CYAN}Xray Panel Uninstaller${PLAIN}"
    echo ""
    
    check_root
    confirm_uninstall
    backup_data
    stop_services
    remove_systemd_service
    remove_binary
    remove_nginx_config
    clean_data
    show_complete
}

main
