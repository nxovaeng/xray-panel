#!/bin/bash
# install.sh — Xray Panel 首次安装 & 生命周期管理脚本
#
# 两种使用场景:
#   1. 首次安装 (在线/本地)  — 由用户或 CI 直接调用
#   2. 安装后管理 (update / uninstall / geo) — 由根目录 xray-panel.sh 的菜单调用
#
# 用法:
#   curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install.sh | sudo bash -s -- install
#   bash install.sh install <pkg.tar.gz>  # 本地包安装
#   bash install.sh update [version]      # 更新面板二进制（默认 latest）
#   bash install.sh update-tool           # 仅更新交互管理脚本 xray-panel
#   bash install.sh uninstall             # 卸载
#   bash install.sh update-geo            # 更新 geo 文件
#   bash install.sh status                # 查看状态
#
# 可选环境变量:
#   GITHUB_REPO   - 仓库 (默认: nxovaeng/xray-panel)
#   PANEL_VERSION - 版本 (默认: latest)

set -euo pipefail

# ── 颜色 ──────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; PLAIN='\033[0m'

# ── 常量 ──────────────────────────────────────────────────────────────────────
GITHUB_REPO="${GITHUB_REPO:-nxovaeng/xray-panel}"
INSTALL_DIR="/opt/xray-panel"
CONFIG_DIR="${INSTALL_DIR}/conf"
DATA_DIR="${INSTALL_DIR}/data"
LOG_DIR="${INSTALL_DIR}/logs"
BINARY_PATH="${INSTALL_DIR}/panel"
SYSTEMD_FILE="/etc/systemd/system/xray-panel.service"
XRAY_ASSETS="/usr/local/share/xray"

# ── 日志工具 ──────────────────────────────────────────────────────────────────
info()    { echo -e "${BLUE}[INFO]${PLAIN}  $*"; }
ok()      { echo -e "${GREEN}[OK]${PLAIN}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${PLAIN}  $*"; }
die()     { echo -e "${RED}[ERROR]${PLAIN} $*" >&2; exit 1; }
hr()      { echo -e "${CYAN}────────────────────────────────────────${PLAIN}"; }

# ── 环境检测 ──────────────────────────────────────────────────────────────────
need_root()  { [[ $EUID -eq 0 ]] || die "请以 root 权限运行"; }

detect_arch() {
    case $(uname -m) in
        x86_64)        ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) die "不支持的架构: $(uname -m)" ;;
    esac
}

detect_os() {
    [[ -f /etc/os-release ]] || die "无法检测操作系统"
    # shellcheck source=/dev/null
    . /etc/os-release
    OS="${ID:-unknown}"
}

pkg_install() {
    # Usage: pkg_install pkg1 pkg2 ...
    case "$OS" in
        ubuntu|debian)
            apt-get install -y -q "$@" ;;
        centos|rhel|rocky|almalinux|fedora)
            yum install -y "$@" ;;
        *) die "不支持的操作系统: $OS（请手动安装: $*）" ;;
    esac
}

