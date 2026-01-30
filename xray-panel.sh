#!/bin/bash

# Xray Panel 便捷管理脚本
# Version: 1.0.0

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PLAIN='\033[0m'

# 配置路径
INSTALL_DIR="/opt/xray-panel"
CONFIG_DIR="${INSTALL_DIR}/conf"
DATA_DIR="${INSTALL_DIR}/data"
LOG_DIR="${INSTALL_DIR}/logs"
BINARY_PATH="${INSTALL_DIR}/panel"
SYSTEMD_SERVICE="/etc/systemd/system/xray-panel.service"

# 检测 GitHub 仓库
detect_github_repo() {
    local repo=""
    
    # 1. 检查环境变量
    if [[ -n "$GITHUB_REPO" ]]; then
        repo="$GITHUB_REPO"
    # 2. 尝试从 git remote 检测
    elif command -v git &> /dev/null && git rev-parse --git-dir > /dev/null 2>&1; then
        repo=$(git config --get remote.origin.url 2>/dev/null | sed -E 's#.*github\.com[:/]([^/]+/[^/]+)(\.git)?$#\1#')
    fi
    
    # 3. 回退到默认值
    if [[ -z "$repo" ]]; then
        repo="nxovaeng/xray-panel"
    fi
    
    echo "$repo"
}

GITHUB_REPO=$(detect_github_repo)

# 检查 root 权限
check_root() {
    if [[ $EUID -ne 0 ]]; then
        echo -e "${RED}错误: 此脚本必须以 root 权限运行${PLAIN}"
        exit 1
    fi
}

# 检查是否已安装
check_installed() {
    if [[ ! -f "$BINARY_PATH" ]]; then
        echo -e "${RED}错误: Xray Panel 未安装${PLAIN}"
        echo -e "${YELLOW}请先运行安装脚本: bash <(curl -Ls https://raw.githubusercontent.com/$GITHUB_REPO/master/scripts/install-online.sh)${PLAIN}"
        exit 1
    fi
}

