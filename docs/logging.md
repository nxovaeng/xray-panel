# 日志系统说明

## 概述

Xray Panel 提供了完善的日志系统，支持日志等级控制、文件输出和自动轮转。

## 日志配置

### 配置项

```yaml
log:
  # 日志等级
  level: "info"
  
  # 日志文件路径（留空则只输出到控制台）
  file: "logs/panel.log"
  
  # 单个日志文件最大大小（MB）
  max_size: 100
  
  # 保留的旧日志文件数量
  max_backups: 10
  
  # 保留旧日志文件的最大天数
  max_age: 30
  
  # 是否压缩旧日志文件
  compress: true
```

---

## 日志等级

### 1. DEBUG（调试）

**用途**: 开发和调试

**包含信息**:
- 详细的函数调用
- 变量值
- 执行流程
- 所有其他等级的日志

**示例**:
```
[DEBUG] 2026/01/29 10:00:00 main.go:123: Processing user request: user_id=123
[DEBUG] 2026/01/29 10:00:00 auth.go:45: Token validation started
[DEBUG] 2026/01/29 10:00:00 auth.go:67: Token is valid, expires at 2026-01-30
```

**适用场景**:
- 本地开发
- 问题排查
- 性能分析

---

### 2. INFO（信息）- 推荐

**用途**: 生产环境默认等级

**包含信息**:
- 服务启动/停止
- 配置加载
- 重要操作
- 警告和错误

**示例**:
```
[INFO]  2026/01/29 10:00:00 Logger initialized with level: info
[INFO]  2026/01/29 10:00:00 Logging to file: /var/log/xray-panel/panel.log
[INFO]  2026/01/29 10:00:00 Using config file: /etc/xray-panel/config.yaml
[INFO]  2026/01/29 10:00:00 Starting xray-panel on 0.0.0.0:8082
```

**适用场景**:
- 生产环境
- 日常运维
- 审计追踪

---

### 3. WARNING（警告）

**用途**: 只记录警告和错误

**包含信息**:
- 潜在问题
- 降级操作
- 错误

**示例**:
```
[WARN]  2026/01/29 10:00:00 Failed to connect to Xray API, using file deployment
[WARN]  2026/01/29 10:00:00 User traffic limit exceeded: user_id=123
[ERROR] 2026/01/29 10:00:00 Database connection failed: timeout
```

**适用场景**:
- 稳定的生产环境
- 减少日志量
- 只关注问题

---

### 4. ERROR（错误）

**用途**: 只记录错误

**包含信息**:
- 错误信息
- 堆栈跟踪

**示例**:
```
[ERROR] 2026/01/29 10:00:00 main.go:456: Server error: listen tcp :8082: bind: address already in use
[ERROR] 2026/01/29 10:00:00 database.go:123: Failed to initialize database: file not found
```

**适用场景**:
- 最小化日志
- 只关注严重问题
- 磁盘空间有限

---

## 日志输出

### 控制台输出

**默认行为**: 日志始终输出到控制台（stdout）

**查看方式**:
```bash
# 直接运行
./panel

# systemd 服务
journalctl -u xray-panel -f

# Docker
docker logs -f xray-panel
```

---

### 文件输出

**配置**:
```yaml
log:
  file: "/var/log/xray-panel/panel.log"
```

**特性**:
- 同时输出到控制台和文件
- 自动创建目录
- 自动轮转（达到大小限制时）
- 可选压缩

**查看方式**:
```bash
# 实时查看
tail -f /var/log/xray-panel/panel.log

# 查看最后 100 行
tail -n 100 /var/log/xray-panel/panel.log

# 搜索错误
grep ERROR /var/log/xray-panel/panel.log

# 搜索特定用户
grep "user_id=123" /var/log/xray-panel/panel.log
```

---

## 日志轮转

### 自动轮转

当日志文件达到 `max_size` 时，自动轮转：

```
panel.log           # 当前日志
panel.log.1         # 最近的备份
panel.log.2         # 较旧的备份
panel.log.3.gz      # 压缩的旧日志
```

### 轮转规则

