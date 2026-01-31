# Nginx 配置追踪功能

## 概述

面板会自动追踪生成的 Nginx 配置文件，特别是为通配符证书生成的随机子域名配置。这样在更新或删除 Inbound 时，可以自动清理相关的 Nginx 配置文件。

## 数据库表结构

### nginx_configs 表

```sql
CREATE TABLE nginx_configs (
    id          TEXT PRIMARY KEY,
    inbound_id  TEXT,           -- 关联的 Inbound ID
    domain      TEXT,           -- 配置的域名（可能是生成的子域名）
    config_path TEXT,           -- Nginx 配置文件路径
    config_type TEXT DEFAULT 'http',  -- http, stream, panel
    is_managed  BOOLEAN DEFAULT true, -- 是否由面板管理
    created_at  DATETIME,
    updated_at  DATETIME,
    
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id)
);
```

## 工作流程

### 1. 创建 Inbound（通配符证书）

当创建使用通配符证书的 Inbound 时：

```
用户选择域名: *.example.com
↓
生成随机子域名: abc123.example.com
↓
保存到 Inbound.ActualDomain
↓
生成 Nginx 配置: /etc/nginx/conf.d/abc123.example.com.conf
↓
记录到 nginx_configs 表:
  - inbound_id: <inbound-uuid>
  - domain: abc123.example.com
  - config_path: /etc/nginx/conf.d/abc123.example.com.conf
  - config_type: http
```

### 2. 更新 Inbound

更新 Inbound 时：

```
如果域名改变:
  ↓
  清理旧的 Nginx 配置
  ↓
  生成新的 Nginx 配置
  ↓
  更新 nginx_configs 记录
```

### 3. 删除 Inbound

删除 Inbound 时：

```
查询 nginx_configs 表
↓
找到所有关联的配置文件
↓
检查文件是否有 "# Managed by Xray Panel" 标记
↓
删除配置文件
↓
删除 nginx_configs 记录
↓
重载 Nginx
```

## 配置文件标记

所有由面板生成的 Nginx 配置文件都会在第一行添加标记：

```nginx
# Managed by Xray Panel
server {
    listen 80;
    ...
}
```

这个标记用于：
- 防止覆盖手动创建的配置
- 确认文件可以安全删除
- 识别面板管理的配置

## API 接口

### 查询 Inbound 的 Nginx 配置

```go
GET /api/inbounds/:id/nginx-configs
```

响应：
```json
{
  "code": 0,
  "data": [
    {
      "id": "config-uuid",
      "inbound_id": "inbound-uuid",
      "domain": "abc123.example.com",
      "config_path": "/etc/nginx/conf.d/abc123.example.com.conf",
      "config_type": "http",
      "is_managed": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### 清理 Inbound 的 Nginx 配置

```go
DELETE /api/inbounds/:id/nginx-configs
```

## 使用示例

### 代码示例

```go
// 创建 Nginx 配置生成器
ng := nginx.NewGenerator("/etc/nginx/conf.d", "/etc/nginx/stream.d")
ng.SetDB(db)  // 设置数据库连接以启用追踪

// 生成配置（会自动记录）
err := ng.GenerateHTTPConfig(inbounds)

// 清理特定 Inbound 的配置
err := ng.CleanupInboundConfigs(inboundID)
```

### 命令行工具

```bash
# 查看所有追踪的配置
./panel nginx list-tracked

# 清理孤立的配置（Inbound 已删除但配置文件还在）
./panel nginx cleanup-orphaned

# 验证配置文件
./panel nginx verify
```

## 通配符证书示例

### 场景

证书：`*.example.com`

创建 3 个 Inbound：

1. **Inbound 1**
   - 生成子域名：`abc123.example.com`
   - 配置文件：`/etc/nginx/conf.d/abc123.example.com.conf`

2. **Inbound 2**
   - 生成子域名：`def456.example.com`
   - 配置文件：`/etc/nginx/conf.d/def456.example.com.conf`

3. **Inbound 3**
   - 生成子域名：`ghi789.example.com`
   - 配置文件：`/etc/nginx/conf.d/ghi789.example.com.conf`

### 数据库记录

```
nginx_configs 表:
┌──────────────┬──────────────┬─────────────────────────┬────────────────────────────────────────┐
│ inbound_id   │ domain       │ config_path                                                     │
├──────────────┼──────────────┼─────────────────────────┼────────────────────────────────────────┤
│ inbound-1-id │ abc123.ex... │ /etc/nginx/conf.d/abc123.example.com.conf                      │
│ inbound-2-id │ def456.ex... │ /etc/nginx/conf.d/def456.example.com.conf                      │
│ inbound-3-id │ ghi789.ex... │ /etc/nginx/conf.d/ghi789.example.com.conf                      │
└──────────────┴──────────────┴─────────────────────────┴────────────────────────────────────────┘
```

### 删除 Inbound 1

```
1. 查询 nginx_configs WHERE inbound_id = 'inbound-1-id'
2. 找到配置文件: /etc/nginx/conf.d/abc123.example.com.conf
3. 检查文件第一行是否包含 "# Managed by Xray Panel"
4. 删除文件
5. 删除数据库记录
6. 重载 Nginx
```

结果：
- ✅ `abc123.example.com.conf` 已删除
- ✅ `def456.example.com.conf` 保留
- ✅ `ghi789.example.com.conf` 保留

## 安全性

### 防止误删

1. **管理标记检查**：只删除包含 `# Managed by Xray Panel` 的文件
2. **数据库记录**：只删除数据库中有记录的文件
3. **is_managed 标志**：可以手动标记某些配置为不可删除

### 手动配置保护

如果用户手动创建了配置文件（没有管理标记），面板会：
- 拒绝覆盖该文件
- 在日志中警告
- 不会删除该文件

## 故障排查

### 配置文件未被追踪

**原因**：
- 数据库连接未设置：`ng.SetDB(db)` 未调用
- 配置生成失败
- 数据库写入失败

**解决方案**：
```bash
# 查看日志
tail -f /opt/xray-panel/logs/panel.log

# 手动添加追踪记录
./panel nginx track-config --inbound-id <id> --domain <domain> --path <path>
```

### 孤立的配置文件

**原因**：
- Inbound 被直接从数据库删除
- 配置生成过程中断
- 数据库记录丢失

**解决方案**：
```bash
# 清理孤立的配置
./panel nginx cleanup-orphaned

# 或手动删除
rm /etc/nginx/conf.d/<orphaned-config>.conf
systemctl reload nginx
```

### 配置文件无法删除

**原因**：
- 文件权限问题
- 文件被其他进程占用
- 文件不存在但数据库有记录

**解决方案**：
```bash
# 检查文件权限
ls -la /etc/nginx/conf.d/

# 强制清理数据库记录
./panel nginx cleanup-db --inbound-id <id>
```

## 最佳实践

1. **始终通过面板管理配置**：不要手动编辑面板生成的配置文件
2. **定期清理**：定期运行 `cleanup-orphaned` 清理孤立配置
3. **备份配置**：在大量操作前备份 Nginx 配置目录
4. **监控日志**：关注配置生成和删除的日志
5. **测试配置**：生成配置后使用 `nginx -t` 测试

## 相关文件

- `internal/models/nginx_config.go` - 数据模型
- `internal/nginx/config.go` - 配置生成和追踪
- `internal/database/database.go` - 数据库迁移
- `docs/nginx-config-tracking.md` - 本文档