# 显示 Logo
show_logo() {
    clear
    echo -e "${CYAN}"
    cat << "EOF"
 __   __                   ____                  _ 
 \ \ / /_ __ __ _ _   _   |  _ \ __ _ _ __   ___| |
  \ V / '__/ _` | | | |  | |_) / _` | '_ \ / _ \ |
   | || | | (_| | |_| |  |  __/ (_| | | | |  __/ |
   |_||_|  \__,_|\__, |  |_|   \__,_|_| |_|\___|_|
                 |___/                             
EOF
    echo -e "${PLAIN}"
    echo -e "${CYAN}Xray Panel 管理脚本${PLAIN}"
    echo -e "${CYAN}========================================${PLAIN}"
    echo ""
}

# 显示菜单
show_menu() {
    echo -e "${GREEN}0.${PLAIN} 退出脚本"
    echo " ————————————————"
    echo -e "${GREEN}1.${PLAIN} 安装 Xray Panel"
    echo -e "${GREEN}2.${PLAIN} 更新 Xray Panel"
    echo -e "${GREEN}3.${PLAIN} 卸载 Xray Panel"
    echo " ————————————————"
    echo -e "${GREEN}4.${PLAIN} 启动 Xray Panel"
    echo -e "${GREEN}5.${PLAIN} 停止 Xray Panel"
    echo -e "${GREEN}6.${PLAIN} 重启 Xray Panel"
    echo -e "${GREEN}7.${PLAIN} 查看 Xray Panel 状态"
    echo -e "${GREEN}8.${PLAIN} 查看 Xray Panel 日志"
    echo " ————————————————"
    echo -e "${GREEN}9.${PLAIN} 设置 Xray Panel 开机自启"
    echo -e "${GREEN}10.${PLAIN} 取消 Xray Panel 开机自启"
    echo " ————————————————"
    echo -e "${GREEN}11.${PLAIN} 重置管理员账户"
    echo -e "${GREEN}12.${PLAIN} 查看管理员信息"
    echo -e "${GREEN}13.${PLAIN} 修改面板端口"
    echo " ————————————————"
    echo -e "${GREEN}14.${PLAIN} 启动 Xray"
    echo -e "${GREEN}15.${PLAIN} 停止 Xray"
    echo -e "${GREEN}16.${PLAIN} 重启 Xray"
    echo -e "${GREEN}17.${PLAIN} 查看 Xray 状态"
    echo -e "${GREEN}18.${PLAIN} 查看 Xray 日志"
    echo " ————————————————"
    echo -e "${GREEN}19.${PLAIN} 配置 Nginx 反向代理"
    echo -e "${GREEN}20.${PLAIN} 申请 SSL 证书"
    echo -e "${GREEN}21.${PLAIN} 申请通配符证书"
    echo -e "${GREEN}22.${PLAIN} 续期 SSL 证书"
    echo " ————————————————"
    echo -e "${GREEN}23.${PLAIN} 备份数据"
    echo -e "${GREEN}24.${PLAIN} 恢复数据"
    echo -e "${GREEN}25.${PLAIN} 清理日志"
    echo " ————————————————"
    echo ""
    read -p "请输入选择 [0-25]: " choice
    echo ""
}

# 1. 安装
install_panel() {
    echo -e "${BLUE}[INFO]${PLAIN} 开始安装 Xray Panel..."
    
    if [[ -f "$BINARY_PATH" ]]; then
        echo -e "${YELLOW}[WARNING]${PLAIN} Xray Panel 已安装"
        read -p "是否重新安装? (y/n): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            return
        fi
    fi
    
    # 下载并执行在线安装脚本
    bash <(curl -Ls https://raw.githubusercontent.com/$GITHUB_REPO/master/scripts/install-online.sh)
}

# 2. 更新
update_panel() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 开始更新 Xray Panel..."
    
    # 下载并执行在线安装脚本
    bash <(curl -Ls "https://raw.githubusercontent.com/$GITHUB_REPO/master/scripts/update.sh")
}

# 3. 卸载
uninstall_panel() {
    check_installed
    echo -e "${RED}[WARNING]${PLAIN} 即将卸载 Xray Panel"
    read -p "确认卸载? (yes/no): " -r
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        return
    fi
    
    bash <(curl -Ls "https://raw.githubusercontent.com/$GITHUB_REPO/master/scripts/uninstall.sh")
}

# 4. 启动
start_panel() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 启动 Xray Panel..."
    systemctl start xray-panel
    sleep 2
    if systemctl is-active --quiet xray-panel; then
        echo -e "${GREEN}[SUCCESS]${PLAIN} Xray Panel 已启动"
    else
        echo -e "${RED}[ERROR]${PLAIN} Xray Panel 启动失败"
        echo -e "${YELLOW}查看日志: journalctl -u xray-panel -n 50${PLAIN}"
    fi
}

# 5. 停止
stop_panel() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 停止 Xray Panel..."
    systemctl stop xray-panel
    echo -e "${GREEN}[SUCCESS]${PLAIN} Xray Panel 已停止"
}

# 6. 重启
restart_panel() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 重启 Xray Panel..."
    systemctl restart xray-panel
    sleep 2
    if systemctl is-active --quiet xray-panel; then
        echo -e "${GREEN}[SUCCESS]${PLAIN} Xray Panel 已重启"
    else
        echo -e "${RED}[ERROR]${PLAIN} Xray Panel 重启失败"
    fi
}

# 7. 查看状态
status_panel() {
    check_installed
    systemctl status xray-panel --no-pager
}

# 8. 查看日志
logs_panel() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 实时日志 (Ctrl+C 退出)..."
    journalctl -u xray-panel -f
}

# 9. 开机自启
enable_panel() {
    check_installed
    systemctl enable xray-panel
    echo -e "${GREEN}[SUCCESS]${PLAIN} 已设置开机自启"
}

# 10. 取消开机自启
disable_panel() {
    check_installed
    systemctl disable xray-panel
    echo -e "${GREEN}[SUCCESS]${PLAIN} 已取消开机自启"
}