1. **大小限制**: 达到 `max_size` MB 时轮转
2. **数量限制**: 保留 `max_backups` 个备份文件
3. **时间限制**: 删除超过 `max_age` 天的文件
4. **压缩**: 如果 `compress: true`，旧文件会被 gzip 压缩

### 示例配置

**开发环境**（快速轮转）:
```yaml
log:
  file: "logs/panel.log"
  max_size: 10      # 10 MB
  max_backups: 3    # 保留 3 个备份
  max_age: 7        # 保留 7 天
  compress: false   # 不压缩
```

**生产环境**（长期保留）:
```yaml
log:
  file: "/var/log/xray-panel/panel.log"
  max_size: 100     # 100 MB
  max_backups: 30   # 保留 30 个备份
  max_age: 90       # 保留 90 天
  compress: true    # 压缩旧文件
```

---

## 不同环境的配置

### Windows 开发环境

```yaml
log:
  level: "debug"
  file: "logs/panel.log"
  max_size: 50
  max_backups: 3
  max_age: 7
  compress: false
```

**特点**:
- 使用相对路径
- DEBUG 等级
- 快速轮转
- 不压缩（方便查看）

---

### Linux 生产环境

```yaml
log:
  level: "info"
  file: "/var/log/xray-panel/panel.log"
  max_size: 100
  max_backups: 30
  max_age: 90
  compress: true
```

**特点**:
- 使用绝对路径
- INFO 等级
- 长期保留
- 压缩节省空间

---

### Docker 环境

```yaml
log:
  level: "info"
  file: ""  # 留空，只输出到控制台
```

**特点**:
- 不写文件
- 通过 Docker 日志系统管理
- 使用 `docker logs` 查看

---

## 日志格式

### 标准格式

```
[LEVEL] YYYY/MM/DD HH:MM:SS [file:line:] message
```

### 示例

```
[INFO]  2026/01/29 10:00:00 Server started successfully
[DEBUG] 2026/01/29 10:00:01 auth.go:45: Validating token
[WARN]  2026/01/29 10:00:02 High memory usage: 85%
[ERROR] 2026/01/29 10:00:03 database.go:123: Connection failed
```

### 字段说明

- `[LEVEL]`: 日志等级
- `YYYY/MM/DD HH:MM:SS`: 时间戳
- `file:line`: 源代码位置（DEBUG 和 ERROR 等级）
- `message`: 日志消息

---

## 常见使用场景

### 场景 1: 排查启动问题

```bash
# 查看启动日志
tail -n 50 /var/log/xray-panel/panel.log | grep -E "INFO|ERROR"

# 查找错误
grep ERROR /var/log/xray-panel/panel.log
```

### 场景 2: 监控运行状态

```bash
# 实时监控
tail -f /var/log/xray-panel/panel.log

# 只看错误和警告
tail -f /var/log/xray-panel/panel.log | grep -E "WARN|ERROR"
```

### 场景 3: 审计用户操作

```bash
# 查找特定用户的操作
grep "user_id=123" /var/log/xray-panel/panel.log

# 查找登录记录
grep "login" /var/log/xray-panel/panel.log
```

### 场景 4: 性能分析

```bash
# 临时启用 DEBUG 等级
# 修改配置文件
vim /etc/xray-panel/config.yaml
# 将 level 改为 "debug"

# 重启服务
systemctl restart xray-panel

# 分析日志
grep "Processing time" /var/log/xray-panel/panel.log
```

---

## 日志管理最佳实践

### 1. 选择合适的日志等级

- **开发**: DEBUG
- **测试**: INFO
- **生产**: INFO 或 WARNING
- **稳定生产**: WARNING 或 ERROR

### 2. 配置合理的轮转策略

```yaml
# 根据磁盘空间和保留需求调整
max_size: 100      # 单文件大小
max_backups: 30    # 备份数量
max_age: 90        # 保留天数
```

**估算磁盘使用**:
```
总大小 = max_size × (max_backups + 1)
示例: 100 MB × 31 = 3.1 GB
```

### 3. 定期检查日志

```bash
# 每周检查一次
crontab -e

# 添加定时任务
0 0 * * 0 grep ERROR /var/log/xray-panel/panel.log | mail -s "Weekly Error Report" admin@example.com
```

