#!/bin/bash

# Xray Panel Update Script
# Author: Xray Panel Team
# Version: 1.0.0
#
# Usage:
#   Update to latest:
#     bash update.sh
#
#   Update to specific version:
#     bash update.sh v1.0.0
#
#   Custom repository:
#     GITHUB_REPO="username/repo" bash update.sh
#     GITHUB_REPO="username/repo" bash update.sh v1.0.0

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PLAIN='\033[0m'

# Configuration
INSTALL_DIR="/opt/xray-panel"
CONFIG_DIR="${INSTALL_DIR}/conf"
DATA_DIR="${INSTALL_DIR}/data"
LOG_DIR="${INSTALL_DIR}/logs"
BINARY_PATH="${INSTALL_DIR}/panel"

# Detect GitHub repository
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
PANEL_VERSION="${1:-latest}"

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
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
}

get_current_version() {
    if [[ -f "$BINARY_PATH" ]]; then
        CURRENT_VERSION=$("$BINARY_PATH" -version 2>/dev/null | grep -oP 'version \K[^ ]+' || echo "unknown")
        log_info "当前版本: $CURRENT_VERSION"
    else
        log_warning "面板未安装"
        CURRENT_VERSION="not_installed"
    fi
}

get_latest_version() {
    log_info "Checking for latest version..."
    LATEST_VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep -oP '"tag_name": "\K(.*)(?=")')
    
    if [[ -z "$LATEST_VERSION" ]]; then
        log_error "Failed to get latest version from GitHub API"
        log_error "Repository: $GITHUB_REPO"
        log_error "Please check your internet connection or try again later"
        exit 1
    fi
    
    log_info "Latest version: $LATEST_VERSION"
}

backup_current() {
    log_info "创建备份..."
    
    BACKUP_DIR="/root/xray-panel-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # 备份二进制文件
    if [[ -f "$BINARY_PATH" ]]; then
        cp "$BINARY_PATH" "$BACKUP_DIR/"
    fi
    
    # 备份数据库
    if [[ -f "$DATA_DIR/panel.db" ]]; then
        cp "$DATA_DIR/panel.db" "$BACKUP_DIR/"
    fi
    
    # 备份配置
    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        cp "$CONFIG_DIR/config.yaml" "$BACKUP_DIR/"
    fi
    
    log_success "备份已创建: $BACKUP_DIR"
}

# 安装依赖
install_dependencies() {
    log_info "检查依赖..."
    
    # 检测并安装缺失的依赖
    local missing_deps=()
    local deps=("curl" "wget" "unzip" "tar" "nginx" "sqlite3" "certbot")
    
    if [[ -f /etc/debian_version ]]; then
        deps+=("python3-certbot-nginx")
    elif [[ -f /etc/redhat-release ]]; then
        deps+=("python3-certbot-nginx")
    fi
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            missing_deps+=("$dep")
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_info "安装缺失的依赖: ${missing_deps[*]}"
        
        if [[ -f /usr/bin/apt-get ]]; then
            apt-get update
            apt-get install -y "${missing_deps[@]}"
        elif [[ -f /usr/bin/yum ]]; then
            yum install -y epel-release
            yum install -y "${missing_deps[@]}"
        else
            log_error "无法自动安装依赖，请手动安装: ${missing_deps[*]}"
            exit 1
        fi
    else
        log_success "所有依赖已安装"
    fi
}
    
stop_panel() {
    log_info "停止面板服务..."
    
    if systemctl is-active --quiet xray-panel; then
        systemctl stop xray-panel
        log_success "面板服务已停止"
    fi
}

