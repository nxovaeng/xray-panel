# 证书扫描功能说明

## 支持的证书目录结构

面板支持两种主流的证书管理工具：

### 1. acme.sh

**目录结构**：
```
/root/.acme.sh/
├── example.com/                 # 普通证书（RSA 默认）
│   ├── example.com.cer          # 证书文件
│   ├── example.com.key          # 私钥文件
│   ├── fullchain.cer            # 完整证书链（推荐）
│   ├── ca.cer                   # CA 证书
│   └── example.com.conf         # 配置文件
├── example.com_ecc/             # ECC 证书
│   ├── example.com_ecc.cer
│   ├── example.com_ecc.key
│   └── fullchain.cer
├── example.com_rsa/             # RSA 证书（显式指定）
│   ├── example.com_rsa.cer
│   ├── example.com_rsa.key
│   └── fullchain.cer
├── nkym.qzz.io_ecc/            # 实际示例：带 ECC 后缀
│   ├── nkym.qzz.io_ecc.cer
│   ├── nkym.qzz.io_ecc.key
│   └── fullchain.cer
├── acme.sh                      # acme.sh 程序（会被跳过）
├── acme.sh.env                  # 环境配置（会被跳过）
├── account.conf                 # 账户配置（会被跳过）
├── ca/                          # CA 目录（会被跳过）
├── deploy/                      # 部署脚本（会被跳过）
├── dnsapi/                      # DNS API（会被跳过）
├── http.header                  # HTTP 头（会被跳过）
└── notify/                      # 通知脚本（会被跳过）
```

**配置方法**：
```yaml
nginx:
  cert_dir: "/root/.acme.sh"
```

**文件识别规则**：
- 证书文件：优先使用 `fullchain.cer`，其次使用 `<dirname>.cer`
- 私钥文件：`<dirname>.key`
- 域名提取：自动去除 `_ecc` 或 `_rsa` 后缀
  - `example.com_ecc` → `example.com`
  - `nkym.qzz.io_ecc` → `nkym.qzz.io`
- 通配符检测：通过解析证书内容判断（检查 SAN 和 CN 字段）

**跳过的目录**：
- `acme.sh` - acme.sh 程序本身
- `acme.sh.env` - 环境配置
- `account.conf` - 账户配置
- `ca` - CA 证书目录
- `deploy` - 部署脚本目录
- `dnsapi` - DNS API 脚本目录
- `http.header` - HTTP 头配置
- `notify` - 通知脚本目录

### 2. Let's Encrypt (certbot)

**目录结构**：
```
/etc/letsencrypt/live/
├── example.com/
│   ├── fullchain.pem            # 完整证书链
│   ├── privkey.pem              # 私钥文件
│   ├── cert.pem                 # 证书文件
│   ├── chain.pem                # CA 证书链
│   └── README                   # 说明文件
└── example.com-0001/            # 续期后的新证书
    ├── fullchain.pem
    └── privkey.pem
```

**配置方法**：
```yaml
nginx:
  cert_dir: "/etc/letsencrypt/live"
```

**文件识别规则**：
- 证书文件：`fullchain.pem`
- 私钥文件：`privkey.pem`

### 3. 自定义目录 (/etc/certs)

如果使用 acme.sh 安装证书到自定义目录：

```bash
# 安装证书到 /etc/certs
acme.sh --install-cert -d example.com \
  --cert-file /etc/certs/example.com/cert.pem \
  --key-file /etc/certs/example.com/key.pem \
  --fullchain-file /etc/certs/example.com/fullchain.pem
```

**目录结构**：
```
/etc/certs/
├── example.com/
│   ├── fullchain.pem
│   ├── cert.pem
│   └── key.pem
└── example2.com/
    ├── fullchain.pem
    └── key.pem
```

**配置方法**：
```yaml
nginx:
  cert_dir: "/etc/certs"
```

## 扫描逻辑

面板会按以下顺序查找证书文件：

