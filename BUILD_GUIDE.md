# 快速构建指南

## 本地构建

### Windows
```cmd
# 构建所有平台
scripts\build-all.bat v1.0.0

# 或使用 Go 命令直接构建当前平台
go build -o panel.exe ./cmd/panel
```

### Linux/macOS
```bash
# 构建所有平台
chmod +x scripts/build-all.sh
./scripts/build-all.sh v1.0.0

# 或使用 Makefile
make build-all VERSION=v1.0.0

# 或构建当前平台
make build
```

## GitHub Actions 自动构建

### 触发方式

1. 访问仓库的 Actions 页面
2. 选择 "Build Release" workflow
3. 点击 "Run workflow"
4. 输入版本号（如 `v1.0.0`）
5. 点击绿色的 "Run workflow" 按钮

### 构建产物

构建完成后会自动创建 Release，包含：

- ✅ Linux AMD64 (x86_64)
- ✅ Linux ARM64 (aarch64)
- ✅ Windows AMD64 (x86_64)
- ✅ Windows ARM64
- ✅ macOS AMD64 (Intel)
- ✅ macOS ARM64 (Apple Silicon)

每个平台都包含：
- 二进制文件
- 配置文件示例
- 文档
- SHA256 校验和

## 构建产物位置

本地构建：`dist/` 目录
GitHub Actions：Release 页面或 Artifacts

## 验证构建

```bash
# 查看版本
./panel-linux-amd64 -version

# 验证 SHA256
sha256sum -c panel-linux-amd64.sha256
```

## 更多信息

详细文档请参考：`docs/building.md`
