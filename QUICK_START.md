# Xray Panel 快速开始

## 一键安装

```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/master/scripts/install-online.sh)
```

## 启动服务

```bash
systemctl start xray-panel
```

## 查看管理员账户

```bash
cd /opt/xray-panel
./panel -show-admin
```

## 访问面板

默认地址：`http://your-server-ip:8082`

## 配置 HTTPS（推荐）

### 1. 配置 Nginx 反向代理

```bash
xray-panel
# 选择 19. 配置 Nginx 反向代理
```

### 2. 申请 SSL 证书

```bash
xray-panel
# 选择 20. 申请 SSL 证书
```

## 管理脚本

运行 `xray-panel` 打开便捷管理菜单：

```bash
xray-panel
```

## 常用命令

```bash
# 启动面板
systemctl start xray-panel

# 停止面板
systemctl stop xray-panel

# 重启面板
systemctl restart xray-panel

# 查看状态
systemctl status xray-panel

# 查看日志
journalctl -u xray-panel -f

# 重置密码
cd /opt/xray-panel
./panel -reset-password -username=admin_xxx -password=NewPassword123
```

## 目录结构

```
/opt/xray-panel/
├── panel              # 主程序
├── conf/              # 配置文件
│   └── config.yaml
├── data/              # 数据库
│   └── panel.db
└── logs/              # 日志
    └── panel.log
```

## 更新面板

```bash
xray-panel
# 选择 2. 更新 Xray Panel
```

## 备份数据

```bash
xray-panel
# 选择 23. 备份数据
```

## 详细文档

- [完整安装指南](docs/installation-guide.md)
- [配置文件说明](docs/configuration.md)
- [CLI 命令使用](docs/cli-commands.md)

## 获取帮助

- 文档: https://github.com/nxovaeng/xray-panel/tree/master/docs
