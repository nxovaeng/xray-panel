# Workflows 对比

## 可用的 Workflows

项目提供了 2 个不同的 GitHub Actions workflows，适用于不同的场景。

---

## 1. Release Build (推荐)

**文件**: `.github/workflows/release.yml`

### 特点
- ✅ 最简单可靠
- ✅ 构建所有平台（6 个组合）
- ✅ 自动创建 Release
- ✅ 包含完整文档

### 适用场景
- 正式发布
- 需要所有平台的构建
- 首次使用

### 使用方法
```
Actions -> Release Build -> Run workflow
输入版本号: v1.0.0
```

### 构建产物
- Linux: amd64, arm64
- Windows: amd64, arm64
- macOS: amd64, arm64

---

## 2. Flexible Build (灵活)

**文件**: `.github/workflows/build-flexible.yml`

### 特点
- ✅ 可选择平台
- ✅ 可选择架构
- ✅ 动态构建矩阵
- ✅ 节省构建时间

### 适用场景
- 只需要特定平台
- 测试单个平台
- 快速构建

### 使用方法
```
Actions -> Flexible Build -> Run workflow
版本号: v1.0.0
平台选择:
  - all (所有平台)
  - linux-only (只构建 Linux)
  - windows-only (只构建 Windows)
  - darwin-only (只构建 macOS)
  - linux-windows (Linux + Windows)
架构选择:
  - both (amd64 + arm64)
  - amd64-only (只构建 amd64)
  - arm64-only (只构建 arm64)
```

### 示例

**只构建 Linux**:
- Platforms: `linux-only`
- Architectures: `both`
- 结果: linux-amd64, linux-arm64

**只构建 Windows AMD64**:
- Platforms: `windows-only`
- Architectures: `amd64-only`
- 结果: windows-amd64

**Linux + Windows (所有架构)**:
- Platforms: `linux-windows`
- Architectures: `both`
- 结果: linux-amd64, linux-arm64, windows-amd64, windows-arm64

---

## 对比表格

| Workflow | 平台选择 | 架构选择 | 自动触发 | 创建 Release |
|----------|---------|---------|---------|-------------|
| Release Build | ❌ 全部 | ❌ 全部 | ❌ | ✅ |
| Flexible Build | ✅ 可选 | ✅ 可选 | ❌ | ✅ |


---

## 使用建议

### 场景 1: 正式发布（所有平台）

**推荐**: Release Build

```
Actions -> Release Build -> Run workflow
版本: v1.0.0
```

**原因**: 最简单，构建所有平台，适合正式发布。

---

### 场景 2: 只发布 Linux 版本

**推荐**: Flexible Build

```
Actions -> Flexible Build -> Run workflow
版本: v1.0.0
平台: linux-only
架构: both
```

**原因**: 节省时间，只构建需要的平台。

---

### 场景 3: 测试 Windows ARM64

**推荐**: Flexible Build

```
Actions -> Flexible Build -> Run workflow
版本: v1.0.0-test
平台: windows-only
架构: arm64-only
```

**原因**: 快速测试单个平台/架构组合。

---

## 参考资源

- [GitHub Actions 使用指南](github-actions.md)
- [故障排查](github-actions-troubleshooting.md)
- [构建文档](building.md)