install_deps() {
    info "检查依赖..."
    local missing=()
    for cmd in curl wget tar; do
        command -v "$cmd" &>/dev/null || missing+=("$cmd")
    done
    if [[ ${#missing[@]} -gt 0 ]]; then
        info "安装缺失依赖: ${missing[*]}"
        case "$OS" in
            ubuntu|debian) apt-get update -qq ;;
            centos|rhel|rocky|almalinux|fedora) yum install -y epel-release 2>/dev/null || true ;;
        esac
        pkg_install "${missing[@]}"
    fi
    ok "依赖就绪"
}

# ── GitHub 版本解析 ───────────────────────────────────────────────────────────
resolve_version() {
    # Sets PANEL_VERSION to an actual tag.
    local ver="${1:-${PANEL_VERSION:-latest}}"
    if [[ "$ver" == "latest" ]]; then
        info "获取最新版本..."
        ver=$(curl -fsSL "https://api.github.com/repos/$GITHUB_REPO/releases/latest" \
            | grep -oP '"tag_name":\s*"\K[^"]+' | head -1)
        [[ -n "$ver" ]] || die "无法获取最新版本（请检查网络或手动指定版本）"
    fi
    PANEL_VERSION="$ver"
    info "版本: $PANEL_VERSION"
}

# ── 下载 panel 二进制 ─────────────────────────────────────────────────────────
download_binary() {
    # Places the binary at $BINARY_PATH. Requires PANEL_VERSION and ARCH set.

    # 版本相同时跳过（update 命令也会调用此函数，这里做二次保护）
    if [[ -f "$BINARY_PATH" ]]; then
        local cur_ver
        cur_ver=$("$BINARY_PATH" version 2>/dev/null | grep -oP 'version \K[^ ]+' || echo "")
        if [[ -n "$cur_ver" && "$cur_ver" == "${PANEL_VERSION#v}" ]]; then
            ok "已是目标版本 ($cur_ver)，跳过下载"
            return
        fi
    fi

    local url="https://github.com/$GITHUB_REPO/releases/download/${PANEL_VERSION}/xray-panel-${PANEL_VERSION}-linux-${ARCH}.tar.gz"
    info "下载: $url"

    local tmp
    tmp=$(mktemp -d)
    # shellcheck disable=SC2064
    trap "rm -rf $tmp" RETURN

    wget -q --show-progress "$url" -O "$tmp/pkg.tar.gz" || die "下载失败，请检查网络或版本号"
    tar xzf "$tmp/pkg.tar.gz" -C "$tmp"

    local bin
    bin=$(find "$tmp" \( -name "panel-linux-${ARCH}" -o -name "panel" \) -type f | head -1)
    [[ -n "$bin" ]] || die "压缩包内未找到 panel 二进制"

    chmod +x "$bin"

    # 停止服务避免 "Text file busy"
    if systemctl is-active --quiet xray-panel 2>/dev/null; then
        info "停止面板服务..."
        systemctl stop xray-panel
    fi
    # 删除旧二进制（运行中的文件不能直接覆盖）
    rm -f "$BINARY_PATH"
    cp "$bin" "$BINARY_PATH"
    ok "面板二进制已安装 ($PANEL_VERSION)"
}

# ── Xray-core 安装 ────────────────────────────────────────────────────────────
install_xray_core() {
    if command -v xray &>/dev/null; then
        warn "Xray-core 已安装 ($(xray version 2>/dev/null | head -1))，跳过"
        return
    fi
    info "安装 Xray-core..."
    bash -c "$(curl -fsSL https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install
    ok "Xray-core 安装完成"
}

# ── 配置模板 & 配置生成 ───────────────────────────────────────────────────────
copy_config_template() {
    # 始终把 config.yaml.example 更新到 conf/，方便用户参考最新字段
    mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
    local example_src=""

    # 优先从解压目录取（本地包安装）
    if [[ -n "${EXTRACTED_DIR:-}" && -f "$EXTRACTED_DIR/conf/config.yaml.example" ]]; then
        example_src="$EXTRACTED_DIR/conf/config.yaml.example"
    fi

    if [[ -n "$example_src" ]]; then
        cp "$example_src" "$CONFIG_DIR/config.yaml.example"
        ok "配置模板已更新: $CONFIG_DIR/config.yaml.example"
    fi
    # 在线安装时模板随二进制包一起发布，跳过（避免额外网络请求）
}

generate_config() {
    mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"

    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        warn "config.yaml 已存在，保留原有配置（不覆盖）"
        return
    fi

    local secret
    secret=$(head /dev/urandom | tr -dc 'A-Za-z0-9' | head -c 48)
    cat > "$CONFIG_DIR/config.yaml" <<EOF
# Xray Panel 配置文件  生成于 $(date '+%Y-%m-%d %H:%M:%S')

server:
  listen: "127.0.0.1:8082"
  debug: false

log:
  level: "info"
  file: "$LOG_DIR/panel.log"
  max_size: 100
  max_backups: 7
  max_age: 30
  compress: true

database:
  path: "$DATA_DIR/panel.db"

jwt:
  secret: "$secret"
  expire_hour: 168

admin:
  username: ""
  password: ""
  email: ""

xray:
  binary_path: "/usr/local/bin/xray"
  config_path: "/usr/local/etc/xray/config.json"
  assets_path: "$XRAY_ASSETS"
  api_port: 10085
  socket_dir: "/dev/shm"

nginx:
  config_dir: "/etc/nginx/conf.d"
  stream_dir: "/etc/nginx/stream.d"
  reload_cmd: "systemctl reload nginx"
  cert_dir: "/root/.acme.sh"
EOF
    ok "配置文件已生成: $CONFIG_DIR/config.yaml"
}

# ── systemd 服务 ──────────────────────────────────────────────────────────────
setup_service() {
    cat > "$SYSTEMD_FILE" <<EOF
[Unit]
Description=Xray Panel Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$BINARY_PATH server -config $CONFIG_DIR/config.yaml
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable xray-panel
    ok "systemd 服务已创建并启用"
}

# ── 备份 ──────────────────────────────────────────────────────────────────────
backup() {
    local dst="/root/xray-panel-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$dst"
    [[ -f "$BINARY_PATH" ]]            && cp "$BINARY_PATH" "$dst/"
    [[ -f "$DATA_DIR/panel.db" ]]      && cp "$DATA_DIR/panel.db" "$dst/"
    [[ -f "$CONFIG_DIR/config.yaml" ]] && cp "$CONFIG_DIR/config.yaml" "$dst/"
    ok "备份已创建: $dst"
}

# ── 安装/更新交互管理工具 ────────────────────────────────────────────────────
# force=true 时无条件覆盖（供 update-tool 使用）；默认仅在不存在时安装。
install_management_tool() {
    local force="${1:-false}"
    local tool_dst="$INSTALL_DIR/xray-panel.sh"
    local link_dst="/usr/bin/xray-panel"

    if [[ "$force" == "false" && -f "$tool_dst" ]]; then
        warn "管理工具已存在，跳过（使用 update-tool 可强制更新）"
        # 确保软链接存在
        ln -sf "$tool_dst" "$link_dst"
        return
    fi

    # 优先从解压目录复制（本地包安装）
    if [[ -n "${EXTRACTED_DIR:-}" && -f "$EXTRACTED_DIR/xray-panel.sh" ]]; then
        cp "$EXTRACTED_DIR/xray-panel.sh" "$tool_dst"
        ok "管理工具已从安装包复制"
    else
        # 从 GitHub master 下载最新版
        info "下载最新管理工具..."
        local url="https://raw.githubusercontent.com/$GITHUB_REPO/master/xray-panel.sh"
        local tmp_tool
        tmp_tool=$(mktemp)
        # shellcheck disable=SC2064
        trap "rm -f $tmp_tool" RETURN
        if curl -fsSL "$url" -o "$tmp_tool" && head -1 "$tmp_tool" | grep -q '^#!'; then
            mv "$tmp_tool" "$tool_dst"
            ok "管理工具已下载（最新版）"
        else
            rm -f "$tmp_tool"
            warn "管理工具下载失败，可稍后运行 update-tool 重试（不影响面板运行）"
            return
        fi
    fi

    chmod +x "$tool_dst"
    ln -sf "$tool_dst" "$link_dst"
    ok "管理工具已就绪: 输入 ${CYAN}xray-panel${PLAIN} 即可打开管理菜单"
}


cmd_install() {
    local local_pkg="${1:-}"   # optional: path to local .tar.gz
    EXTRACTED_DIR=""           # 解压根目录，供 copy_config_template / install_management_tool 使用

    need_root; detect_arch; detect_os; install_deps

    mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"

    if [[ -n "$local_pkg" ]]; then
        # ── 本地安装 ──
        [[ -f "$local_pkg" ]] || die "文件不存在: $local_pkg"
        info "从本地包安装: $local_pkg"
        local tmp
        tmp=$(mktemp -d)
        trap "rm -rf $tmp" RETURN
        tar xzf "$local_pkg" -C "$tmp"
        local bin
        bin=$(find "$tmp" \( -name "panel-linux-${ARCH}" -o -name "panel" \) -type f | head -1)
        [[ -n "$bin" ]] || die "压缩包内未找到 panel 二进制"
        chmod +x "$bin"
        # 停止旧服务 + 删除旧二进制，避免 Text file busy
        if systemctl is-active --quiet xray-panel 2>/dev/null; then
            info "停止面板服务..."
            systemctl stop xray-panel
        fi
        rm -f "$BINARY_PATH"
        cp "$bin" "$BINARY_PATH"
        # 记录解压根目录（包含 conf/ scripts/ xray-panel.sh）
        EXTRACTED_DIR=$(find "$tmp" -name "xray-panel.sh" -type f | head -1 | xargs -r dirname 2>/dev/null || echo "")
        ok "面板二进制已安装（本地包）"
    else
        # ── 在线安装 ──
        resolve_version "${PANEL_VERSION:-latest}"
        download_binary
    fi

    install_xray_core
    copy_config_template      # 复制/更新 config.yaml.example
    generate_config           # 仅在无 config.yaml 时生成
    setup_service
    install_management_tool   # 安装 xray-panel 交互管理工具
    update_geodata            # 初始安装时顺带更新一次 geo 文件

    hr
    echo -e "${GREEN}安装完成${PLAIN}"
    hr
    echo -e "  安装目录: ${CYAN}$INSTALL_DIR${PLAIN}"
    echo -e "  配置文件: ${CYAN}$CONFIG_DIR/config.yaml${PLAIN}"
    echo -e "  配置模板: ${CYAN}$CONFIG_DIR/config.yaml.example${PLAIN}"
    echo -e "  管理工具: ${CYAN}xray-panel${PLAIN}"
    echo ""
    echo -e "${YELLOW}下一步:${PLAIN}"
    echo -e "  1. 检查配置:   ${CYAN}$CONFIG_DIR/config.yaml${PLAIN}"
    echo -e "  2. 启动服务:   ${CYAN}systemctl start xray-panel${PLAIN}"
    echo -e "  3. 查看账户:   ${CYAN}cd $INSTALL_DIR && ./panel admin${PLAIN}"
    echo -e "  4. 打开管理:   ${CYAN}xray-panel${PLAIN}"
    echo ""
}

# ── 子命令: update ────────────────────────────────────────────────────────────
cmd_update() {
    local target_ver="${1:-latest}"

    need_root; detect_arch

    [[ -f "$BINARY_PATH" ]] || die "面板未安装，请先运行: $0 install"

    local cur_ver
    cur_ver=$("$BINARY_PATH" version 2>/dev/null | grep -oP 'version \K[^ ]+' || echo "unknown")
    info "当前版本: $cur_ver"

    resolve_version "$target_ver"

    if [[ "$cur_ver" == "$PANEL_VERSION" ]]; then
        warn "已是最新版本 ($cur_ver)，无需更新"
        read -rp "仍要重新安装? (y/N): " ans
        [[ "$ans" =~ ^[Yy]$ ]] || { info "已取消"; return; }
    fi

    backup

    download_binary  # 内部已处理停服 + rm -f 旧文件

    info "启动面板服务..."
    systemctl start xray-panel
    sleep 2
    systemctl is-active --quiet xray-panel || die "面板启动失败，请查看: journalctl -u xray-panel -n 50"

    local new_ver
    new_ver=$("$BINARY_PATH" version 2>/dev/null | grep -oP 'version \K[^ ]+' || echo "unknown")

    hr
    echo -e "${GREEN}更新完成${PLAIN}  ${YELLOW}$cur_ver${PLAIN} → ${GREEN}$new_ver${PLAIN}"
    hr
    echo -e "  查看日志:   ${CYAN}journalctl -u xray-panel -f${PLAIN}"
    echo ""
}

# ── 子命令: uninstall ─────────────────────────────────────────────────────────
cmd_uninstall() {
    need_root

    hr
    echo -e "${RED}即将卸载 Xray Panel${PLAIN}"
    hr
    echo -e "将删除:  面板二进制、systemd 服务、Nginx 面板配置（如有）"
    echo -e "不会删:  数据库、配置文件、日志、Xray-core、Nginx"
    echo ""
    read -rp "确认继续? (yes/no): " ans
    [[ "$ans" =~ ^[Yy][Ee][Ss]$ ]] || { info "已取消"; return; }

    backup

    # 停止并移除服务
    systemctl stop xray-panel 2>/dev/null || true
    if [[ -f "$SYSTEMD_FILE" ]]; then
        systemctl disable xray-panel 2>/dev/null || true
        rm -f "$SYSTEMD_FILE"
        systemctl daemon-reload
    fi
    ok "systemd 服务已移除"

    # 移除全局命令软链接
    rm -f /usr/bin/xray-panel
    ok "管理工具软链接已移除"

    # 清理 Nginx 面板配置（如有）
    if [[ -f /etc/nginx/conf.d/xray-panel.conf ]]; then
        rm -f /etc/nginx/conf.d/xray-panel.conf
        systemctl is-active --quiet nginx && nginx -t && systemctl reload nginx || true
        ok "Nginx 面板配置已移除"
    fi

    echo ""
    read -rp "是否同时删除所有数据和配置? (yes/no): " ans2
    if [[ "$ans2" =~ ^[Yy][Ee][Ss]$ ]]; then
        rm -rf "$INSTALL_DIR"
        ok "所有数据已删除"
    else
        info "数据已保留: $INSTALL_DIR"
    fi

    hr
    echo -e "${GREEN}卸载完成${PLAIN}"
    echo -e "如需卸载 Xray-core:"
    echo -e "  ${CYAN}bash -c \"\$(curl -L https://github.com/XTLS/Xray-install/raw/master/install-release.sh)\" @ remove${PLAIN}"
    echo ""
}

# ── 子命令: update-geo ────────────────────────────────────────────────────────
update_geodata() {
    # Can be called standalone (cmd_update_geo) or from install flow.
    local assets_dir="${1:-$XRAY_ASSETS}"
    local standalone="${2:-false}"   # true = called as subcommand, print header

    if [[ "$standalone" == "true" ]]; then
        need_root
        hr
        echo -e "更新 geoip.dat / geosite.dat"
        hr
    fi

    [[ -d "$assets_dir" ]] || { mkdir -p "$assets_dir"; }

    local -A sources=(
        ["geoip.dat"]="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
        ["geosite.dat"]="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
    )

    local updated=0
    for file in "${!sources[@]}"; do
        local url="${sources[$file]}"
        local dest="$assets_dir/$file"
        local tmp="$dest.tmp"

        info "下载 $file ..."
        if wget -q --show-progress "$url" -O "$tmp"; then
            # 比较文件大小，确保下载到了有效文件（至少 1 MB）
            local size
            size=$(stat -c%s "$tmp" 2>/dev/null || echo 0)
            if [[ "$size" -gt 1048576 ]]; then
                mv "$tmp" "$dest"
                ok "$file 已更新 ($(numfmt --to=iec "$size" 2>/dev/null || echo "${size}B"))"
                (( updated++ )) || true
            else
                rm -f "$tmp"
                warn "$file 下载文件过小 (${size}B)，可能下载失败，保留原文件"
            fi
        else
            rm -f "$tmp"
            warn "$file 下载失败，保留原文件（请检查网络）"
        fi
    done

    if [[ "$updated" -gt 0 ]]; then
        # 重启 Xray 使新 geo 文件生效
        if systemctl is-active --quiet xray 2>/dev/null; then
            info "重启 Xray 使 geo 文件生效..."
            systemctl restart xray
            ok "Xray 已重启"
        fi
    fi

    if [[ "$standalone" == "true" ]]; then
        echo ""
        echo -e "  geo 文件目录: ${CYAN}$assets_dir${PLAIN}"
        ls -lh "$assets_dir"/*.dat 2>/dev/null || true
        echo ""
    fi
}

cmd_update_geo() { update_geodata "$XRAY_ASSETS" "true"; }

# ── 子命令: update-tool ───────────────────────────────────────────────────────
cmd_update_tool() {
    need_root
    EXTRACTED_DIR=""  # 无本地包，强制从网络拉取

    hr
    echo -e "更新 xray-panel 交互管理脚本"
    hr

    local tool_dst="$INSTALL_DIR/xray-panel.sh"
    local cur_mtime=""
    [[ -f "$tool_dst" ]] && cur_mtime=$(stat -c '%y' "$tool_dst" | cut -d'.' -f1)

    install_management_tool "force"

    local new_mtime=""
    [[ -f "$tool_dst" ]] && new_mtime=$(stat -c '%y' "$tool_dst" | cut -d'.' -f1)

    if [[ -n "$cur_mtime" && "$cur_mtime" != "$new_mtime" ]]; then
        echo -e "  ${YELLOW}更新前:${PLAIN} $cur_mtime"
        echo -e "  ${GREEN}更新后:${PLAIN} $new_mtime"
    fi
    echo ""
}

# ── 子命令: status ────────────────────────────────────────────────────────────
cmd_status() {
    hr
    echo -e "Xray Panel 状态"
    hr

    # 面板服务
    if systemctl is-active --quiet xray-panel 2>/dev/null; then
        echo -e "  面板服务:  ${GREEN}运行中${PLAIN}"
    else
        echo -e "  面板服务:  ${RED}已停止${PLAIN}"
    fi

    # 面板版本
    if [[ -f "$BINARY_PATH" ]]; then
        local ver
        ver=$("$BINARY_PATH" version 2>/dev/null | grep -oP 'version \K[^ ]+' || echo "unknown")
        echo -e "  面板版本:  ${CYAN}$ver${PLAIN}"
    else
        echo -e "  面板版本:  ${YELLOW}未安装${PLAIN}"
    fi

    # Xray-core
    if systemctl is-active --quiet xray 2>/dev/null; then
        echo -e "  Xray-core: ${GREEN}运行中${PLAIN}"
    else
        echo -e "  Xray-core: ${RED}已停止${PLAIN}"
    fi

    # Geo 文件
    echo ""
    echo -e "  Geo 文件 ($XRAY_ASSETS):"
    for f in geoip.dat geosite.dat; do
        local p="$XRAY_ASSETS/$f"
        if [[ -f "$p" ]]; then
            echo -e "    ${GREEN}✓${PLAIN} $f  $(stat -c '%y' "$p" | cut -d' ' -f1)"
        else
            echo -e "    ${RED}✗${PLAIN} $f  不存在"
        fi
    done

    echo ""
    echo -e "  常用命令:"
    echo -e "    ${CYAN}journalctl -u xray-panel -f${PLAIN}     实时日志"
    echo -e "    ${CYAN}systemctl restart xray-panel${PLAIN}    重启面板"
    echo ""
}

# ── 帮助 ──────────────────────────────────────────────────────────────────────
cmd_help() {
    cat <<EOF

${CYAN}Xray Panel 管理脚本${PLAIN}

用法: $(basename "$0") <命令> [参数]

命令:
  install              在线安装（从 GitHub 自动下载最新版）
  install <pkg.tar.gz> 本地安装（指定本地压缩包）
  update [version]     更新面板二进制（默认 latest）
  update-tool          仅更新交互管理脚本 xray-panel
  uninstall            卸载面板
  update-geo           更新 geoip.dat / geosite.dat
  status               查看面板及 Geo 文件状态
  help                 显示此帮助

环境变量:
  GITHUB_REPO    GitHub 仓库（默认: nxovaeng/xray-panel）
  PANEL_VERSION  安装版本（默认: latest）

示例:
  $(basename "$0") install
  $(basename "$0") install ./xray-panel-v1.0.0-linux-amd64.tar.gz
  $(basename "$0") update v1.2.0
  $(basename "$0") update-geo
  GITHUB_REPO=myorg/xray-panel $(basename "$0") install

EOF
}

# ── 入口 ──────────────────────────────────────────────────────────────────────
main() {
    local cmd="${1:-help}"
    shift || true
    case "$cmd" in
        install)    cmd_install "$@" ;;
        update)     cmd_update  "$@" ;;
        update-tool) cmd_update_tool ;;
        uninstall)  cmd_uninstall   ;;
        update-geo) cmd_update_geo  ;;
        status)     cmd_status      ;;
        help|--help|-h) cmd_help    ;;
        *) die "未知命令: $cmd（运行 $(basename "$0") help 查看帮助）" ;;
    esac
}

main "$@"
