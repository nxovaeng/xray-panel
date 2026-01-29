#!/bin/bash

# Xray Panel One-Click Installation Script
# Supports: Ubuntu 20.04+, Debian 10+, CentOS 8+
# Author: Xray Panel Team
# Version: 1.0.0
#
# Usage:
#   Default installation:
#     bash install.sh
#
#   Custom repository:
#     GITHUB_REPO="username/repo" bash install.sh
#
#   Custom version:
#     PANEL_VERSION="v1.0.0" bash install.sh
#
#   Both:
#     GITHUB_REPO="username/repo" PANEL_VERSION="v1.0.0" bash install.sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PLAIN='\033[0m'

# Configuration
PANEL_VERSION="${PANEL_VERSION:-latest}"
INSTALL_DIR="/opt/xray-panel"
CONFIG_DIR="/etc/xray-panel"
DATA_DIR="/var/lib/xray-panel"
LOG_DIR="/var/log/xray-panel"
SYSTEMD_SERVICE="/etc/systemd/system/xray-panel.service"

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

# Functions
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

detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    else
        log_error "Cannot detect OS"
        exit 1
    fi
    
    log_info "Detected OS: $OS $OS_VERSION"
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
    log_info "Detected architecture: $ARCH"
}

install_dependencies() {
    log_info "Installing dependencies..."
    
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
            log_error "Unsupported OS: $OS"
            exit 1
            ;;
    esac
    
    log_success "Dependencies installed"
}

install_xray() {
    log_info "Installing Xray-core..."
    
    if command -v xray &> /dev/null; then
        log_warning "Xray is already installed"
        xray version
    else
        bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install
        log_success "Xray-core installed"
    fi
}

download_panel() {
    log_info "Downloading Xray Panel..."
    
    # Create directories
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR/conf"
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"
    
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
    if wget -q --show-progress "$DOWNLOAD_URL" -O xray-panel.tar.gz; then
        tar xzf xray-panel.tar.gz -C "$INSTALL_DIR" --strip-components=1
        chmod +x "$INSTALL_DIR/panel-linux-${ARCH}"
        ln -sf "$INSTALL_DIR/panel-linux-${ARCH}" /usr/local/bin/xray-panel
        rm -f xray-panel.tar.gz
        log_success "Panel downloaded and installed"
    else
        log_error "Failed to download panel"
        log_info "Please check if the release exists or build from source"
        exit 1
    fi
}

configure_panel() {
    log_info "Configuring Xray Panel..."
    
    # Generate random JWT secret
    JWT_SECRET=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 32)
    
    # Create configuration file
    cat > "$CONFIG_DIR/conf/config.yaml" <<EOF
server:
  listen: "127.0.0.1:8082"
  debug: false

database:
  path: "$DATA_DIR/panel.db"

jwt:
  secret: "$JWT_SECRET"
  expire_hour: 168  # 7 days

admin:
  username: ""  # Auto-generate on first run
  password: ""  # Auto-generate on first run
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
    
    log_success "Configuration file created: $CONFIG_DIR/conf/config.yaml"
}

