# 快速修复 403 错误

## 问题
GitHub Actions 创建 Release 时出现 403 错误。

## 解决方案（3 步）

### 步骤 1: 修改仓库权限设置

1. 进入你的仓库页面
2. 点击 `Settings`（设置）
3. 在左侧菜单找到 `Actions` -> `General`
4. 滚动到 `Workflow permissions` 部分
5. 选择 **`Read and write permissions`**（读写权限）
6. 勾选 **`Allow GitHub Actions to create and approve pull requests`**
7. 点击 **`Save`**（保存）

### 步骤 2: 重新运行 Workflow

1. 进入 `Actions` 标签
2. 找到失败的 workflow run
3. 点击右上角的 `Re-run all jobs`（重新运行所有任务）

### 步骤 3: 验证

等待构建完成，检查 `Releases` 页面是否有新的 release。

---

## 截图指南

### 找到设置位置

```
仓库首页 -> Settings -> Actions -> General -> Workflow permissions
```

### 正确的配置

应该选择：
- ✅ **Read and write permissions**
- ✅ **Allow GitHub Actions to create and approve pull requests**

不要选择：
- ❌ Read repository contents and packages permissions

---

## 如果还是不行

### 方法 A: 使用 Personal Access Token

1. **创建 Token**:
   - 访问: https://github.com/settings/tokens
   - 点击 `Generate new token (classic)`
   - 勾选 `repo` 权限（所有子权限）
   - 生成并复制 token

2. **添加到仓库**:
   - 仓库 `Settings` -> `Secrets and variables` -> `Actions`
   - 点击 `New repository secret`
   - Name: `GH_TOKEN`
   - Value: 粘贴你的 token
   - 保存

3. **修改 workflow**:
   编辑 `.github/workflows/release.yml`，找到 `Create Release` 步骤，修改为：
   ```yaml
   - name: Create Release
     uses: softprops/action-gh-release@v2
     with:
       tag_name: ${{ github.event.inputs.version }}
       name: Release ${{ github.event.inputs.version }}
       body_path: release-notes.md
       draft: false
       prerelease: false
       files: release/*
       fail_on_unmatched_files: true
     env:
       GITHUB_TOKEN: ${{ secrets.GH_TOKEN }}  # 改这里
   ```

### 方法 B: 检查组织设置

如果仓库属于组织：

1. 进入组织设置: `Organization Settings`
2. `Actions` -> `General`
3. 确保允许 Actions 运行
4. 检查 `Workflow permissions` 设置

---

## 验证权限

运行这个命令检查当前权限：

```bash
curl -H "Authorization: token YOUR_GITHUB_TOKEN" \
  https://api.github.com/repos/nxovaeng/xray-panel
```

查看输出中的 `permissions` 字段。

---

## 常见问题

**Q: 我已经设置了权限，为什么还是 403？**

A: 尝试以下步骤：
1. 清除浏览器缓存
2. 退出并重新登录 GitHub
3. 等待 5-10 分钟（权限更新可能需要时间）
4. 使用无痕模式重新尝试

**Q: 我的仓库是 fork 的，有影响吗？**

A: Fork 的仓库需要在自己的仓库中设置权限，不能使用上游仓库的权限。

**Q: 可以手动创建 Release 吗？**

A: 可以，但不推荐：
1. 下载构建产物（Artifacts）
2. 手动创建 Release
3. 上传文件

---

## 需要帮助？

如果以上方法都不行，请：

1. 检查 Actions 日志中的详细错误信息
2. 在项目 Issues 中提问: https://github.com/nxovaeng/xray-panel/issues
3. 提供以下信息：
   - 完整的错误日志
   - 你的权限设置截图
   - 是否是组织仓库
   - 是否是 fork 的仓库

---

## 相关文档

- [详细故障排查指南](docs/github-actions-troubleshooting.md)
- [GitHub Actions 使用指南](docs/github-actions.md)
- [构建文档](docs/building.md)
