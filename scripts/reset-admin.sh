#!/bin/bash

# Xray Panel - 管理员密码重置脚本
# 用法: ./reset-admin.sh [username] [new_password]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
DB_PATH="/etc/xray-panel/panel.db"
CONFIG_FILE="/etc/xray-panel/config.yaml"

# 从配置文件读取数据库路径
if [ -f "$CONFIG_FILE" ]; then
    DB_PATH=$(grep "path:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
fi

# 检查数据库是否存在
if [ ! -f "$DB_PATH" ]; then
    echo -e "${RED}错误: 数据库文件不存在: $DB_PATH${NC}"
    exit 1
fi

# 获取参数
USERNAME=${1:-}
NEW_PASSWORD=${2:-}

# 如果没有提供参数，交互式输入
if [ -z "$USERNAME" ]; then
    echo -e "${YELLOW}请输入管理员用户名:${NC}"
    read -r USERNAME
fi

if [ -z "$NEW_PASSWORD" ]; then
    echo -e "${YELLOW}请输入新密码:${NC}"
    read -rs NEW_PASSWORD
    echo
    echo -e "${YELLOW}请再次输入新密码:${NC}"
    read -rs NEW_PASSWORD_CONFIRM
    echo
    
    if [ "$NEW_PASSWORD" != "$NEW_PASSWORD_CONFIRM" ]; then
        echo -e "${RED}错误: 两次输入的密码不一致${NC}"
        exit 1
    fi
fi

# 检查密码长度
if [ ${#NEW_PASSWORD} -lt 8 ]; then
    echo -e "${RED}错误: 密码长度至少为 8 个字符${NC}"
    exit 1
fi

echo -e "${YELLOW}正在重置密码...${NC}"

# 使用 Go 程序重置密码
# 这里需要编译一个小工具或者使用 panel 的子命令
# 暂时使用 SQL 直接操作（需要 bcrypt 哈希）

echo -e "${YELLOW}注意: 此脚本需要 panel 程序支持 reset-password 命令${NC}"
echo -e "${YELLOW}请使用以下命令重置密码:${NC}"
echo -e "${GREEN}./panel reset-password --username=$USERNAME --password=$NEW_PASSWORD${NC}"

exit 0