download_new_version() {
    log_info "下载新版本..."
    
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
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    if wget -q --show-progress "$DOWNLOAD_URL" -O xray-panel-new.tar.gz; then
        tar xzf xray-panel-new.tar.gz
        
        # 查找解压后的目录（处理可能嵌套的结构）
        EXTRACTED_DIR=$(find . -mindepth 1 -maxdepth 1 -type d | head -n 1)
        if [[ -z "$EXTRACTED_DIR" ]]; then
            EXTRACTED_DIR="."
        fi
        
        # 查找二进制文件
        BINARY=$(find "$EXTRACTED_DIR" -name "panel-linux-${ARCH}" -o -name "panel" | head -n 1)
        
        if [[ -n "$BINARY" ]]; then
            chmod +x "$BINARY"
            mv "$BINARY" "$BINARY_PATH"
            
            # 更新 Web 资源（如果存在）
            if [[ -d "$EXTRACTED_DIR/web" ]]; then
                log_info "更新 Web 资源..."
                rm -rf "$INSTALL_DIR/web"
                cp -r "$EXTRACTED_DIR/web" "$INSTALL_DIR/"
            fi
            
            # 更新配置示例
            if [[ -d "$EXTRACTED_DIR/conf" ]]; then
                cp "$EXTRACTED_DIR/conf/config.yaml.example" "$CONFIG_DIR/" 2>/dev/null || true
            fi
            
            # 更新管理脚本
            if [[ -f "$EXTRACTED_DIR/xray-panel.sh" ]]; then
                log_info "更新管理脚本..."
                cp "$EXTRACTED_DIR/xray-panel.sh" "$INSTALL_DIR/"
                chmod +x "$INSTALL_DIR/xray-panel.sh"
                ln -sf "$INSTALL_DIR/xray-panel.sh" /usr/local/bin/xray-panel.sh
                chmod +x /usr/local/bin/xray-panel.sh
                ln -sf /usr/local/bin/xray-panel.sh /usr/bin/xray-panel
            fi
            
            log_success "新版本已安装"
        else
            log_error "未找到二进制文件"
            rm -rf "$TEMP_DIR"
            exit 1
        fi
        
        rm -rf "$TEMP_DIR"
    else
        log_error "下载失败"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
}

start_panel() {
    log_info "启动面板服务..."
    
    systemctl start xray-panel
    
    # 等待服务启动
    sleep 2
    
    if systemctl is-active --quiet xray-panel; then
        log_success "面板服务已启动"
    else
        log_error "面板服务启动失败"
        log_info "查看日志: journalctl -u xray-panel -n 50"
        exit 1
    fi
}

verify_update() {
    log_info "验证更新..."
    
    NEW_VERSION=$("$BINARY_PATH" -version 2>/dev/null | grep -oP 'version \K[^ ]+' || echo "unknown")
    
    if [[ "$NEW_VERSION" != "$CURRENT_VERSION" ]]; then
        log_success "更新成功！"
        log_info "旧版本: $CURRENT_VERSION"
        log_info "新版本: $NEW_VERSION"
    else
        log_warning "版本未变化"
    fi
}

show_complete() {
    echo ""
    echo -e "${GREEN}========================================${PLAIN}"
    echo -e "${GREEN}  更新完成！${PLAIN}"
    echo -e "${GREEN}========================================${PLAIN}"
    echo ""
    echo -e "${CYAN}面板状态:${PLAIN}"
    systemctl status xray-panel --no-pager -l
    echo ""
    echo -e "${YELLOW}常用命令:${PLAIN}"
    echo -e "  查看日志:     ${GREEN}journalctl -u xray-panel -f${PLAIN}"
    echo -e "  重启面板:     ${GREEN}systemctl restart xray-panel${PLAIN}"
    echo -e "  面板状态:     ${GREEN}systemctl status xray-panel${PLAIN}"
    echo -e "  管理脚本:     ${GREEN}xray-panel${PLAIN}"
    echo ""
}

main() {
    clear
    echo -e "${BLUE}"
    cat << "EOF"
 _   _           _       _       
| | | |_ __   __| | __ _| |_ ___ 
| | | | '_ \ / _` |/ _` | __/ _ \
| |_| | |_) | (_| | (_| | ||  __/
 \___/| .__/ \__,_|\__,_|\__\___|
      |_|                         
EOF
    echo -e "${PLAIN}"
    echo -e "${CYAN}Xray Panel Updater${PLAIN}"
    echo -e "${CYAN}Repository: $GITHUB_REPO${PLAIN}"
    echo ""
    
    check_root
    detect_arch
    get_current_version
    
    if [[ "$PANEL_VERSION" == "latest" ]]; then
        get_latest_version
        
        if [[ "$CURRENT_VERSION" == "$LATEST_VERSION" ]]; then
            log_info "Already running the latest version"
            read -p "Do you want to reinstall? (yes/no): " -r
            echo ""
            if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
                log_info "Update cancelled"
                exit 0
            fi
        fi
    fi
    
    backup_current
    stop_panel
    download_new_version
    start_panel
    verify_update
    show_complete
}

# Check if version argument is provided
if [[ $# -gt 1 ]]; then
    echo "Usage: $0 [version]"
    echo "Example: $0 v1.0.0"
    echo "         $0 latest (default)"
    exit 1
fi

main