# 11. 重置管理员
reset_admin() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 重置管理员账户"
    echo ""
    
    read -p "请输入管理员用户名: " username
    read -sp "请输入新密码: " password
    echo ""
    read -sp "请再次输入密码: " password2
    echo ""
    
    if [[ "$password" != "$password2" ]]; then
        echo -e "${RED}[ERROR]${PLAIN} 两次密码不一致"
        return
    fi
    
    if [[ ${#password} -lt 8 ]]; then
        echo -e "${RED}[ERROR]${PLAIN} 密码长度至少 8 位"
        return
    fi
    
    cd "$INSTALL_DIR"
    ./panel reset-password -username "$username" -password "$password"
}

# 12. 查看管理员信息
show_admin() {
    check_installed
    cd "$INSTALL_DIR"
    ./panel admin
}

# 13. 修改面板端口
change_port() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 修改面板端口"
    echo ""
    
    current_port=$(grep "listen:" "$CONFIG_DIR/config.yaml" | awk '{print $2}' | tr -d '"' | cut -d':' -f2)
    echo -e "当前端口: ${GREEN}$current_port${PLAIN}"
    echo ""
    
    read -p "请输入新端口 (1024-65535): " new_port
    
    if [[ ! $new_port =~ ^[0-9]+$ ]] || [[ $new_port -lt 1024 ]] || [[ $new_port -gt 65535 ]]; then
        echo -e "${RED}[ERROR]${PLAIN} 无效的端口号"
        return
    fi
    
    sed -i "s/listen: \".*\"/listen: \"0.0.0.0:$new_port\"/" "$CONFIG_DIR/config.yaml"
    
    echo -e "${GREEN}[SUCCESS]${PLAIN} 端口已修改为: $new_port"
    echo -e "${YELLOW}[INFO]${PLAIN} 请重启面板使配置生效"
    
    read -p "是否立即重启? (y/n): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        restart_panel
    fi
}

# 14-18. Xray 管理
start_xray() {
    systemctl start xray
    echo -e "${GREEN}[SUCCESS]${PLAIN} Xray 已启动"
}

stop_xray() {
    systemctl stop xray
    echo -e "${GREEN}[SUCCESS]${PLAIN} Xray 已停止"
}

restart_xray() {
    systemctl restart xray
    echo -e "${GREEN}[SUCCESS]${PLAIN} Xray 已重启"
}

status_xray() {
    systemctl status xray --no-pager
}

logs_xray() {
    echo -e "${BLUE}[INFO]${PLAIN} 实时日志 (Ctrl+C 退出)..."
    journalctl -u xray -f
}

# 19. 配置 Nginx 反向代理
configure_nginx() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 配置 Nginx 反向代理 (HTTPS)"
    echo ""
    
    read -p "请输入域名: " domain
    
    if [[ -z "$domain" ]]; then
        echo -e "${RED}[ERROR]${PLAIN} 域名不能为空"
        return
    fi
    
    echo -e "${YELLOW}[INFO]${PLAIN} 请确认您已申请了 SSL 证书"
    
     # 默认路径猜测
    default_cert="/etc/letsencrypt/live/$domain/fullchain.pem"
    default_key="/etc/letsencrypt/live/$domain/privkey.pem"
    
    read -p "请输入证书路径 (默认: $default_cert): " cert_path
    read -p "请输入私钥路径 (默认: $default_key): " key_path
    
    cert_path=${cert_path:-$default_cert}
    key_path=${key_path:-$default_key}
    
    # 检查证书文件
    if [[ ! -f "$cert_path" ]] || [[ ! -f "$key_path" ]]; then
        echo -e "${RED}[ERROR]${PLAIN} 证书文件不存在"
        echo -e "${YELLOW}[INFO]${PLAIN} 请先申请 SSL 证书（选项 20 或 21）"
        return
    fi
    
    cd "$INSTALL_DIR"
    ./panel nginx panel -domain="$domain" -cert="$cert_path" -key="$key_path"
    
    if [[ $? -eq 0 ]]; then
         ./panel nginx reload
         echo -e "${GREEN}[SUCCESS]${PLAIN} Nginx 配置完成"
         echo -e "${YELLOW}[INFO]${PLAIN} 访问地址: https://$domain"
    else
         echo -e "${RED}[ERROR]${PLAIN} Nginx 配置失败"
    fi
}