### 4. 备份重要日志

```bash
# 备份脚本
#!/bin/bash
tar -czf /backup/panel-logs-$(date +%Y%m%d).tar.gz /var/log/xray-panel/
```

### 5. 监控日志文件大小

```bash
# 检查日志目录大小
du -sh /var/log/xray-panel/

# 查找大文件
find /var/log/xray-panel/ -type f -size +100M
```

---

## 故障排查

### 问题 1: 日志文件未创建

**原因**: 目录不存在或权限不足

**解决**:
```bash
# 创建目录
mkdir -p /var/log/xray-panel

# 设置权限
chown xray-panel:xray-panel /var/log/xray-panel
chmod 755 /var/log/xray-panel
```

### 问题 2: 日志文件过大

**原因**: 日志等级太低或轮转配置不当

**解决**:
```yaml
# 调整配置
log:
  level: "warning"  # 提高等级
  max_size: 50      # 减小单文件大小
  max_backups: 5    # 减少备份数量
```

### 问题 3: 找不到日志

**检查**:
```bash
# 查看配置
cat /etc/xray-panel/config.yaml | grep -A 10 "log:"

# 查看进程
ps aux | grep panel

# 查看 systemd 日志
journalctl -u xray-panel -n 50
```

---

## 总结

- ✅ 支持 4 个日志等级（DEBUG、INFO、WARNING、ERROR）
- ✅ 同时输出到控制台和文件
- ✅ 自动日志轮转和压缩
- ✅ 灵活的配置选项
- ✅ 适配不同环境（开发、生产、Docker）
- ✅ 详细的时间戳和源代码位置
- ✅ 易于搜索和分析

**推荐配置**:
- 开发环境: DEBUG + 文件
- 生产环境: INFO + 文件 + 轮转
- Docker 环境: INFO + 控制台


## 记录的操作

系统会记录以下关键操作：

### 认证相关
- ✅ 管理员登录成功
- ⚠️ 登录失败（用户名不存在或密码错误）
- ✅ 管理员登出
- ✅ 管理员账户创建
- ✅ 密码重置

### 用户管理
- ✅ 用户创建（包含邮箱和 UUID）
- ✅ 用户更新
- ✅ 用户删除
- ❌ 操作失败（包含错误详情）

### 入站/出站管理
- ✅ 入站节点创建（包含标签、协议、端口）
- ✅ 入站节点删除
- ✅ 出站节点创建（包含标签、类型）
- ✅ 出站节点更新
- ❌ 操作失败

### 配置管理
- ✅ Xray 配置应用
- ✅ Xray 服务重启
- ✅ Nginx 配置重载
- ⚠️ 服务重启失败

### 系统启动
- ✅ 日志系统初始化
- ✅ 数据库连接
- ✅ 服务器启动
- ✅ 配置文件加载

## 安全注意事项

⚠️ **重要**: 日志文件可能包含敏感信息：

- 不要记录密码（已在代码中避免）
- 不要记录完整的 JWT token
- 注意日志文件的访问权限
- 定期清理旧日志文件
- 考虑加密存储敏感日志

## 集成监控系统

日志格式兼容常见的日志收集工具：

- **ELK Stack** (Elasticsearch, Logstash, Kibana)
- **Grafana Loki**
- **Fluentd**
- **Prometheus** (通过日志导出器)

示例 Logstash 配置：

```ruby
input {
  file {
    path => "/var/log/xray-panel/panel.log"
    start_position => "beginning"
  }
}

filter {
  grok {
    match => { "message" => "\[%{WORD:level}\]\s+%{TIMESTAMP_ISO8601:timestamp}\s+%{GREEDYDATA:message}" }
  }
}

output {
  elasticsearch {
    hosts => ["localhost:9200"]
    index => "xray-panel-%{+YYYY.MM.dd}"
  }
}
```

## 更新日志

### v1.0.0 (2026-01-29)
- ✅ 实现基础日志系统
- ✅ 支持 4 个日志级别
- ✅ 支持文件输出和自动轮转
- ✅ 集成到所有模块（API、Web、Database）
- ✅ 添加关键操作日志记录
- ✅ 支持动态调整日志级别