### 1. 目录过滤

**acme.sh 目录**（路径包含 `.acme.sh` 或 `acme.sh`）：
- 跳过 acme.sh 程序目录：`acme.sh`, `acme.sh.env`, `account.conf`, `ca`, `deploy`, `dnsapi`, `http.header`, `notify`
- 扫描其他所有子目录

**非 acme.sh 目录**（如 Let's Encrypt）：
- 跳过：`README`, `ca`
- 扫描其他所有子目录

### 2. 域名提取

从目录名提取实际域名：
- `example.com` → `example.com`
- `example.com_ecc` → `example.com`（去除 ECC 后缀）
- `example.com_rsa` → `example.com`（去除 RSA 后缀）
- `nkym.qzz.io_ecc` → `nkym.qzz.io`

### 3. 证书文件查找

**acme.sh 格式**：
- 证书：`fullchain.cer` → `<dirname>.cer`
- 私钥：`<dirname>.key`
- 注意：使用完整目录名（包含后缀）查找文件

**Let's Encrypt 格式**：
- 证书：`fullchain.pem`
- 私钥：`privkey.pem`

### 4. 通配符证书检测

**重要**：通配符证书只能通过解析证书内容来判断，不能从目录名或文件名判断。

检测方法：
1. 解析证书文件（PEM 格式）
2. 检查 Subject Alternative Names (SAN) 字段
3. 检查 Common Name (CN) 字段
4. 如果任一字段包含 `*.` 前缀，则为通配符证书

示例：
- 证书 SAN 包含 `*.example.com` → 通配符证书
- 证书 CN 为 `*.example.com` → 通配符证书
- 目录名为 `example.com_ecc`，但证书内容为 `*.example.com` → 通配符证书

## 配置示例

### 完整配置文件示例

```yaml
# config.yaml
server:
  listen: "127.0.0.1:8082"

database:
  path: "/opt/xray-panel/data/panel.db"

xray:
  binary_path: "/usr/local/bin/xray"
  config_path: "/usr/local/etc/xray/config.json"
  assets_path: "/usr/local/share/xray"
  api_port: 10085

nginx:
  config_dir: "/etc/nginx/conf.d"
  stream_dir: "/etc/nginx/stream.d"
  reload_cmd: "systemctl reload nginx"
  cert_dir: "/root/.acme.sh"        # 使用 acme.sh
  # cert_dir: "/etc/letsencrypt/live" # 或使用 Let's Encrypt
  # cert_dir: "/etc/certs"            # 或使用自定义目录

log:
  level: "info"
  file: "/opt/xray-panel/logs/panel.log"
```

## 常见问题

### 1. 配置 /root/.acme.sh 后扫描没结果

**原因**：
- ~~之前的代码会跳过所有以 `.` 开头的目录~~（已修复）
- 证书目录可能被误判为 acme.sh 程序目录

**解决方案**：
- ✅ 已修复：现在会检测路径是否包含 `acme.sh`，并只跳过特定的程序目录
- ✅ 支持 `_ecc` 和 `_rsa` 后缀的目录名

### 2. 带 _ecc 或 _rsa 后缀的证书

acme.sh 使用不同的密钥类型时会添加后缀：
- `example.com` - 默认 RSA 证书
- `example.com_ecc` - ECC 证书
- `example.com_rsa` - 显式指定的 RSA 证书

面板会自动：
- 识别这些后缀
- 去除后缀得到实际域名
- 使用完整目录名查找证书文件

示例：
```
目录：nkym.qzz.io_ecc/
文件：nkym.qzz.io_ecc.cer, nkym.qzz.io_ecc.key
显示域名：nkym.qzz.io
```

### 3. 通配符证书识别

**重要**：通配符证书不能从目录名判断，必须解析证书内容。

错误的判断方式：
- ❌ 目录名包含 `_wildcard`
- ❌ 目录名包含 `*`
- ❌ 文件名包含特殊标记

