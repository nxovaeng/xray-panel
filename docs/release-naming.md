# Release 文件命名规范

## 概述

本文档定义了 Xray Panel 项目的 Release 文件命名规范，确保安装脚本能够正确下载文件。

---

## 命名格式

### 压缩包文件名

```
xray-panel-{VERSION}-{OS}-{ARCH}.{EXT}
```

**参数说明**:
- `{VERSION}`: 版本号，必须包含 `v` 前缀（如 `v1.0.0`）
- `{OS}`: 操作系统（`linux`, `windows`, `darwin`）
- `{ARCH}`: 架构（`amd64`, `arm64`）
- `{EXT}`: 扩展名（Linux/macOS 用 `tar.gz`，Windows 用 `zip`）

### 示例

**Linux**:
```
xray-panel-v1.0.0-linux-amd64.tar.gz
xray-panel-v1.0.0-linux-arm64.tar.gz
```

**Windows**:
```
xray-panel-v1.0.0-windows-amd64.zip
xray-panel-v1.0.0-windows-arm64.zip
```

**macOS**:
```
xray-panel-v1.0.0-darwin-amd64.tar.gz
xray-panel-v1.0.0-darwin-arm64.tar.gz
```

### 校验和文件

每个压缩包都应该有对应的 SHA256 校验和文件：

```
xray-panel-v1.0.0-linux-amd64.tar.gz.sha256
xray-panel-v1.0.0-windows-amd64.zip.sha256
```

---

## 下载 URL 格式

### 标准格式

```
https://github.com/{USER}/{REPO}/releases/download/{VERSION}/{FILENAME}
```

### 完整示例

```
https://github.com/nxovaeng/xray-panel/releases/download/v1.0.0/xray-panel-v1.0.0-linux-amd64.tar.gz
```

### 分解说明

| 部分 | 值 | 说明 |
|------|-----|------|
| 基础 URL | `https://github.com` | GitHub 域名 |
| 用户/组织 | `nxovaeng` | 仓库所有者 |
| 仓库名 | `xray-panel` | 仓库名称 |
| 路径 | `/releases/download/` | 固定路径 |
| 版本标签 | `v1.0.0` | Git tag 名称 |
| 文件名 | `xray-panel-v1.0.0-linux-amd64.tar.gz` | 完整文件名 |

---

## 常见错误

### ❌ 错误 1: 文件名缺少版本号

**错误**:
```
xray-panel-linux-amd64.tar.gz
```

**正确**:
```
xray-panel-v1.0.0-linux-amd64.tar.gz
```

**影响**: 安装脚本无法下载文件。

---

### ❌ 错误 2: 版本号缺少 v 前缀

**错误**:
```
xray-panel-1.0.0-linux-amd64.tar.gz
```

**正确**:
```
xray-panel-v1.0.0-linux-amd64.tar.gz
```

**影响**: 与 Git tag 不一致。

---

### ❌ 错误 3: 使用 latest 作为文件名

**错误**:
```
https://github.com/user/repo/releases/latest/download/xray-panel-linux-amd64.tar.gz
```

**问题**: GitHub 的 `latest` 重定向不会自动添加版本号到文件名。

**解决方案**: 
1. 先获取最新版本号
2. 使用完整的版本化 URL

```bash
# 获取最新版本
LATEST=$(curl -s https://api.github.com/repos/user/repo/releases/latest | grep tag_name | cut -d'"' -f4)

# 使用版本化 URL
wget https://github.com/user/repo/releases/download/${LATEST}/xray-panel-${LATEST}-linux-amd64.tar.gz
```

---

### ❌ 错误 4: URL 中的版本与文件名不匹配

**错误**:
```
https://github.com/user/repo/releases/download/v1.0.0/xray-panel-v1.0.1-linux-amd64.tar.gz
```

**正确**:
```
https://github.com/user/repo/releases/download/v1.0.0/xray-panel-v1.0.0-linux-amd64.tar.gz
```

---

## GitHub Actions 配置

### 确保正确的文件名

在 workflow 中，确保文件名包含版本号：