configure_nginx() {
    log_info "Configuring Nginx..."
    
    # Create stream directory
    mkdir -p /etc/nginx/stream.d
    
    # Backup nginx.conf
    if [[ ! -f /etc/nginx/nginx.conf.backup ]]; then
        cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.backup
    fi
    
    # Add stream support if not exists
    if ! grep -q "include.*stream.d/\*.conf" /etc/nginx/nginx.conf; then
        cat >> /etc/nginx/nginx.conf <<EOF

# Stream configuration for SNI routing
stream {
    include /etc/nginx/stream.d/*.conf;
}
EOF
        log_success "Nginx stream support added"
    else
        log_info "Nginx stream support already configured"
    fi
    
    # Create panel proxy configuration
    cat > /etc/nginx/conf.d/xray-panel.conf <<EOF
# Xray Panel Web Interface
server {
    listen 80;
    server_name _;
    
    location / {
        proxy_pass http://127.0.0.1:8082;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
EOF
    
    # Test nginx configuration
    if nginx -t; then
        systemctl reload nginx
        log_success "Nginx configured and reloaded"
    else
        log_error "Nginx configuration test failed"
        exit 1
    fi
}

create_systemd_service() {
    log_info "Creating systemd service..."
    
    cat > "$SYSTEMD_SERVICE" <<EOF
[Unit]
Description=Xray Panel Service
Documentation=https://github.com/$GITHUB_REPO
After=network.target nss-lookup.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=/usr/local/bin/xray-panel -config $CONFIG_DIR/conf/config.yaml
Restart=on-failure
RestartSec=10
LimitNOFILE=65536

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
    log_success "Systemd service created and enabled"
}

start_services() {
    log_info "Starting services..."
    
    # Start Xray
    systemctl enable xray
    systemctl restart xray
    log_success "Xray service started"
    
    # Start Panel
    systemctl restart xray-panel
    log_success "Panel service started"
    
    # Start Nginx
    systemctl enable nginx
    systemctl restart nginx
    log_success "Nginx service started"
}

show_info() {
    echo ""
    echo -e "${GREEN}========================================${PLAIN}"
    echo -e "${GREEN}  Xray Panel Installation Complete!${PLAIN}"
    echo -e "${GREEN}========================================${PLAIN}"
    echo ""
    echo -e "${CYAN}Installation Directory:${PLAIN} $INSTALL_DIR"
    echo -e "${CYAN}Configuration File:${PLAIN} $CONFIG_DIR/conf/config.yaml"
    echo -e "${CYAN}Database File:${PLAIN} $DATA_DIR/panel.db"
    echo -e "${CYAN}Log File:${PLAIN} $LOG_DIR/panel.log"
    echo ""
    echo -e "${CYAN}Web Interface:${PLAIN} http://$(curl -s ifconfig.me):80"
    echo -e "${CYAN}Panel Port:${PLAIN} 8082 (localhost only)"
    echo ""
    echo -e "${YELLOW}Admin Credentials:${PLAIN}"
    echo -e "  Run this command to view admin credentials:"
    echo -e "  ${GREEN}xray-panel -show-admin${PLAIN}"
    echo ""
    echo -e "${YELLOW}Useful Commands:${PLAIN}"
    echo -e "  View logs:        ${GREEN}journalctl -u xray-panel -f${PLAIN}"
    echo -e "  Restart panel:    ${GREEN}systemctl restart xray-panel${PLAIN}"
    echo -e "  Panel status:     ${GREEN}systemctl status xray-panel${PLAIN}"
    echo -e "  Reset password:   ${GREEN}xray-panel -reset-password -username=admin_xxx -password=NewPass${PLAIN}"
    echo ""
    echo -e "${YELLOW}Next Steps:${PLAIN}"
    echo -e "  1. View admin credentials: ${GREEN}xray-panel -show-admin${PLAIN}"
    echo -e "  2. Access web interface and login"
    echo -e "  3. Change admin password immediately"
    echo -e "  4. Configure SSL certificate (recommended)"
    echo -e "     ${GREEN}certbot --nginx -d yourdomain.com${PLAIN}"
    echo ""
    echo -e "${CYAN}Documentation:${PLAIN} https://github.com/$GITHUB_REPO"
    echo -e "${GREEN}========================================${PLAIN}"
    echo ""
}

# Main installation process
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
    echo -e "${CYAN}Version: 1.0.0${PLAIN}"
    echo -e "${CYAN}GitHub: https://github.com/$GITHUB_REPO${PLAIN}"
    echo ""
    
    log_info "Starting installation..."
    log_info "Using repository: $GITHUB_REPO"
    
    check_root
    detect_os
    detect_arch
    install_dependencies
    install_xray
    download_panel
    configure_panel
    configure_nginx
    create_systemd_service
    start_services
    
    # Wait for panel to start
    sleep 3
    
    show_info
}

# Run main function
main