正确的判断方式：
- ✅ 解析证书文件
- ✅ 检查 SAN (Subject Alternative Names)
- ✅ 检查 CN (Common Name)
- ✅ 查找 `*.` 前缀

### 4. 权限问题

如果面板无法读取证书目录：

```bash
# 确保面板有读取权限
chmod 755 /root/.acme.sh
chmod 755 /root/.acme.sh/*
chmod 644 /root/.acme.sh/*/*.cer
chmod 644 /root/.acme.sh/*/*.key
```

### 5. 通配符证书识别

acme.sh 使用 `_wildcard` 前缀命名通配符证书目录：
- `_wildcard.example.com` 对应 `*.example.com`
- 面板会自动识别并标记为通配符证书

### 6. 证书文件不存在

确保证书文件存在且路径正确：

```bash
# 检查 acme.sh 证书
ls -la /root/.acme.sh/example.com/

# 检查 Let's Encrypt 证书
ls -la /etc/letsencrypt/live/example.com/

# 检查自定义目录证书
ls -la /etc/certs/example.com/
```

## 使用 acme.sh 的推荐配置

### 安装 acme.sh

```bash
curl https://get.acme.sh | sh -s email=your@email.com
```

### 申请证书

```bash
# 普通证书
acme.sh --issue -d example.com --nginx

# 通配符证书（需要 DNS API）
acme.sh --issue -d example.com -d *.example.com --dns dns_cf
```

### 安装证书到自定义目录（推荐）

```bash
# 创建目录
mkdir -p /etc/certs/example.com

# 安装证书
acme.sh --install-cert -d example.com \
  --key-file /etc/certs/example.com/key.pem \
  --fullchain-file /etc/certs/example.com/fullchain.pem \
  --reloadcmd "systemctl reload nginx"

# 配置面板使用 /etc/certs
# 在 config.yaml 中设置：
# nginx:
#   cert_dir: "/etc/certs"
```

### 自动续期

acme.sh 会自动添加 cron 任务进行续期：

```bash
# 查看 cron 任务
crontab -l | grep acme.sh

# 手动续期
acme.sh --renew -d example.com --force
```

## API 接口

### 扫描证书

```bash
GET /api/certificates/scan
```

