# 脚本使用说明

## 安装脚本

### install.sh

一键安装 Xray Panel 的脚本。

**基本使用**:
```bash
bash install.sh
```

**环境变量**:
- `GITHUB_REPO`: 指定 GitHub 仓库（默认: nxovaeng/xray-panel）
- `PANEL_VERSION`: 指定版本（默认: latest）

**示例**:
```bash
# 默认安装
bash install.sh

# 自定义仓库
GITHUB_REPO="your-username/xray-panel" bash install.sh

# 指定版本
PANEL_VERSION="v1.0.0" bash install.sh

# 同时指定
GITHUB_REPO="your-username/xray-panel" PANEL_VERSION="v1.0.0" bash install.sh
```

**远程安装**:
```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/main/scripts/install.sh)
```

---

## 更新脚本

### update.sh

更新已安装的 Xray Panel。

**基本使用**:
```bash
bash update.sh          # 更新到最新版本
bash update.sh v1.0.0   # 更新到指定版本
```

**环境变量**:
- `GITHUB_REPO`: 指定 GitHub 仓库（默认: nxovaeng/xray-panel）

**示例**:
```bash
# 更新到最新版本
bash update.sh

# 更新到指定版本
bash update.sh v1.0.0

# 自定义仓库
GITHUB_REPO="your-username/xray-panel" bash update.sh
```

**远程更新**:
```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/main/scripts/update.sh)
```

---

## 卸载脚本

### uninstall.sh

卸载 Xray Panel。

**基本使用**:
```bash
bash uninstall.sh
```

**功能**:
- 停止服务
- 删除二进制文件
- 删除 systemd 服务
- 删除 Nginx 配置
- 可选：删除数据和配置

**远程卸载**:
```bash
bash <(curl -Ls https://raw.githubusercontent.com/nxovaeng/xray-panel/main/scripts/uninstall.sh)
```

---

## 构建脚本

### build-all.sh / build-all.bat

构建所有平台的二进制文件。

**Linux/macOS**:
```bash
chmod +x build-all.sh
./build-all.sh v1.0.0
```

**Windows**:
```cmd
build-all.bat v1.0.0
```

---

## 其他脚本

### reset-admin.sh

重置管理员密码。

```bash
bash reset-admin.sh
```

### reset-admin-password.sh

使用 CLI 重置密码。

```bash
bash reset-admin-password.sh admin_xxx NewPassword123!
```

---

## GitHub 仓库检测

脚本会按以下顺序检测 GitHub 仓库：

1. **环境变量** `GITHUB_REPO`
   ```bash
   GITHUB_REPO="user/repo" bash install.sh
   ```

2. **Git Remote**（如果在 git 仓库中）
   ```bash
   git config --get remote.origin.url
   ```

3. **默认值**: `nxovaeng/xray-panel`

### 测试仓库检测

```bash
bash test-repo-detection.sh
```

---

## 常见问题

### Q: 如何使用 fork 的仓库？

```bash
GITHUB_REPO="your-username/xray-panel" bash install.sh
```

### Q: 如何安装开发版本？

```bash
PANEL_VERSION="dev" bash install.sh
```

### Q: 脚本在哪里下载二进制文件？

从 GitHub Releases，URL 格式：
```
https://github.com/$GITHUB_REPO/releases/download/$VERSION/xray-panel-$VERSION-$OS-$ARCH.tar.gz
```

示例：
```
https://github.com/nxovaeng/xray-panel/releases/download/v1.0.0/xray-panel-v1.0.0-linux-amd64.tar.gz
https://github.com/nxovaeng/xray-panel/releases/download/v1.0.0/xray-panel-v1.0.0-windows-amd64.zip
```

**注意**: 文件名中包含版本号！

### Q: 如何测试下载 URL？

```bash
bash scripts/test-download-url.sh v1.0.0
```

### Q: 如何验证下载的文件？

脚本会自动下载并验证 SHA256 校验和。

---

## 安全建议

1. ✅ 始终从官方仓库下载脚本
2. ✅ 检查脚本内容后再运行
3. ✅ 使用 HTTPS 下载
4. ✅ 验证 SHA256 校验和
5. ✅ 以 root 权限运行（需要安装系统服务）

---

## 故障排查

### 脚本下载失败

```bash
# 检查网络连接
curl -I https://github.com

# 使用代理
export https_proxy=http://proxy:port
bash install.sh
```

### 权限错误

```bash
# 确保以 root 运行
sudo bash install.sh
```

### 仓库检测错误

```bash
# 手动指定仓库
GITHUB_REPO="nxovaeng/xray-panel" bash install.sh
```

---

## 开发

### 测试脚本

```bash
# 在虚拟机或容器中测试
docker run -it ubuntu:22.04 bash
# 然后运行安装脚本
```

### 修改脚本

1. Fork 仓库
2. 修改脚本
3. 测试
4. 提交 Pull Request

---

## 参考

- [安装文档](../docs/installation.md)
- [构建文档](../docs/building.md)
- [GitHub Actions](../docs/github-actions.md)