# 20. 申请 SSL 证书
apply_ssl() {
    echo -e "${BLUE}[INFO]${PLAIN} 申请 SSL 证书"
    echo ""
    
    read -p "请输入域名: " domain
    read -p "请输入邮箱: " email
    
    if [[ -z "$domain" ]] || [[ -z "$email" ]]; then
        echo -e "${RED}[ERROR]${PLAIN} 域名和邮箱不能为空"
        return
    fi
    
    certbot --nginx -d "$domain" --email "$email" --agree-tos --no-eff-email
    
    if [[ $? -eq 0 ]]; then
        echo -e "${GREEN}[SUCCESS]${PLAIN} SSL 证书申请成功"
    else
        echo -e "${RED}[ERROR]${PLAIN} SSL 证书申请失败"
    fi
}

# 21. 申请通配符证书
apply_wildcard_ssl() {
    echo -e "${BLUE}[INFO]${PLAIN} 申请通配符 SSL 证书"
    echo ""
    echo -e "${YELLOW}注意: 通配符证书需要 DNS 验证${PLAIN}"
    echo ""
    
    read -p "请输入主域名 (例如: example.com): " domain
    read -p "请输入邮箱: " email
    
    if [[ -z "$domain" ]] || [[ -z "$email" ]]; then
        echo -e "${RED}[ERROR]${PLAIN} 域名和邮箱不能为空"
        return
    fi
    
    echo ""
    echo -e "${CYAN}选择 DNS 提供商:${PLAIN}"
    echo "1. Cloudflare"
    echo "2. 阿里云 (Aliyun)"
    echo "3. 腾讯云 (Tencent)"
    echo "4. 手动 DNS 验证"
    read -p "请选择 [1-4]: " dns_choice
    
    case $dns_choice in
        1)
            # Cloudflare
            if ! command -v certbot &> /dev/null; then
                apt-get install -y certbot python3-certbot-dns-cloudflare || yum install -y certbot python3-certbot-dns-cloudflare
            fi
            
            read -p "请输入 Cloudflare API Token: " cf_token
            
            mkdir -p /root/.secrets
            cat > /root/.secrets/cloudflare.ini <<EOF
dns_cloudflare_api_token = $cf_token
EOF
            chmod 600 /root/.secrets/cloudflare.ini
            
            certbot certonly --dns-cloudflare \
                --dns-cloudflare-credentials /root/.secrets/cloudflare.ini \
                -d "$domain" -d "*.$domain" \
                --email "$email" --agree-tos --no-eff-email
            ;;
        2)
            # 阿里云
            echo -e "${YELLOW}[INFO]${PLAIN} 阿里云 DNS 插件需要手动安装"
            echo "参考: https://github.com/justjavac/certbot-dns-aliyun"
            ;;
        3)
            # 腾讯云
            echo -e "${YELLOW}[INFO]${PLAIN} 腾讯云 DNS 插件需要手动安装"
            echo "参考: https://github.com/frefreak/certbot-dns-tencent"
            ;;
        4)
            # 手动验证
            certbot certonly --manual --preferred-challenges dns \
                -d "$domain" -d "*.$domain" \
                --email "$email" --agree-tos --no-eff-email
            ;;
        *)
            echo -e "${RED}[ERROR]${PLAIN} 无效的选择"
            return
            ;;
    esac
    
    if [[ $? -eq 0 ]]; then
        echo -e "${GREEN}[SUCCESS]${PLAIN} 通配符证书申请成功"
        echo -e "${YELLOW}[INFO]${PLAIN} 证书路径: /etc/letsencrypt/live/$domain/"
    else
        echo -e "${RED}[ERROR]${PLAIN} 证书申请失败"
    fi
}

# 22. 续期证书
renew_ssl() {
    echo -e "${BLUE}[INFO]${PLAIN} 续期 SSL 证书..."
    certbot renew
    
    if [[ $? -eq 0 ]]; then
        echo -e "${GREEN}[SUCCESS]${PLAIN} 证书续期成功"
        systemctl reload nginx
    else
        echo -e "${RED}[ERROR]${PLAIN} 证书续期失败"
    fi
}

