# 用户与入站的关系说明

## 设计原理

### 核心概念

在 Xray Panel 中，**用户和入站是多对多的关系**，但这个关系是**自动管理**的，不需要手动配置。

```
┌─────────────┐         ┌─────────────┐
│   用户 1    │         │   入站 1    │
│  (UUID-A)   │◄───────►│  (端口 10001)│
└─────────────┘    │    └─────────────┘
                   │
┌─────────────┐    │    ┌─────────────┐
│   用户 2    │    └───►│   入站 2    │
│  (UUID-B)   │◄───────►│  (端口 10002)│
└─────────────┘    │    └─────────────┘
                   │
┌─────────────┐    │    ┌─────────────┐
│   用户 3    │    └───►│   入站 3    │
│  (UUID-C)   │◄────────│  (端口 10003)│
└─────────────┘         └─────────────┘
```

### 自动关联机制

**所有活跃用户会自动添加到所有启用的入站中**

这意味着：
- ✅ 创建用户后，无需手动配置入站
- ✅ 用户可以通过任何入站连接
- ✅ 添加新入站时，所有用户自动可用
- ✅ 禁用用户后，自动从所有入站移除

## 工作流程

### 1. 创建用户

```
管理员创建用户
    ↓
生成 UUID
    ↓
用户状态：启用
    ↓
自动添加到所有入站
```

**示例**：
```
用户：张三
UUID: 12345678-1234-1234-1234-123456789abc
状态：启用

→ 自动添加到：
  - 入站 1 (VLESS + WS)
  - 入站 2 (VLESS + gRPC)
  - 入站 3 (Trojan + XHTTP)
```

### 2. 创建入站

```
管理员创建入站
    ↓
配置协议和传输
    ↓
入站状态：启用
    ↓
自动包含所有活跃用户
```

**示例**：
```
入站：新节点
协议：VLESS
传输：WebSocket
端口：10004

→ 自动包含所有用户：
  - 用户 1 (UUID-A)
  - 用户 2 (UUID-B)
  - 用户 3 (UUID-C)
```

### 3. 生成订阅

```
用户访问订阅链接
    ↓
系统查询所有启用的入站
    ↓
为每个入站生成连接链接
    ↓
返回聚合订阅
```

**示例订阅内容**：
```
vless://UUID-A@domain1.com:443?...#节点1-VLESS-WS
vless://UUID-A@domain2.com:443?...#节点2-VLESS-gRPC
trojan://UUID-A@domain3.com:443?...#节点3-Trojan-XHTTP
```

## Xray 配置生成

### VLESS 入站配置

```json
{
  "tag": "inbound-vless-ws",
  "port": 10001,
  "protocol": "vless",
  "settings": {
    "clients": [
      {
        "id": "UUID-A",
        "email": "user1@example.com"
      },
      {
        "id": "UUID-B",
        "email": "user2@example.com"
      },
      {
        "id": "UUID-C",
        "email": "user3@example.com"
      }
    ],
    "decryption": "none"
  }
}
```

### Trojan 入站配置

```json
{
  "tag": "inbound-trojan-xhttp",
  "port": 10002,
  "protocol": "trojan",
  "settings": {
    "clients": [
      {
        "password": "UUID-A",
        "email": "user1@example.com"
      },
      {
        "password": "UUID-B",
        "email": "user2@example.com"
      },
      {
        "password": "UUID-C",
        "email": "user3@example.com"
      }
    ]
  }
}
```

**注意**：Trojan 使用 `password` 字段，但我们使用 UUID 作为密码，保持一致性。

## 用户管理

### 启用/禁用用户

**禁用用户**：
```
用户状态：启用 → 禁用
    ↓
重新生成 Xray 配置
    ↓
该用户从所有入站移除
    ↓
用户无法连接
```

**重新启用**：
```
用户状态：禁用 → 启用
    ↓
重新生成 Xray 配置
    ↓
该用户添加到所有入站
    ↓
用户可以连接
```

### 删除用户

```
删除用户
    ↓
从数据库移除
    ↓
重新生成 Xray 配置
    ↓
该用户从所有入站移除
```

## 入站管理

### 启用/禁用入站

**禁用入站**：
```
入站状态：启用 → 禁用
    ↓
重新生成 Xray 配置
    ↓
该入站不再监听
    ↓
不影响其他入站
```

**重新启用**：
```
入站状态：禁用 → 启用
    ↓
重新生成 Xray 配置
    ↓
该入站开始监听
    ↓
自动包含所有活跃用户
```

