# 构建文档

## 概述

本项目支持多平台、多架构的交叉编译，可以在任何平台上构建所有目标平台的二进制文件。

## 支持的平台和架构

| 平台 | 架构 | 说明 |
|------|------|------|
| Linux | amd64 | 64位 Intel/AMD 处理器 |
| Linux | arm64 | 64位 ARM 处理器（树莓派4、服务器等） |
| Windows | amd64 | 64位 Intel/AMD 处理器 |
| Windows | arm64 | 64位 ARM 处理器（Surface Pro X 等） |
| macOS | amd64 | Intel Mac |
| macOS | arm64 | Apple Silicon (M1/M2/M3) |

## 构建方法

### 方法 1: 使用 Makefile（推荐）

#### 查看帮助
```bash
make help
```

#### 构建当前平台
```bash
make build
```

#### 构建所有平台
```bash
make build-all
```

#### 构建特定平台
```bash
# Linux
make build-linux

# Windows
make build-windows

# macOS
make build-darwin
```

#### 指定版本号
```bash
make build-all VERSION=v1.2.3
```

#### 清理构建产物
```bash
make clean
```

#### 创建发布包
```bash
make package VERSION=v1.2.3
```

### 方法 2: 使用构建脚本

#### Linux/macOS
```bash
chmod +x scripts/build-all.sh
./scripts/build-all.sh v1.0.0
```

#### Windows
```cmd
scripts\build-all.bat v1.0.0
```

### 方法 3: 手动构建

#### 构建当前平台
```bash
go build -o panel ./cmd/panel
```

#### 交叉编译示例

**Linux AMD64:**
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o panel-linux-amd64 ./cmd/panel
```

**Linux ARM64:**
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o panel-linux-arm64 ./cmd/panel
```

**Windows AMD64:**
```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o panel-windows-amd64.exe ./cmd/panel
```

**Windows ARM64:**
```bash
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o panel-windows-arm64.exe ./cmd/panel
```

**macOS AMD64 (Intel):**
```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o panel-darwin-amd64 ./cmd/panel
```

**macOS ARM64 (Apple Silicon):**
```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o panel-darwin-arm64 ./cmd/panel
```

#### 优化构建（减小体积）
```bash
go build -ldflags="-s -w" -o panel ./cmd/panel
```

#### 带版本信息的构建
```bash
go build -ldflags="-s -w -X main.Version=v1.0.0" -o panel ./cmd/panel
```

## GitHub Actions 自动构建

### 触发构建

1. 进入 GitHub 仓库页面
2. 点击 "Actions" 标签
3. 选择 "Build Release" 或 "Build (AMD64 & ARM64)" workflow
4. 点击 "Run workflow"
5. 输入版本号（如 `v1.0.0`）
6. 点击 "Run workflow" 按钮

### 构建产物

构建完成后，会自动创建 GitHub Release，包含：

- 所有平台的二进制文件压缩包
- SHA256 校验和文件
- 自动生成的 Release Notes

### 下载构建产物

**从 Release 页面下载:**
```
https://github.com/YOUR_USERNAME/xray-panel/releases
```

**从 Actions 页面下载:**
1. 进入 Actions 标签
2. 选择对应的 workflow run
3. 在 "Artifacts" 部分下载

## 构建选项

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `GOOS` | 目标操作系统 | 当前系统 |
| `GOARCH` | 目标架构 | 当前架构 |
| `CGO_ENABLED` | 是否启用 CGO | 0 (禁用) |
| `VERSION` | 版本号 | v1.0.0 |

### 编译标志

| 标志 | 说明 |
|------|------|
| `-ldflags="-s -w"` | 去除调试信息，减小体积 |
| `-trimpath` | 去除文件路径信息 |
| `-v` | 显示详细编译信息 |

## 构建产物

