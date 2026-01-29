# GitHub Actions 使用指南

## 可用的 Workflows

### 1. Release Build (推荐)
**文件**: `.github/workflows/release.yml`

**用途**: 构建所有平台的发布版本并创建 GitHub Release

**触发方式**: 手动触发

**使用步骤**:
1. 进入仓库的 Actions 页面
2. 选择 "Release Build" workflow
3. 点击 "Run workflow"
4. 输入版本号（如 `v1.0.0`）
5. 点击绿色的 "Run workflow" 按钮

**产物**:
- 6 个平台的构建包（tar.gz/zip）
- SHA256 校验和文件
- 自动创建的 GitHub Release

**优点**:
- ✅ 最简单可靠
- ✅ 自动创建 Release
- ✅ 包含完整的文档和配置
- ✅ 错误处理完善

---

### 2. Build Release (完整版)
**文件**: `.github/workflows/build.yml`

**用途**: 完整的构建流程，包含详细的打包步骤

**触发方式**: 手动触发

**特点**:
- 更详细的打包过程
- 包含 README.txt
- 自定义 Release Notes

---

### 3. Build (AMD64 & ARM64) (简化版)
**文件**: `.github/workflows/build-simple.yml`

**用途**: 简化的构建流程

**触发方式**: 手动触发

**特点**:
- 代码更简洁
- 构建速度快
- 适合快速测试

---

### 4. Test Build
**文件**: `.github/workflows/test-build.yml`

**用途**: 自动测试构建

**触发方式**: 
- Push 到 master/develop 分支
- Pull Request 到 master 分支

**特点**:
- 自动运行测试
- 验证所有平台构建
- 不创建 Release

---

## 使用 Release Build（推荐）

### 步骤详解

#### 1. 准备发布

确保代码已经提交并推送到 GitHub：

```bash
git add .
git commit -m "Release v1.0.0"
git push origin master
```

#### 2. 触发构建

1. 访问: `https://github.com/YOUR_USERNAME/xray-panel/actions`
2. 点击左侧的 "Release Build"
3. 点击右侧的 "Run workflow" 按钮
4. 在弹出的对话框中：
   - Branch: 选择 `master` 或你的主分支
   - Version: 输入 `v1.0.0`（必须以 v 开头）
5. 点击绿色的 "Run workflow" 按钮

#### 3. 等待构建完成

- 构建时间: 约 5-10 分钟
- 可以点击 workflow run 查看实时日志
- 绿色勾号表示成功，红色叉号表示失败

#### 4. 查看 Release

构建成功后：
1. 访问: `https://github.com/YOUR_USERNAME/xray-panel/releases`
2. 找到新创建的 Release
3. 下载对应平台的文件

---

## 常见问题

### Q1: 构建失败怎么办？

**查看日志**:
1. 点击失败的 workflow run
2. 点击失败的 job
3. 查看详细错误信息

**常见错误**:

**错误**: `go: cannot find main module`
```
解决: 确保 go.mod 文件存在且正确
```

**错误**: `permission denied`
```
解决: 检查 GITHUB_TOKEN 权限
Settings -> Actions -> General -> Workflow permissions
选择 "Read and write permissions"
```

**错误**: `artifact upload failed`
```
解决: 检查文件路径是否正确
确保 dist/ 目录中有构建产物
```

---

### Q2: 如何修改构建配置？

编辑 `.github/workflows/release.yml`:

**修改 Go 版本**:
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.22'  # 改为你需要的版本
```

**修改构建标志**:
```yaml
go build -v -trimpath \
  -ldflags="-s -w -X main.Version=${{ github.event.inputs.version }}" \
  -o "dist/${OUTPUT}" \
  ./cmd/panel
```

**添加更多平台**:
```yaml
strategy:
  matrix:
    include:
      - os: linux
        arch: amd64
      - os: freebsd  # 添加新平台
        arch: amd64
```

---

### Q3: 如何测试构建而不创建 Release？

使用 Test Build workflow:

```bash
# Push 代码会自动触发
git push origin master

# 或者创建 Pull Request
```

或者手动运行 Build (AMD64 & ARM64) workflow，但不要让它创建 Release。

---

### Q4: 构建产物在哪里？

**Artifacts** (临时):
- 位置: Actions -> Workflow run -> Artifacts
- 保留时间: 30 天（可配置）
- 用途: 测试和验证

**Release** (永久):
- 位置: Releases 页面
- 保留时间: 永久
- 用途: 正式发布

---

### Q5: 如何自动化发布流程？

创建 Git tag 自动触发:

编辑 `.github/workflows/release.yml`，添加:

```yaml
on:
  workflow_dispatch:
    # ... 保留手动触发
  push:
    tags:
      - 'v*'  # 推送 v* 标签时自动触发
```

使用方法:
```bash
git tag v1.0.0
git push origin v1.0.0
```

---

## 最佳实践

### 1. 版本号规范

使用语义化版本号:
- `v1.0.0` - 主版本.次版本.修订号
- `v1.0.0-beta.1` - 预发布版本
- `v1.0.0-rc.1` - 候选版本

### 2. 发布前检查清单

- [ ] 代码已提交并推送到 master 分支
- [ ] 所有测试通过
- [ ] 更新了 CHANGELOG
- [ ] 更新了版本号
- [ ] 更新了文档

### 3. 构建验证

发布后验证:
```bash
# 下载构建产物
wget https://github.com/USER/REPO/releases/download/v1.0.0/xray-panel-v1.0.0-linux-amd64.tar.gz

# 验证校验和
sha256sum -c xray-panel-v1.0.0-linux-amd64.tar.gz.sha256

# 解压测试
tar xzf xray-panel-v1.0.0-linux-amd64.tar.gz
cd xray-panel-v1.0.0-linux-amd64
./panel-linux-amd64 -version
```

### 4. 回滚策略

如果发布有问题:

1. **删除 Release**:
   - 进入 Releases 页面
   - 点击 Release 右侧的删除按钮

2. **删除 Tag**:
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

3. **修复并重新发布**:
   ```bash
   # 修复代码
   git commit -m "Fix: ..."
   git push
   
   # 重新触发构建
   ```

---

## 高级配置

### 并行构建

当前配置已经使用矩阵策略并行构建所有平台。

### 构建缓存

Go 模块缓存已启用:
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.21'
    cache: true  # 启用缓存
```

### 条件构建

只在特定条件下构建:
```yaml
jobs:
  build:
    if: startsWith(github.ref, 'refs/tags/v')  # 只在 tag 时构建
```

### 通知

添加构建通知:
```yaml
- name: Notify on success
  if: success()
  run: |
    curl -X POST https://your-webhook-url \
      -d "Build ${{ github.event.inputs.version }} succeeded"
```

---

## 故障排查

### 查看详细日志

```bash
# 启用调试日志
# 在 workflow 文件中添加:
env:
  ACTIONS_STEP_DEBUG: true
  ACTIONS_RUNNER_DEBUG: true
```

### 本地测试

使用 [act](https://github.com/nektos/act) 在本地运行 GitHub Actions:

```bash
# 安装 act
brew install act  # macOS
# 或
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# 运行 workflow
act workflow_dispatch -W .github/workflows/release.yml
```

---

## 参考资源

- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Go 构建文档](https://golang.org/doc/install/source)
- [语义化版本](https://semver.org/)
- [项目构建文档](building.md)
