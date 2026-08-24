#!/bin/bash
# Xray Panel 交互式管理脚本

set -euo pipefail

# ── 颜色 ──────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; PLAIN='\033[0m'

# ── 常量 ──────────────────────────────────────────────────────────────────────
GITHUB_REPO="nxovaeng/xray-panel"
INSTALL_DIR="/opt/xray-panel"
CONFIG_DIR="${INSTALL_DIR}/conf"
DATA_DIR="${INSTALL_DIR}/data"
LOG_DIR="${INSTALL_DIR}/logs"
BINARY_PATH="${INSTALL_DIR}/panel"
SCRIPTS_DIR="${INSTALL_DIR}/scripts"
XRAY_ASSETS="/usr/local/share/xray"

# ── 工具函数 ──────────────────────────────────────────────────────────────────
info()    { echo -e "${BLUE}[INFO]${PLAIN}  $*"; }
ok()      { echo -e "${GREEN}[OK]${PLAIN}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${PLAIN}  $*"; }
err()     { echo -e "${RED}[ERROR]${PLAIN} $*"; }
pause()   { echo ""; read -rp "按回车键继续..."; }

need_root()      { [[ $EUID -eq 0 ]] || { err "请以 root 权限运行"; exit 1; }; }
need_installed() {
    [[ -f "$BINARY_PATH" ]] || {
        err "Xray Panel 未安装"
        echo -e "  运行安装命令: ${CYAN}bash <(curl -Ls https://raw.githubusercontent.com/$GITHUB_REPO/master/scripts/install.sh) install${PLAIN}"
        exit 1
    }
}

# 调用 scripts/install.sh（首次安装 & 生命周期管理脚本）
call_scripts() {
    local script="$SCRIPTS_DIR/install.sh"
    if [[ -f "$script" ]]; then
        bash "$script" "$@"
    else
        # 回退：从网络下载执行
        bash <(curl -fsSL "https://raw.githubusercontent.com/$GITHUB_REPO/master/scripts/install.sh") "$@"
    fi
}