### 删除入站

```
删除入站
    ↓
从数据库移除
    ↓
重新生成 Xray 配置
    ↓
该入站停止服务
    ↓
用户订阅中移除该节点
```

## 常见问题

### Q1: 如何为特定用户创建专用入站？

**A**: 当前设计不支持用户与入站的一对一绑定。如果需要这个功能，可以通过路由规则实现：

```
创建入站 → 创建路由规则（类型：入站）→ 指定出站
```

这样可以实现：
- 特定入站的流量走特定出站
- 间接实现用户分流

### Q2: 为什么不能选择哪些用户可以使用哪个入站？

**A**: 这是 Xray 的标准设计模式：
- ✅ **简化管理**：不需要为每个用户配置入站
- ✅ **灵活性**：用户可以使用任何节点
- ✅ **可扩展性**：添加节点时自动对所有用户生效
- ✅ **订阅聚合**：一个订阅包含所有可用节点

### Q3: 如何限制某些用户只能使用特定节点？

**A**: 有两种方案：

**方案 1：使用多个面板实例**
- 为不同用户组部署独立的面板
- 每个面板有自己的入站配置

**方案 2：使用路由规则**
- 创建不同的入站标签
- 使用路由规则将特定入站路由到特定出站
- 通过出站控制访问权限

### Q4: 创建用户后需要做什么？

**A**: 什么都不需要做！
1. 创建用户（填写用户名、邮箱、UUID）
2. 系统自动将用户添加到所有入站
3. 用户可以立即使用订阅链接

### Q5: 创建入站后需要做什么？

**A**: 只需要配置 Nginx 反向代理：
1. 创建入站（选择协议、传输、端口、域名）
2. 系统自动包含所有活跃用户
3. 配置 Nginx 反向代理到该端口
4. 用户订阅会自动更新

## 最佳实践

### 1. 用户命名

使用有意义的用户名：
```
✅ 好的命名：
- zhangsan
- user001
- vip-member-01

❌ 不好的命名：
- 张三 (中文可能导致文件名问题)
- user@#$ (特殊字符)
- (空) (没有名称)
```

### 2. 入站规划

根据用途创建不同的入站：
```
入站 1: VLESS + WebSocket (通用)
入站 2: VLESS + gRPC (高性能)
入站 3: VLESS + XHTTP (推荐)
入站 4: Trojan + WebSocket (兼容性)
```

### 3. 域名管理

为每个入站配置独立域名：
```
node1.example.com → 入站 1
node2.example.com → 入站 2
node3.example.com → 入站 3
```

### 4. 订阅更新

提醒用户定期更新订阅：
- 添加新节点后，用户需要更新订阅
- 修改入站配置后，用户需要更新订阅
- 建议设置自动更新（24小时）

## 技术细节

### 代码实现

**获取活跃用户** (`internal/xray/config.go`):
```go
func (g *Generator) getActiveUsers() []models.User {
    var users []models.User
    g.db.Where("status = ?", "active").Find(&users)
    return users
}
```

**生成 VLESS 配置** (`internal/xray/inbounds.go`):
```go
func (g *Generator) generateVLESSSettings() map[string]interface{} {
    clients := make([]map[string]interface{}, 0)
    for _, user := range g.getActiveUsers() {
        client := map[string]interface{}{
            "id":    user.UUID,
            "email": user.Email,
            "level": 0,
        }
        clients = append(clients, client)
    }
    return map[string]interface{}{
        "clients":    clients,
        "decryption": "none",
    }
}
```

**生成订阅** (`internal/api/subscription.go`):
```go
func (s *Server) handleSubscription(c *gin.Context) {
    // 获取用户
    var user models.User
    db.Where("sub_path = ?", path).First(&user)
    
    // 获取所有启用的入站
    var inbounds []models.Inbound
    db.Where("enabled = ?", true).Find(&inbounds)
    
    // 为每个入站生成链接
    for _, inbound := range inbounds {
        link := generateVLESSLink(user, inbound)
        links = append(links, link)
    }
}
```

## 总结

- ✅ **自动管理**：用户和入站自动关联，无需手动配置
- ✅ **简单易用**：创建用户或入站后立即生效
- ✅ **灵活扩展**：添加节点时所有用户自动可用
- ✅ **订阅聚合**：一个订阅包含所有可用节点
- ✅ **统一管理**：通过启用/禁用控制访问权限

这种设计符合 Xray 的最佳实践，简化了管理流程，提高了系统的可维护性。