**响应示例**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "cert_dir": "/root/.acme.sh",
    "certificates": [
      {
        "domain": "example.com",
        "domains": [
          "example.com",
          "www.example.com"
        ],
        "cert_path": "/root/.acme.sh/example.com/fullchain.cer",
        "key_path": "/root/.acme.sh/example.com/example.com.key",
        "expiry_date": "2024-12-31T23:59:59Z",
        "days_to_expiry": 90,
        "status": "有效",
        "issuer": "Let's Encrypt",
        "is_wildcard": false,
        "exists": true
      },
      {
        "domain": "example.com",
        "domains": [
          "*.example.com",
          "example.com"
        ],
        "cert_path": "/root/.acme.sh/example.com_ecc/fullchain.cer",
        "key_path": "/root/.acme.sh/example.com_ecc/example.com_ecc.key",
        "expiry_date": "2024-12-31T23:59:59Z",
        "days_to_expiry": 90,
        "status": "有效",
        "issuer": "Let's Encrypt",
        "is_wildcard": true,
        "exists": true
      },
      {
        "domain": "nkym.qzz.io",
        "domains": [
          "nkym.qzz.io"
        ],
        "cert_path": "/root/.acme.sh/nkym.qzz.io_ecc/fullchain.cer",
        "key_path": "/root/.acme.sh/nkym.qzz.io_ecc/nkym.qzz.io_ecc.key",
        "expiry_date": "2024-12-31T23:59:59Z",
        "days_to_expiry": 90,
        "status": "有效",
        "issuer": "Let's Encrypt",
        "is_wildcard": false,
        "exists": true
      }
    ],
    "count": 3
  }
}
```

**字段说明**：
- `domain`: 主域名（从目录名提取，去除 _ecc/_rsa 后缀）
- `domains`: 证书包含的所有域名（从证书 SAN 和 CN 解析）
- `cert_path`: 证书文件完整路径
- `key_path`: 私钥文件完整路径
- `expiry_date`: 证书过期时间（ISO 8601 格式）
- `days_to_expiry`: 距离过期的天数
- `status`: 证书状态
  - `有效`: 距离过期 > 30 天
  - `即将过期`: 距离过期 7-30 天
  - `已过期`: 已过期
  - `未知`: 无法解析证书
- `issuer`: 证书颁发者（CA）
- `is_wildcard`: 是否为通配符证书（域名包含 `*.`）
- `exists`: 证书文件是否存在

### 扫描并导入证书

```bash
POST /api/certificates/scan-import
```

自动将扫描到的证书导入为域名配置。

## 前端显示

### 证书扫描结果页面

扫描结果会显示以下信息：

1. **主域名**：从目录名提取的域名（去除 _ecc/_rsa 后缀）
   - 通配符证书会显示 ⚝ 图标

2. **证书包含的域名**：从证书解析的所有域名
   - 通配符域名（如 `*.example.com`）会用紫色徽章显示
   - 普通域名用代码格式显示

3. **过期时间**：
   - 显示具体过期日期
   - 显示距离过期的天数
   - 已过期显示负数天数

4. **状态徽章**：
   - 🟢 **有效**：距离过期 > 30 天（绿色）
   - 🟡 **即将过期**：距离过期 7-30 天（黄色）
   - 🔴 **已过期**：已过期（红色）
   - ⚪ **未知**：无法解析证书（灰色）

5. **颁发者**：显示 CA 名称（如 Let's Encrypt）

6. **操作按钮**：一键导入证书到域名管理

### 示例界面

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 🔍 扫描完成                                                                  │
│ 在 /root/.acme.sh 目录下找到 3 个证书                                        │
└─────────────────────────────────────────────────────────────────────────────┘

┌──────────────┬──────────────────────┬─────────────────┬────────┬──────────┐
│ 主域名       │ 证书包含的域名        │ 过期时间        │ 状态   │ 操作     │
├──────────────┼──────────────────────┼─────────────────┼────────┼──────────┤
│ 🌐 example.com│ example.com          │ 2024-12-31      │ 🟢 有效│ [导入]   │
│              │ www.example.com      │ 还有 90 天      │        │          │
├──────────────┼──────────────────────┼─────────────────┼────────┼──────────┤
│ ⚝ example.com│ ⚝ *.example.com      │ 2024-12-31      │ 🟢 有效│ [导入]   │
│              │ example.com          │ 还有 90 天      │        │          │
├──────────────┼──────────────────────┼─────────────────┼────────┼──────────┤
│ 🌐 nkym.qzz.io│ nkym.qzz.io          │ 2024-06-30      │ 🟡 即将│ [导入]   │
│              │                      │ 还有 15 天      │   过期 │          │
└──────────────┴──────────────────────┴─────────────────┴────────┴──────────┘
```

### 模板文件

证书扫描结果模板：`web/templates/components/certificates-scan-result.html`

该模板会：
- 显示所有扫描到的证书
- 高亮显示通配符证书
- 根据过期时间显示不同颜色的状态
- 提供一键导入功能

## 调试

如果扫描不到证书，可以：

1. **检查配置**：
   ```bash
   cat /opt/xray-panel/conf/config.yaml | grep cert_dir
   ```

2. **检查目录权限**：
   ```bash
   ls -la /root/.acme.sh/
   ```

3. **手动测试扫描**：
   ```bash
   curl -H "Authorization: Bearer <token>" \
     http://localhost:8082/api/certificates/scan
   ```

4. **查看日志**：
   ```bash
   tail -f /opt/xray-panel/logs/panel.log
   journalctl -u xray-panel -f
   ```
