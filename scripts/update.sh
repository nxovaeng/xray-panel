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
CONFIG_DIR="/etc/xray-panel"
DATA_DIR="/var/lib/xray-panel"
LOG_DIR="/var/log/xray-panel"

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
    if command -v xray-panel &> /dev/null; then
        CURRENT_VERSION=$(xray-panel -version 2>/dev/null | grep -oP 'version \K[^ ]+' || echo "unknown")
        log_info "Current version: $CURRENT_VERSION"
    else
        log_warning "Panel not installed"
        CURRENT_VERSION="not_installed"
    fi
}

get_latest_version() {
    log_info "Checking for latest version..."
    LATEST_VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep -oP '"tag_name": "\K(.*)(?=")')
    
    if [[ -z "$LATEST_VERSION" ]]; then
        log_error "Failed to get latest version"
        exit 1
    fi
    
    log_info "Latest version: $LATEST_VERSION"
}

backup_current() {
    log_info "Creating backup..."
    
    BACKUP_DIR="/root/xray-panel-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # Backup binary
    if [[ -f /usr/local/bin/xray-panel ]]; then
        cp /usr/local/bin/xray-panel "$BACKUP_DIR/"
    fi
    
    # Backup database
    if [[ -f "$DATA_DIR/panel.db" ]]; then
        cp "$DATA_DIR/panel.db" "$BACKUP_DIR/"
    fi
    
    # Backup config
    if [[ -f "$CONFIG_DIR/conf/config.yaml" ]]; then
        cp "$CONFIG_DIR/conf/config.yaml" "$BACKUP_DIR/"
    fi
    
    log_success "Backup created at: $BACKUP_DIR"
}

stop_panel() {
    log_info "Stopping panel service..."
    
    if systemctl is-active --quiet xray-panel; then
        systemctl stop xray-panel
        log_success "Panel service stopped"
    fi
}

download_new_version() {
    log_info "Downloading new version..."
    
    # Get the actual version tag if using "latest"
    if [[ "$PANEL_VERSION" == "latest" ]]; then
        log_info "Fetching latest version..."
        ACTUAL_VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep -oP '"tag_name": "\K(.*)(?=")')
        if [[ -z "$ACTUAL_VERSION" ]]; then
            log_error "Failed to get latest version"
            exit 1
        fi
        log_info "Latest version: $ACTUAL_VERSION"
    else
        ACTUAL_VERSION="$PANEL_VERSION"
    fi
    
    # Construct download URL with version in filename
    DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/${ACTUAL_VERSION}/xray-panel-${ACTUAL_VERSION}-linux-${ARCH}.tar.gz"
    
    log_info "Downloading from: $DOWNLOAD_URL"
    
    # Download and extract
    cd /tmp
    if wget -q --show-progress "$DOWNLOAD_URL" -O xray-panel-new.tar.gz; then
        tar xzf xray-panel-new.tar.gz
        
        # Find the binary
        BINARY=$(find . -name "panel-linux-${ARCH}" -type f | head -n 1)
        
        if [[ -n "$BINARY" ]]; then
            chmod +x "$BINARY"
            mv "$BINARY" /usr/local/bin/xray-panel
            log_success "New version installed"
        else
            log_error "Binary not found in archive"
            exit 1
        fi
        
        rm -f xray-panel-new.tar.gz
    else
        log_error "Failed to download new version"
        exit 1
    fi
}

start_panel() {
    log_info "Starting panel service..."
    
    systemctl start xray-panel
    
    # Wait for service to start
    sleep 2
    
    if systemctl is-active --quiet xray-panel; then
        log_success "Panel service started"
    else
        log_error "Failed to start panel service"
        log_info "Check logs: journalctl -u xray-panel -n 50"
        exit 1
    fi
}

verify_update() {
    log_info "Verifying update..."
    
    NEW_VERSION=$(xray-panel -version 2>/dev/null | grep -oP 'version \K[^ ]+' || echo "unknown")
    
    if [[ "$NEW_VERSION" != "$CURRENT_VERSION" ]]; then
        log_success "Update successful!"
        log_info "Old version: $CURRENT_VERSION"
        log_info "New version: $NEW_VERSION"
    else
        log_warning "Version unchanged"
    fi
}

show_complete() {
    echo ""
    echo -e "${GREEN}========================================${PLAIN}"
    echo -e "${GREEN}  Update Complete!${PLAIN}"
    echo -e "${GREEN}========================================${PLAIN}"
    echo ""
    echo -e "${CYAN}Panel Status:${PLAIN}"
    systemctl status xray-panel --no-pager -l
    echo ""
    echo -e "${YELLOW}Useful Commands:${PLAIN}"
    echo -e "  View logs:     ${GREEN}journalctl -u xray-panel -f${PLAIN}"
    echo -e "  Restart panel: ${GREEN}systemctl restart xray-panel${PLAIN}"
    echo -e "  Panel status:  ${GREEN}systemctl status xray-panel${PLAIN}"
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
