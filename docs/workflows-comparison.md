# Workflows 对比

## 可用的 Workflows

项目提供了 4 个不同的 GitHub Actions workflows，适用于不同的场景。

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

## 3. Build (AMD64 & ARM64)

**文件**: `.github/workflows/build-simple.yml`

### 特点
- ⚠️ 平台过滤功能有限
- ✅ 代码简洁
- ✅ 支持所有平台

### 适用场景
- 不推荐使用（建议使用 Flexible Build）

### 问题
- `platforms` 参数不能完全控制构建
- 使用条件判断，可能不够灵活

---

## 4. Build Release (完整)

**文件**: `.github/workflows/build.yml`

### 特点
- ✅ 详细的打包过程
- ✅ 自定义 Release Notes
- ✅ 包含 README.txt

### 适用场景
- 需要自定义打包内容
- 需要详细的构建日志

### 使用方法
```
Actions -> Build Release -> Run workflow
输入版本号: v1.0.0
```

---

## 5. Test Build (自动)

**文件**: `.github/workflows/test-build.yml`

### 特点
- ✅ 自动触发
- ✅ 验证构建
- ❌ 不创建 Release

### 触发条件
- Push 到 main/master/develop
- Pull Request

### 适用场景
- 开发过程中验证
- CI/CD 自动测试

---

## 对比表格

| Workflow | 平台选择 | 架构选择 | 自动触发 | 创建 Release | 推荐度 |
|----------|---------|---------|---------|-------------|--------|
| Release Build | ❌ 全部 | ❌ 全部 | ❌ | ✅ | ⭐⭐⭐⭐⭐ |
| Flexible Build | ✅ 可选 | ✅ 可选 | ❌ | ✅ | ⭐⭐⭐⭐⭐ |
| Build (Simple) | ⚠️ 有限 | ❌ 全部 | ❌ | ✅ | ⭐⭐ |
| Build Release | ❌ 全部 | ❌ 全部 | ❌ | ✅ | ⭐⭐⭐⭐ |
| Test Build | ❌ 全部 | ❌ 全部 | ✅ | ❌ | ⭐⭐⭐ |

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

### 场景 4: 开发测试

**推荐**: Test Build（自动）或 Flexible Build（手动）

**自动**: Push 代码会自动触发 Test Build

**手动**: 使用 Flexible Build 选择需要的平台

---

### 场景 5: 自定义打包内容

**推荐**: Build Release

修改 `.github/workflows/build.yml` 中的打包步骤。

---

## 构建时间对比

基于 6 个平台/架构组合：

| Workflow | 平台数 | 时间（估计） |
|----------|--------|-------------|
| Release Build (全部) | 6 | 8-12 分钟 |
| Flexible (linux-only, both) | 2 | 3-5 分钟 |
| Flexible (linux-only, amd64) | 1 | 2-3 分钟 |
| Build Release (全部) | 6 | 10-15 分钟 |
| Test Build (全部) | 6 | 8-12 分钟 |

---

## 常见问题

### Q: 我应该使用哪个 workflow？

**A**: 
- 正式发布 → **Release Build**
- 需要选择平台 → **Flexible Build**
- 开发测试 → **Test Build**（自动）

### Q: Flexible Build 和 Release Build 有什么区别？

**A**:
- **Flexible Build**: 可以选择平台和架构
- **Release Build**: 构建所有平台，更简单

### Q: 为什么有这么多 workflows？

**A**: 不同场景有不同需求：
- 有时需要所有平台（发布）
- 有时只需要一个平台（测试）
- 有时需要自动构建（CI）

### Q: 可以删除不用的 workflows 吗？

**A**: 可以，建议保留：
- `release.yml` 或 `build-flexible.yml`（二选一）
- `test-build.yml`（自动测试）

### Q: 如何修改构建配置？

**A**: 编辑对应的 `.github/workflows/*.yml` 文件。

---

## 迁移指南

### 从 Build (Simple) 迁移到 Flexible Build

**之前**:
```yaml
# build-simple.yml
platforms: 'linux'  # 不起作用
```

**现在**:
```yaml
# build-flexible.yml
platforms: 'linux-only'  # 真正起作用
architectures: 'both'
```

### 从 Build Release 迁移到 Release Build

**区别**: Release Build 更简单，但功能相同。

**建议**: 如果不需要自定义打包，使用 Release Build。

---

## 最佳实践

1. **正式发布**: 使用 Release Build 或 Flexible Build (all)
2. **测试构建**: 使用 Flexible Build 选择特定平台
3. **开发验证**: 依赖 Test Build 自动触发
4. **版本号**: 使用语义化版本（v1.0.0）
5. **标签**: 正式发布后打 git tag

---

## 参考资源

- [GitHub Actions 使用指南](github-actions.md)
- [故障排查](github-actions-troubleshooting.md)
- [构建文档](building.md)