# ── 界面 ──────────────────────────────────────────────────────────────────────
show_header() {
    clear
    echo -e "${CYAN}"
    cat << 'EOF'
 __   __                   ____                  _
 \ \ / /_ __ __ _ _   _   |  _ \ __ _ _ __   ___| |
  \ V / '__/ _` | | | |  | |_) / _` | '_ \ / _ \ |
   | || | | (_| | |_| |  |  __/ (_| | | | |  __/ |
   |_||_|  \__,_|\__, |  |_|   \__,_|_| |_|\___|_|
                 |___/
EOF
    echo -e "${PLAIN}"

    # 状态行
    local panel_status xray_status
    if systemctl is-active --quiet xray-panel 2>/dev/null; then
        panel_status="${GREEN}运行中${PLAIN}"
    else
        panel_status="${RED}已停止${PLAIN}"
    fi
    if systemctl is-active --quiet xray 2>/dev/null; then
        xray_status="${GREEN}运行中${PLAIN}"
    else
        xray_status="${RED}已停止${PLAIN}"
    fi

    echo -e "  面板: $panel_status    Xray: $xray_status"
    echo -e "${CYAN}────────────────────────────────────────${PLAIN}"
    echo ""
}

show_menu() {
    echo -e "  ${CYAN}── 面板管理 ──────────────────────────${PLAIN}"
    echo -e "  ${GREEN}1.${PLAIN}  安装面板           ${GREEN}2.${PLAIN}  更新面板"
    echo -e "  ${GREEN}3.${PLAIN}  卸载面板           ${GREEN}4.${PLAIN}  更新管理脚本"
    echo ""
    echo -e "  ${CYAN}── 服务控制 ──────────────────────────${PLAIN}"
    echo -e "  ${GREEN}5.${PLAIN}  启动面板           ${GREEN}6.${PLAIN}  停止面板"
    echo -e "  ${GREEN}7.${PLAIN}  重启面板           ${GREEN}8.${PLAIN}  查看面板状态"
    echo -e "  ${GREEN}9.${PLAIN}  实时日志           ${GREEN}10.${PLAIN} 开机自启 (切换)"
    echo ""
    echo -e "  ${CYAN}── 账户配置 ──────────────────────────${PLAIN}"
    echo -e "  ${GREEN}11.${PLAIN} 查看管理员信息     ${GREEN}12.${PLAIN} 重置管理员密码"
    echo -e "  ${GREEN}13.${PLAIN} 修改面板端口"
    echo ""
    echo -e "  ${CYAN}── Xray-core ─────────────────────────${PLAIN}"
    echo -e "  ${GREEN}14.${PLAIN} 启动 Xray          ${GREEN}15.${PLAIN} 停止 Xray"
    echo -e "  ${GREEN}16.${PLAIN} 重启 Xray          ${GREEN}17.${PLAIN} Xray 状态/日志"
    echo -e "  ${GREEN}18.${PLAIN} 更新 Xray-core     ${GREEN}19.${PLAIN} 更新 Geo 文件"
    echo ""
    echo -e "  ${CYAN}── Nginx / 数据 ──────────────────────${PLAIN}"
    echo -e "  ${GREEN}20.${PLAIN} 配置 Nginx 反代    ${GREEN}21.${PLAIN} 备份数据"
    echo -e "  ${GREEN}22.${PLAIN} 恢复数据           ${GREEN}23.${PLAIN} 清理旧日志"
    echo ""
    echo -e "  ${CYAN}── 工具 ──────────────────────────────${PLAIN}"
    echo -e "  ${GREEN}24.${PLAIN} 申请 WARP 配置"
    echo ""
    echo -e "  ${GREEN}0.${PLAIN}  退出"
    echo ""
    read -rp "  请输入选择 [0-24]: " choice
    echo ""
}

# ══════════════════════════════════════════════════════════════════════════════
# 1. 安装面板
# ══════════════════════════════════════════════════════════════════════════════
do_install() {
    if [[ -f "$BINARY_PATH" ]]; then
        warn "Xray Panel 已安装"
        read -rp "是否重新安装? (y/N): " ans
        [[ "$ans" =~ ^[Yy]$ ]] || return
    fi
    call_scripts install
}

# ══════════════════════════════════════════════════════════════════════════════
# 2. 更新面板
# ══════════════════════════════════════════════════════════════════════════════
do_update() {
    need_installed
    read -rp "更新到版本 (留空=latest): " ver
    call_scripts update "${ver:-latest}"
}

# ══════════════════════════════════════════════════════════════════════════════
# 3. 卸载面板
# ══════════════════════════════════════════════════════════════════════════════
do_uninstall() {
    need_installed
    call_scripts uninstall
}

# ══════════════════════════════════════════════════════════════════════════════
# 4. 更新管理脚本（本文件自身）
# ══════════════════════════════════════════════════════════════════════════════
do_update_self() {
    info "从 GitHub 下载最新管理脚本..."
    local url="https://raw.githubusercontent.com/$GITHUB_REPO/master/xray-panel.sh"
    local tmp
    tmp=$(mktemp)
    if curl -fsSL "$url" -o "$tmp" && head -1 "$tmp" | grep -q '^#!'; then
        cp "$tmp" "$INSTALL_DIR/xray-panel.sh"
        chmod +x "$INSTALL_DIR/xray-panel.sh"
        ln -sf "$INSTALL_DIR/xray-panel.sh" /usr/local/bin/xray-panel.sh
        ln -sf /usr/local/bin/xray-panel.sh /usr/bin/xray-panel
        rm -f "$tmp"
        ok "管理脚本已更新，请重新运行 xray-panel"
        exit 0
    else
        rm -f "$tmp"
        err "下载失败，请检查网络"
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# 5-10. 面板服务控制
# ══════════════════════════════════════════════════════════════════════════════
do_start()   { need_installed; systemctl start   xray-panel && ok "面板已启动"   || err "启动失败，查看: journalctl -u xray-panel -n 30"; }
do_stop()    { need_installed; systemctl stop    xray-panel && ok "面板已停止"; }
do_restart() { need_installed; systemctl restart xray-panel && ok "面板已重启"   || err "重启失败，查看: journalctl -u xray-panel -n 30"; }
do_status()  { need_installed; systemctl status  xray-panel --no-pager -l; }
do_logs()    { need_installed; info "实时日志 (Ctrl+C 退出)"; journalctl -u xray-panel -f; }

do_autostart() {
    need_installed
    if systemctl is-enabled --quiet xray-panel 2>/dev/null; then
        systemctl disable xray-panel
        ok "已关闭开机自启"
    else
        systemctl enable xray-panel
        ok "已开启开机自启"
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# 11. 查看管理员信息
# ══════════════════════════════════════════════════════════════════════════════
do_show_admin() {
    need_installed
    cd "$INSTALL_DIR"
    ./panel admin -config "$CONFIG_DIR/config.yaml"
}

# ══════════════════════════════════════════════════════════════════════════════
# 12. 重置管理员密码
# ══════════════════════════════════════════════════════════════════════════════
do_reset_password() {
    need_installed
    echo ""
    read -rp "管理员用户名: " uname
    [[ -n "$uname" ]] || { err "用户名不能为空"; return; }

    local pass pass2
    read -rsp "新密码: " pass; echo ""
    read -rsp "确认密码: " pass2; echo ""

    [[ "$pass" == "$pass2" ]] || { err "两次密码不一致"; return; }
    [[ ${#pass} -ge 8 ]]     || { err "密码长度至少 8 位"; return; }

    cd "$INSTALL_DIR"
    ./panel reset-password -config "$CONFIG_DIR/config.yaml" -u "$uname" -p "$pass"
}

# ══════════════════════════════════════════════════════════════════════════════
# 13. 修改面板端口
# ══════════════════════════════════════════════════════════════════════════════
do_change_port() {
    need_installed
    local cfg="$CONFIG_DIR/config.yaml"
    local cur_port
    cur_port=$(grep -oP 'listen:\s*"\K[^"]+' "$cfg" | cut -d: -f2)
    info "当前端口: ${CYAN}$cur_port${PLAIN}"
    read -rp "新端口 (1024-65535): " new_port
    [[ "$new_port" =~ ^[0-9]+$ ]] && (( new_port >= 1024 && new_port <= 65535 )) \
        || { err "无效端口"; return; }
    sed -i "s/listen: \".*\"/listen: \"127.0.0.1:$new_port\"/" "$cfg"
    ok "端口已修改为 $new_port，需重启面板生效"
    read -rp "立即重启? (y/N): " ans
    [[ "$ans" =~ ^[Yy]$ ]] && do_restart || true
}

# ══════════════════════════════════════════════════════════════════════════════
# 14-17. Xray-core 服务控制
# ══════════════════════════════════════════════════════════════════════════════
do_xray_start()   { systemctl start   xray && ok "Xray 已启动"; }
do_xray_stop()    { systemctl stop    xray && ok "Xray 已停止"; }
do_xray_restart() { systemctl restart xray && ok "Xray 已重启"; }
do_xray_status()  {
    systemctl status xray --no-pager -l
    echo ""
    info "实时日志 (Ctrl+C 退出)"
    journalctl -u xray -f
}

# ══════════════════════════════════════════════════════════════════════════════
# 18. 更新 Xray-core
# ══════════════════════════════════════════════════════════════════════════════
do_update_xray() {
    local cur=""
    command -v xray &>/dev/null && cur=$(xray version 2>/dev/null | head -1 | grep -oP 'Xray \K[\d.]+' || echo "")
    [[ -n "$cur" ]] && info "当前版本: $cur"

    local latest
    latest=$(curl -fsSL "https://api.github.com/repos/XTLS/Xray-core/releases/latest" \
        | grep -oP '"tag_name":\s*"\K[^"]+' | head -1 || echo "")
    [[ -n "$latest" ]] || { err "获取最新版本失败"; return; }
    info "最新版本: $latest"

    [[ "${cur}" == "${latest#v}" ]] && { ok "已是最新版本"; return; }

    read -rp "确认更新到 $latest? (y/N): " ans
    [[ "$ans" =~ ^[Yy]$ ]] || return

    bash -c "$(curl -fsSL https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install
    systemctl restart xray
    ok "Xray-core 已更新并重启"
}

# ══════════════════════════════════════════════════════════════════════════════
# 19. 更新 Geo 文件
# ══════════════════════════════════════════════════════════════════════════════
do_update_geo() { call_scripts update-geo; }

# ══════════════════════════════════════════════════════════════════════════════
# 20. 配置 Nginx 反向代理
# ══════════════════════════════════════════════════════════════════════════════
do_nginx() {
    need_installed
    echo ""
    read -rp "面板域名: " domain
    [[ -n "$domain" ]] || { err "域名不能为空"; return; }

    local default_cert="/etc/letsencrypt/live/$domain/fullchain.pem"
    local default_key="/etc/letsencrypt/live/$domain/privkey.pem"

    read -rp "证书路径 (默认: $default_cert): " cert_path
    read -rp "私钥路径 (默认: $default_key): " key_path
    cert_path="${cert_path:-$default_cert}"
    key_path="${key_path:-$default_key}"

    [[ -f "$cert_path" ]] || { err "证书文件不存在: $cert_path"; return; }
    [[ -f "$key_path"  ]] || { err "私钥文件不存在: $key_path";  return; }

    cd "$INSTALL_DIR"
    ./panel nginx panel -config "$CONFIG_DIR/config.yaml" -d "$domain" -cert "$cert_path" -key "$key_path"
    ./panel nginx reload -config "$CONFIG_DIR/config.yaml"
    ok "Nginx 配置完成，访问: https://$domain"
}

# ══════════════════════════════════════════════════════════════════════════════
# 21. 备份数据
# ══════════════════════════════════════════════════════════════════════════════
do_backup() {
    need_installed
    local dst="/root/xray-panel-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$dst"
    [[ -f "$DATA_DIR/panel.db" ]]      && cp "$DATA_DIR/panel.db" "$dst/"
    [[ -f "$CONFIG_DIR/config.yaml" ]] && cp "$CONFIG_DIR/config.yaml" "$dst/"
    tar czf "$dst.tar.gz" -C "$(dirname "$dst")" "$(basename "$dst")"
    rm -rf "$dst"
    ok "备份完成: $dst.tar.gz"
}

# ══════════════════════════════════════════════════════════════════════════════
# 22. 恢复数据
# ══════════════════════════════════════════════════════════════════════════════
do_restore() {
    need_installed
    read -rp "备份文件路径: " bak
    [[ -f "$bak" ]] || { err "文件不存在: $bak"; return; }

    warn "恢复将覆盖当前数据库和配置"
    read -rp "确认继续? (yes/no): " ans
    [[ "$ans" =~ ^[Yy][Ee][Ss]$ ]] || return

    systemctl stop xray-panel 2>/dev/null || true
    local tmp
    tmp=$(mktemp -d)
    trap "rm -rf $tmp" RETURN
    tar xzf "$bak" -C "$tmp"
    local src
    src=$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -1)
    [[ -f "$src/panel.db" ]]    && cp "$src/panel.db" "$DATA_DIR/"
    [[ -f "$src/config.yaml" ]] && cp "$src/config.yaml" "$CONFIG_DIR/"
    systemctl start xray-panel
    ok "恢复完成"
}

# ══════════════════════════════════════════════════════════════════════════════
# 23. 清理旧日志
# ══════════════════════════════════════════════════════════════════════════════
do_clean_logs() {
    need_installed
    [[ -d "$LOG_DIR" ]] && find "$LOG_DIR" -name "*.log.*" -mtime +7 -delete
    journalctl --vacuum-time=7d
    ok "7 天前的旧日志已清理"
}

# ══════════════════════════════════════════════════════════════════════════════
# 24. 申请 WARP WireGuard 配置
# ══════════════════════════════════════════════════════════════════════════════
do_warp() {
    # 安装 wgcf
    if ! command -v wgcf &>/dev/null; then
        info "下载 wgcf..."
        local arch
        case $(uname -m) in
            x86_64)  arch="amd64" ;;
            aarch64) arch="arm64" ;;
            armv7l)  arch="armv7" ;;
            *) err "不支持的架构: $(uname -m)"; return ;;
        esac
        local ver
        ver=$(curl -fsSL "https://api.github.com/repos/ViRb3/wgcf/releases/latest" \
            | grep -oP '"tag_name":\s*"\K[^"]+' | head -1 || echo "v2.2.26")
        wget -q "https://github.com/ViRb3/wgcf/releases/download/${ver}/wgcf_${ver#v}_linux_${arch}" \
            -O /usr/local/bin/wgcf && chmod +x /usr/local/bin/wgcf \
            || { err "wgcf 下载失败"; return; }
        ok "wgcf 已安装 ($ver)"
    fi

    local wdir="$INSTALL_DIR/warp"
    mkdir -p "$wdir"
    cd "$wdir"

    if [[ -f "wgcf-account.toml" ]]; then
        warn "已存在 WARP 账户"
        read -rp "重新注册新账户? (y/N): " ans
        [[ "$ans" =~ ^[Yy]$ ]] && rm -f wgcf-account.toml wgcf-profile.conf
    fi

    if [[ ! -f "wgcf-account.toml" ]]; then
        info "注册 WARP 账户..."
        wgcf register --accept-tos || { err "注册失败，请检查网络"; return; }
    fi

    info "生成 WireGuard 配置..."
    wgcf generate || { err "配置生成失败"; return; }

    local conf="$wdir/wgcf-profile.conf"
    [[ -f "$conf" ]] || { err "配置文件未生成"; return; }

    local priv_key ep peer_pub addr ipv4 ipv6 mtu
    priv_key=$(awk '/PrivateKey/ {print $3}' "$conf")
    ep=$(awk '/Endpoint/ {print $3}' "$conf")
    peer_pub=$(awk '/PublicKey/ {print $3}' "$conf")
    addr=$(awk '/Address/ {print $3}' "$conf")
    mtu=$(awk '/MTU/ {print $3}' "$conf")
    ipv4=$(echo "$addr" | tr ',' '\n' | grep -v ':' | tr -d ' ')
    ipv6=$(echo "$addr" | tr ',' '\n' | grep  ':' | tr -d ' ')

    echo ""
    echo -e "${GREEN}────────────────────────────────────────${PLAIN}"
    echo -e "${GREEN}  WARP WireGuard 配置${PLAIN}"
    echo -e "${GREEN}────────────────────────────────────────${PLAIN}"
    echo -e "  服务器:     ${CYAN}${ep%:*}${PLAIN}"
    echo -e "  端口:       ${CYAN}${ep##*:} (UDP)${PLAIN}"
    echo -e "  私钥:       ${CYAN}$priv_key${PLAIN}"
    echo -e "  对端公钥:   ${CYAN}$peer_pub${PLAIN}"
    echo -e "  本地 IPv4:  ${CYAN}$ipv4${PLAIN}"
    echo -e "  本地 IPv6:  ${CYAN}$ipv6${PLAIN}"
    echo -e "  MTU:        ${CYAN}${mtu:-1280}${PLAIN}"
    echo -e "  AllowedIPs: ${CYAN}0.0.0.0/0, ::/0${PLAIN}"
    echo -e "${GREEN}────────────────────────────────────────${PLAIN}"
    echo -e "  配置文件:   $conf"
    echo ""
}

# ══════════════════════════════════════════════════════════════════════════════
# 主循环
# ══════════════════════════════════════════════════════════════════════════════
main() {
    need_root

    while true; do
        show_header
        show_menu

        case "${choice:-}" in
            0)  echo -e "${GREEN}再见${PLAIN}"; exit 0 ;;
            1)  do_install ;;
            2)  do_update ;;
            3)  do_uninstall ;;
            4)  do_update_self ;;
            5)  do_start ;;
            6)  do_stop ;;
            7)  do_restart ;;
            8)  do_status ;;
            9)  do_logs ;;
            10) do_autostart ;;
            11) do_show_admin ;;
            12) do_reset_password ;;
            13) do_change_port ;;
            14) do_xray_start ;;
            15) do_xray_stop ;;
            16) do_xray_restart ;;
            17) do_xray_status ;;
            18) do_update_xray ;;
            19) do_update_geo ;;
            20) do_nginx ;;
            21) do_backup ;;
            22) do_restore ;;
            23) do_clean_logs ;;
            24) do_warp ;;
            *)  err "无效选择: ${choice:-}，请输入 0-24" ;;
        esac

        pause
    done
}

main