### 目录结构
```
dist/
├── panel-linux-amd64
├── panel-linux-amd64.tar.gz
├── panel-linux-amd64.sha256
├── panel-linux-arm64
├── panel-linux-arm64.tar.gz
├── panel-linux-arm64.sha256
├── panel-windows-amd64.exe
├── panel-windows-amd64.zip
├── panel-windows-amd64.exe.sha256
├── panel-windows-arm64.exe
├── panel-windows-arm64.zip
├── panel-windows-arm64.exe.sha256
├── panel-darwin-amd64
├── panel-darwin-amd64.tar.gz
├── panel-darwin-amd64.sha256
├── panel-darwin-arm64
├── panel-darwin-arm64.tar.gz
└── panel-darwin-arm64.sha256
```

### 文件大小

典型的构建产物大小：

| 平台 | 架构 | 大小（约） |
|------|------|-----------|
| Linux | amd64 | 15-20 MB |
| Linux | arm64 | 15-20 MB |
| Windows | amd64 | 15-20 MB |
| Windows | arm64 | 15-20 MB |
| macOS | amd64 | 15-20 MB |
| macOS | arm64 | 15-20 MB |

## 验证构建

### 检查二进制文件
```bash
# 查看文件信息
file dist/panel-linux-amd64

# 查看版本
./dist/panel-linux-amd64 -version

# 验证 SHA256
sha256sum -c dist/panel-linux-amd64.sha256
```

### 测试运行
```bash
# 显示帮助
./dist/panel-linux-amd64 -h

# 显示管理员信息
./dist/panel-linux-amd64 -show-admin
```

## 开发构建

### 快速构建（开发用）
```bash
go build -o panel ./cmd/panel
```

### 带竞态检测的构建
```bash
make build-race
# 或
go build -race -o panel ./cmd/panel
```

### 运行测试
```bash
make test
# 或
go test -v ./...
```

### 开发模式运行
```bash
make dev
# 或
go run ./cmd/panel
```

## 依赖管理

### 安装依赖
```bash
make install
# 或
go mod download
go mod tidy
```

### 更新依赖
```bash
make update
# 或
go get -u ./...
go mod tidy
```

### 查看依赖
```bash
go mod graph
go list -m all
```

## 故障排查

### 构建失败

**问题**: `go: cannot find main module`
```bash
# 解决: 确保在项目根目录
cd /path/to/xray-panel
```

**问题**: `package xxx is not in GOROOT`
```bash
# 解决: 安装依赖
go mod download
```

**问题**: 交叉编译失败
```bash
# 解决: 确保 CGO_ENABLED=0
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/panel
```

### 二进制文件过大

```bash
# 使用优化标志
go build -ldflags="-s -w" -o panel ./cmd/panel

# 进一步压缩（需要 upx）
upx --best --lzma panel
```

### 权限问题

```bash
# Linux/macOS: 添加执行权限
chmod +x panel-linux-amd64

# Windows: 以管理员身份运行
```

## 最佳实践

### 生产环境构建

1. **使用版本标签**
   ```bash
   make build-all VERSION=v1.2.3
   ```

2. **启用优化**
   - 已在 Makefile 中默认启用 `-ldflags="-s -w"`

3. **验证构建**
   ```bash
   # 检查所有构建产物
   ls -lh dist/
   
   # 验证校验和
   cd dist && sha256sum -c *.sha256
   ```

4. **测试二进制**
   ```bash
   # 在目标平台上测试
   ./panel-linux-amd64 -version
   ./panel-linux-amd64 -show-admin
   ```

### CI/CD 集成

项目已包含 GitHub Actions 配置：

- `.github/workflows/build.yml` - 完整构建流程
- `.github/workflows/build-simple.yml` - 简化构建流程

可以根据需要修改或扩展这些配置。

## 发布流程

1. **更新版本号**
   ```bash
   # 在 cmd/panel/main.go 中更新
   var Version = "v1.2.3"
   ```

2. **提交代码**
   ```bash
   git add .
   git commit -m "Release v1.2.3"
   git push
   ```

3. **触发构建**
   - 在 GitHub Actions 中手动触发
   - 或使用 git tag 自动触发

4. **验证 Release**
   - 检查 Release 页面
   - 下载并测试二进制文件
   - 验证 SHA256 校验和

## 参考资源

- [Go 交叉编译文档](https://golang.org/doc/install/source#environment)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Go 构建标志](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)