# 23. 备份数据
backup_data() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 备份数据..."
    
    BACKUP_DIR="/root/xray-panel-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # 备份数据库
    if [[ -f "$DATA_DIR/panel.db" ]]; then
        cp "$DATA_DIR/panel.db" "$BACKUP_DIR/"
    fi
    
    # 备份配置
    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        cp "$CONFIG_DIR/config.yaml" "$BACKUP_DIR/"
    fi
    
    # 打包
    tar czf "$BACKUP_DIR.tar.gz" -C "$(dirname $BACKUP_DIR)" "$(basename $BACKUP_DIR)"
    rm -rf "$BACKUP_DIR"
    
    echo -e "${GREEN}[SUCCESS]${PLAIN} 备份完成: $BACKUP_DIR.tar.gz"
}

# 24. 恢复数据
restore_data() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 恢复数据"
    echo ""
    
    read -p "请输入备份文件路径: " backup_file
    
    if [[ ! -f "$backup_file" ]]; then
        echo -e "${RED}[ERROR]${PLAIN} 备份文件不存在"
        return
    fi
    
    echo -e "${YELLOW}[WARNING]${PLAIN} 恢复数据将覆盖当前数据"
    read -p "确认恢复? (yes/no): " -r
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        return
    fi
    
    # 停止服务
    systemctl stop xray-panel
    
    # 解压
    TEMP_DIR="/tmp/xray-panel-restore-$$"
    mkdir -p "$TEMP_DIR"
    tar xzf "$backup_file" -C "$TEMP_DIR"
    
    # 恢复文件
    BACKUP_CONTENT=$(find "$TEMP_DIR" -mindepth 1 -maxdepth 1 -type d | head -n 1)
    
    if [[ -f "$BACKUP_CONTENT/panel.db" ]]; then
        cp "$BACKUP_CONTENT/panel.db" "$DATA_DIR/"
    fi
    
    if [[ -f "$BACKUP_CONTENT/config.yaml" ]]; then
        cp "$BACKUP_CONTENT/config.yaml" "$CONFIG_DIR/"
    fi
    
    rm -rf "$TEMP_DIR"
    
    # 启动服务
    systemctl start xray-panel
    
    echo -e "${GREEN}[SUCCESS]${PLAIN} 数据恢复完成"
}

# 25. 清理日志
clean_logs() {
    check_installed
    echo -e "${BLUE}[INFO]${PLAIN} 清理日志..."
    
    # 清理面板日志
    if [[ -d "$LOG_DIR" ]]; then
        find "$LOG_DIR" -name "*.log" -mtime +7 -delete
        find "$LOG_DIR" -name "*.gz" -delete
    fi
    
    # 清理系统日志
    journalctl --vacuum-time=7d
    
    echo -e "${GREEN}[SUCCESS]${PLAIN} 日志清理完成"
}

# 主循环
main() {
    check_root
    
    while true; do
        show_logo
        show_menu
        
        case $choice in
            0)
                echo -e "${GREEN}退出脚本${PLAIN}"
                exit 0
                ;;
            1) install_panel ;;
            2) update_panel ;;
            3) uninstall_panel ;;
            4) start_panel ;;
            5) stop_panel ;;
            6) restart_panel ;;
            7) status_panel ;;
            8) logs_panel ;;
            9) enable_panel ;;
            10) disable_panel ;;
            11) reset_admin ;;
            12) show_admin ;;
            13) change_port ;;
            14) start_xray ;;
            15) stop_xray ;;
            16) restart_xray ;;
            17) status_xray ;;
            18) logs_xray ;;
            19) configure_nginx ;;
            20) apply_ssl ;;
            21) apply_wildcard_ssl ;;
            22) renew_ssl ;;
            23) backup_data ;;
            24) restore_data ;;
            25) clean_logs ;;
            *)
                echo -e "${RED}无效的选择${PLAIN}"
                ;;
        esac
        
        echo ""
        read -p "按回车键继续..." 
    done
}

# 运行主程序
main