```yaml
- name: Package
  run: |
    VERSION="${{ github.event.inputs.version }}"
    OS="${{ matrix.os }}"
    ARCH="${{ matrix.arch }}"
    
    # 正确的文件名格式
    PACKAGE="xray-panel-${VERSION}-${OS}-${ARCH}"
    
    if [ "$OS" = "windows" ]; then
      zip -r "${PACKAGE}.zip" "${PACKAGE}"
    else
      tar czf "${PACKAGE}.tar.gz" "${PACKAGE}"
    fi
```

### 验证文件名

添加验证步骤：

```yaml
- name: Verify package names
  run: |
    cd dist
    for file in *.tar.gz *.zip; do
      if [[ ! "$file" =~ xray-panel-v[0-9]+\.[0-9]+\.[0-9]+-.*\.(tar\.gz|zip) ]]; then
        echo "Error: Invalid filename: $file"
        exit 1
      fi
    done
    echo "All filenames are valid"
```

---

## 安装脚本兼容性

### install.sh 期望的格式

```bash
# 对于 latest 版本
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep -oP '"tag_name": "\K(.*)(?=")')
URL="https://github.com/$GITHUB_REPO/releases/download/${LATEST_VERSION}/xray-panel-${LATEST_VERSION}-linux-${ARCH}.tar.gz"

# 对于指定版本
VERSION="v1.0.0"
URL="https://github.com/$GITHUB_REPO/releases/download/${VERSION}/xray-panel-${VERSION}-linux-${ARCH}.tar.gz"
```

### update.sh 期望的格式

与 install.sh 相同。

---

## 测试

### 测试下载 URL

使用测试脚本：

```bash
bash scripts/test-download-url.sh v1.0.0
```

### 手动测试

```bash
# 测试 URL 是否可访问
curl -I https://github.com/nxovaeng/xray-panel/releases/download/v1.0.0/xray-panel-v1.0.0-linux-amd64.tar.gz

# 应该返回 200 OK 或 302 Found
```

### 测试下载

```bash
# 实际下载测试
wget https://github.com/nxovaeng/xray-panel/releases/download/v1.0.0/xray-panel-v1.0.0-linux-amd64.tar.gz

# 验证文件
tar tzf xray-panel-v1.0.0-linux-amd64.tar.gz
```

---

## 版本号规范

### 语义化版本

使用 [Semantic Versioning](https://semver.org/):

```
v{MAJOR}.{MINOR}.{PATCH}[-{PRERELEASE}][+{BUILD}]
```

**示例**:
- `v1.0.0` - 正式版本
- `v1.0.0-beta.1` - Beta 版本
- `v1.0.0-rc.1` - Release Candidate
- `v1.0.0-alpha.1` - Alpha 版本

### Git Tag

Git tag 必须与版本号一致：

```bash
git tag v1.0.0
git push origin v1.0.0
```

---

## 检查清单

发布前检查：

- [ ] 版本号格式正确（`v1.0.0`）
- [ ] Git tag 已创建
- [ ] 文件名包含版本号
- [ ] 文件名格式：`xray-panel-{VERSION}-{OS}-{ARCH}.{EXT}`
- [ ] 所有平台的文件都已生成
- [ ] SHA256 校验和文件已生成
- [ ] 下载 URL 可访问
- [ ] 安装脚本测试通过

---

## 参考

### 完整的 Release 文件列表

一个完整的 v1.0.0 release 应该包含：

```
xray-panel-v1.0.0-linux-amd64.tar.gz
xray-panel-v1.0.0-linux-amd64.tar.gz.sha256
xray-panel-v1.0.0-linux-arm64.tar.gz
xray-panel-v1.0.0-linux-arm64.tar.gz.sha256
xray-panel-v1.0.0-windows-amd64.zip
xray-panel-v1.0.0-windows-amd64.zip.sha256
xray-panel-v1.0.0-windows-arm64.zip
xray-panel-v1.0.0-windows-arm64.zip.sha256
xray-panel-v1.0.0-darwin-amd64.tar.gz
xray-panel-v1.0.0-darwin-amd64.tar.gz.sha256
xray-panel-v1.0.0-darwin-arm64.tar.gz
xray-panel-v1.0.0-darwin-arm64.tar.gz.sha256
```

### 相关文档

- [GitHub Actions 使用指南](github-actions.md)
- [构建文档](building.md)
- [安装文档](installation.md)
