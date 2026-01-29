# GitHub Actions 故障排查

## 常见错误及解决方案

### 错误 1: Release failed with status: 403

**错误信息**:
```
GitHub release failed with status: 403undefined
```

**原因**: GitHub Actions 没有足够的权限创建 Release。

**解决方案**:

#### 方法 1: 检查仓库设置（推荐）

1. 进入仓库设置: `Settings` -> `Actions` -> `General`
2. 找到 `Workflow permissions` 部分
3. 选择 `Read and write permissions`
4. 勾选 `Allow GitHub Actions to create and approve pull requests`
5. 点击 `Save`

![Workflow Permissions](https://docs.github.com/assets/cb-45061/images/help/repository/actions-workflow-permissions.png)

#### 方法 2: 使用 Personal Access Token (PAT)

如果方法 1 不起作用，可以使用 PAT：

1. **创建 PAT**:
   - 进入 `Settings` -> `Developer settings` -> `Personal access tokens` -> `Tokens (classic)`
   - 点击 `Generate new token (classic)`
   - 选择权限: `repo` (完整权限)
   - 生成并复制 token

2. **添加到仓库 Secrets**:
   - 进入仓库 `Settings` -> `Secrets and variables` -> `Actions`
   - 点击 `New repository secret`
   - Name: `GH_TOKEN`
   - Value: 粘贴你的 PAT
   - 点击 `Add secret`

3. **修改 workflow 文件**:
   ```yaml
   - name: Create Release
     uses: softprops/action-gh-release@v2
     with:
       tag_name: ${{ github.event.inputs.version }}
       # ... 其他配置
     env:
       GITHUB_TOKEN: ${{ secrets.GH_TOKEN }}  # 使用自定义 token
   ```

#### 方法 3: 检查 workflow 文件权限

确保 workflow 文件中有正确的权限配置：

```yaml
name: Release Build

on:
  workflow_dispatch:
    # ...

permissions:
  contents: write      # 必需：创建 Release
  packages: write      # 可选：发布包

jobs:
  # ...
```

---

### 错误 2: Artifact upload failed

**错误信息**:
```
Error: No files were found with the provided path
```

**原因**: 构建产物路径不正确或文件不存在。

**解决方案**:

1. **检查构建步骤**:
   ```yaml
   - name: Build binary
     run: |
       mkdir -p dist
       go build -o dist/panel ./cmd/panel
       ls -lh dist/  # 验证文件存在
   ```

2. **检查上传路径**:
   ```yaml
   - name: Upload artifact
     uses: actions/upload-artifact@v4
     with:
       name: build-artifacts
       path: dist/*  # 确保路径正确
       if-no-files-found: error  # 如果没有文件则报错
   ```

3. **调试**:
   ```yaml
   - name: Debug
     run: |
       echo "Current directory:"
       pwd
       echo "Files in dist:"
       ls -lhR dist/
   ```

---

### 错误 3: Go build failed

**错误信息**:
```
go: cannot find main module
```

**原因**: 没有正确 checkout 代码或 go.mod 不存在。

**解决方案**:

```yaml
- name: Checkout code
  uses: actions/checkout@v4  # 确保使用 v4

- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.21'
    cache: true  # 启用缓存

- name: Get dependencies
  run: go mod download

- name: Build
  run: go build ./cmd/panel
```

---

### 错误 4: Permission denied

**错误信息**:
```
permission denied while trying to connect to the Docker daemon socket
```

**原因**: 在某些步骤中需要 Docker 权限。

**解决方案**:

```yaml
- name: Set up Docker
  run: |
    sudo chmod 666 /var/run/docker.sock
```

或使用 Docker action:
```yaml
- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3
```

---

### 错误 5: Rate limit exceeded

**错误信息**:
```
API rate limit exceeded
```

**原因**: GitHub API 调用次数超限。

**解决方案**:

1. **使用 GITHUB_TOKEN**:
   ```yaml
   env:
     GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
   ```

2. **减少 API 调用**:
   - 使用缓存
   - 合并多个请求

3. **等待限制重置**:
   - 通常 1 小时后重置

---

### 错误 6: Tag already exists

**错误信息**:
```
tag already exists
```

**原因**: 尝试创建已存在的 tag。

**解决方案**:

1. **删除旧 tag**:
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

2. **使用不同的版本号**:
   ```bash
   # 使用 v1.0.1 而不是 v1.0.0
   ```

3. **允许覆盖**（不推荐）:
   ```yaml
   - name: Create Release
     uses: softprops/action-gh-release@v2
     with:
       tag_name: ${{ github.event.inputs.version }}
       draft: false
       prerelease: false
       overwrite: true  # 允许覆盖
   ```

---

### 错误 7: Workflow not found

**错误信息**:
```
Workflow does not exist or does not have a workflow_dispatch trigger
```

**原因**: Workflow 文件不在 master 分支或没有 workflow_dispatch 触发器。

**解决方案**:

1. **确保文件在正确位置**:
   ```
   .github/workflows/release.yml
   ```

2. **确保已推送到 master 分支**:
   ```bash
   git add .github/workflows/release.yml
   git commit -m "Add release workflow"
   git push origin master
   ```

3. **检查触发器配置**:
   ```yaml
   on:
     workflow_dispatch:  # 必需
       inputs:
         version:
           required: true
   ```

---

## 调试技巧

### 1. 启用调试日志

在 workflow 中添加:
```yaml
env:
  ACTIONS_STEP_DEBUG: true
  ACTIONS_RUNNER_DEBUG: true
```

### 2. 添加调试步骤

```yaml
- name: Debug Info
  run: |
    echo "Event: ${{ github.event_name }}"
    echo "Ref: ${{ github.ref }}"
    echo "SHA: ${{ github.sha }}"
    echo "Actor: ${{ github.actor }}"
    echo "Repository: ${{ github.repository }}"
    env
```

### 3. 使用 tmate 进行交互式调试

```yaml
- name: Setup tmate session
  uses: mxschmitt/action-tmate@v3
  if: failure()  # 只在失败时启动
```

### 4. 检查 workflow 语法

使用 GitHub CLI:
```bash
gh workflow view release.yml
```

或在线验证:
```bash
# 使用 actionlint
docker run --rm -v $(pwd):/repo rhysd/actionlint:latest -color /repo/.github/workflows/release.yml
```

---

## 最佳实践

### 1. 使用最新版本的 Actions

```yaml
- uses: actions/checkout@v4        # 不是 v3
- uses: actions/setup-go@v5        # 不是 v4
- uses: actions/upload-artifact@v4 # 不是 v3
```

### 2. 添加错误处理

```yaml
- name: Build
  run: go build ./cmd/panel
  continue-on-error: false  # 失败时停止

- name: Upload on failure
  if: failure()
  uses: actions/upload-artifact@v4
  with:
    name: failure-logs
    path: |
      *.log
      go.sum
```

### 3. 使用矩阵策略

```yaml
strategy:
  matrix:
    os: [linux, windows, darwin]
    arch: [amd64, arm64]
  fail-fast: false  # 一个失败不影响其他
```

### 4. 缓存依赖

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.21'
    cache: true  # 自动缓存 go mod

- name: Cache Go modules
  uses: actions/cache@v3
  with:
    path: ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
```

### 5. 设置超时

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    timeout-minutes: 30  # 30 分钟超时
    steps:
      - name: Build
        timeout-minutes: 10  # 单步超时
        run: go build ./cmd/panel
```

---

## 检查清单

发布前检查:

- [ ] 代码已提交并推送到 master 分支
- [ ] Workflow 文件在 `.github/workflows/` 目录
- [ ] 仓库设置中启用了 Actions
- [ ] Workflow permissions 设置为 "Read and write"
- [ ] 版本号格式正确（如 v1.0.0）
- [ ] 没有同名的 tag 或 release
- [ ] go.mod 和 go.sum 文件存在
- [ ] 所有测试通过

---

## 获取帮助

### 查看 Actions 日志

1. 进入 `Actions` 标签
2. 点击失败的 workflow run
3. 点击失败的 job
4. 展开失败的步骤
5. 查看详细错误信息

### 查看 API 限制

```bash
curl -H "Authorization: token YOUR_TOKEN" \
  https://api.github.com/rate_limit
```

### 联系支持

- [GitHub Community](https://github.community/)
- [GitHub Support](https://support.github.com/)
- [项目 Issues](https://github.com/nxovaeng/xray-panel/issues)

---

## 参考资源

- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Workflow 语法](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)
- [权限配置](https://docs.github.com/en/actions/security-guides/automatic-token-authentication)
- [故障排查指南](https://docs.github.com/en/actions/monitoring-and-troubleshooting-workflows)
