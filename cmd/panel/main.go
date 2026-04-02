package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/gorm"

	"xray-panel/internal/api"
	"xray-panel/internal/config"
	"xray-panel/internal/database"
	"xray-panel/internal/logger"
	"xray-panel/internal/models"
	"xray-panel/internal/nginx"
)

var (
	Version   = "0.1.0"
	BuildTime = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		runServer()
		return
	}

	// Parse subcommand
	switch os.Args[1] {
	case "version":
		cmdVersion()
	case "admin":
		cmdAdmin()
	case "reset-password":
		cmdResetPassword()
	case "nginx":
		cmdNginx()
	case "start", "run", "server":
		runServer()
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Printf("未知命令: %s\n", os.Args[1])
		fmt.Println("运行 'panel help' 查看可用命令")
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`xray-panel v%s - Xray 管理面板

用法:
  panel [command] [flags]

可用命令:
  start, run, server    启动面板服务器 (默认)
  version               显示版本信息
  admin                 显示管理员账户信息
  reset-password        重置管理员密码
  nginx                 Nginx 配置管理
  help                  显示帮助信息

运行 'panel [command] -h' 查看命令的详细帮助

示例:
  panel                                    # 启动服务器
  panel version                            # 显示版本
  panel admin                              # 显示管理员信息
  panel reset-password -u admin -p newpass # 重置密码
  panel nginx sync                         # 同步 Nginx 配置
  panel nginx reload                       # 重载 Nginx
  panel nginx panel -d example.com         # 生成面板配置

`, Version)
}

func cmdVersion() {
	fmt.Printf("xray-panel version %s\n", Version)
	fmt.Printf("Build time: %s\n", BuildTime)
}

func cmdAdmin() {
	fs := flag.NewFlagSet("admin", flag.ExitOnError)
	configPath := fs.String("config", "", "配置文件路径")
	fs.Parse(os.Args[2:])

	_, db := initSystem(*configPath)
	showAdminInfo(db)
}

func cmdResetPassword() {
	fs := flag.NewFlagSet("reset-password", flag.ExitOnError)
	username := fs.String("u", "", "用户名 (必需)")
	username2 := fs.String("username", "", "用户名 (必需)")
	password := fs.String("p", "", "新密码 (必需)")
	password2 := fs.String("password", "", "新密码 (必需)")
	configPath := fs.String("config", "", "配置文件路径")

	fs.Usage = func() {
		fmt.Println("用法: panel reset-password [flags]")
		fmt.Println("\n重置管理员密码")
		fmt.Println("\n参数:")
		fs.PrintDefaults()
		fmt.Println("\n示例:")
		fmt.Println("  panel reset-password -u admin -p newpassword")
		fmt.Println("  panel reset-password --username admin --password newpassword")
	}

	fs.Parse(os.Args[2:])

	// Support both short and long flags
	user := *username
	if user == "" {
		user = *username2
	}
	pass := *password
	if pass == "" {
		pass = *password2
	}

	if user == "" || pass == "" {
		fmt.Println("错误: 用户名和密码都是必需的")
		fs.Usage()
		os.Exit(1)
	}

	_, db := initSystem(*configPath)

	if err := database.ResetAdminPassword(db, user, pass); err != nil {
		logger.Fatal("重置密码失败: %v", err)
	}

	fmt.Printf("✅ 用户 '%s' 的密码已成功重置\n", user)
}

func cmdNginx() {
	if len(os.Args) < 3 {
		fmt.Println("用法: panel nginx [action]")
		fmt.Println("\n可用操作:")
		fmt.Println("  sync     同步生成所有 Nginx 配置")
		fmt.Println("  reload   重载 Nginx 服务")
		fmt.Println("  panel    生成面板的 Nginx 配置")
		fmt.Println("\n运行 'panel nginx [action] -h' 查看详细帮助")
		os.Exit(1)
	}

	action := os.Args[2]

	switch action {
	case "sync":
		nginxSync()
	case "reload":
		nginxReload()
	case "panel":
		nginxPanel()
	default:
		fmt.Printf("未知操作: %s\n", action)
		fmt.Println("可用操作: sync, reload, panel")
		os.Exit(1)
	}
}

func nginxSync() {
	fs := flag.NewFlagSet("nginx sync", flag.ExitOnError)
	configPath := fs.String("config", "", "配置文件路径")

	fs.Usage = func() {
		fmt.Println("用法: panel nginx sync [flags]")
		fmt.Println("\n从数据库同步生成所有 Nginx 配置文件")
		fmt.Println("\n参数:")
		fs.PrintDefaults()
	}

	fs.Parse(os.Args[3:])

	cfg, db := initSystem(*configPath)
	ng := nginx.NewGenerator(cfg.Nginx.ConfigDir, cfg.Nginx.StreamDir)

	logger.Info("正在同步 Nginx 配置...")

	var inbounds []models.Inbound
	if err := db.Preload("Domain").Where("enabled = ?", true).Find(&inbounds).Error; err != nil {
		logger.Fatal("查询入站配置失败: %v", err)
	}

	if err := ng.GenerateHTTPConfig(inbounds); err != nil {
		logger.Fatal("生成 HTTP 配置失败: %v", err)
	}

	logger.Info("✅ Nginx 配置已成功生成")
	fmt.Println("💡 提示: 运行 'panel nginx reload' 重载配置")
}

func nginxReload() {
	fs := flag.NewFlagSet("nginx reload", flag.ExitOnError)
	configPath := fs.String("config", "", "配置文件路径")

	fs.Usage = func() {
		fmt.Println("用法: panel nginx reload [flags]")
		fmt.Println("\n重载 Nginx 服务")
		fmt.Println("\n参数:")
		fs.PrintDefaults()
	}

	fs.Parse(os.Args[3:])

	cfg, _ := initSystem(*configPath)
	ng := nginx.NewGenerator(cfg.Nginx.ConfigDir, cfg.Nginx.StreamDir)

	logger.Info("正在重载 Nginx...")

	if err := ng.Reload(); err != nil {
		logger.Fatal("重载 Nginx 失败: %v", err)
	}

	logger.Info("✅ Nginx 已成功重载")
}

func nginxPanel() {
	fs := flag.NewFlagSet("nginx panel", flag.ExitOnError)
	domain := fs.String("d", "", "面板域名 (必需)")
	domain2 := fs.String("domain", "", "面板域名 (必需)")
	certPath := fs.String("cert", "", "SSL 证书路径 (必需)")
	keyPath := fs.String("key", "", "SSL 密钥路径 (必需)")
	configPath := fs.String("config", "", "配置文件路径")

	fs.Usage = func() {
		fmt.Println("用法: panel nginx panel [flags]")
		fmt.Println("\n为面板生成 Nginx 反向代理配置")
		fmt.Println("\n参数:")
		fs.PrintDefaults()
		fmt.Println("\n示例:")
		fmt.Println("  panel nginx panel -d panel.example.com -cert /path/to/cert.pem -key /path/to/key.pem")
	}

	fs.Parse(os.Args[3:])

	// Support both short and long flags
	dom := *domain
	if dom == "" {
		dom = *domain2
	}

	if dom == "" || *certPath == "" || *keyPath == "" {
		fmt.Println("错误: 域名、证书路径和密钥路径都是必需的")
		fs.Usage()
		os.Exit(1)
	}

	cfg, _ := initSystem(*configPath)
	ng := nginx.NewGenerator(cfg.Nginx.ConfigDir, cfg.Nginx.StreamDir)

	logger.Info("正在为面板生成 Nginx 配置 (%s)...", dom)

	if err := ng.GeneratePanelConfig(dom, *certPath, *keyPath, cfg.Server.Listen); err != nil {
		logger.Fatal("生成面板配置失败: %v", err)
	}

	logger.Info("✅ 面板配置已生成")
	fmt.Println("💡 提示: 运行 'panel nginx reload' 重载配置")
}

func runServer() {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	configPath := fs.String("config", "", "配置文件路径")

	// Parse flags after "start/run/server" or from beginning if no subcommand
	args := os.Args[1:]
	if len(os.Args) > 1 && (os.Args[1] == "start" || os.Args[1] == "run" || os.Args[1] == "server") {
		args = os.Args[2:]
	}
	fs.Parse(args)

	cfg, db := initSystem(*configPath)

	// Run auto migrations
	if err := database.Migrate(db); err != nil {
		logger.Fatal("数据库迁移失败: %v", err)
	}

	// Seed default data
	if err := database.Seed(db, cfg); err != nil {
		logger.Fatal("数据库初始化失败: %v", err)
	}

	// Initialize and run API server
	logger.Info("使用配置文件: %s", *configPath)
	server := api.NewServer(cfg, db)
	logger.Info("正在启动 xray-panel，监听地址: %s", cfg.Server.Listen)

	if err := server.Run(); err != nil {
		logger.Fatal("服务器错误: %v", err)
	}
}

// initSystem initializes configuration, logger, and database
func initSystem(configPath string) (*config.Config, *gorm.DB) {
	// Auto-detect config file based on OS if not specified
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("警告: 无法加载配置文件，使用默认配置: %v", err)
		cfg = config.Default()
	}

	// Initialize logger
	if err := logger.Init(&cfg.Log); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// Initialize database
	db, err := database.Init(cfg.Database.Path)
	if err != nil {
		logger.Fatal("初始化数据库失败: %v", err)
	}

	return cfg, db
}

// getDefaultConfigPath returns the default config path based on OS
func getDefaultConfigPath() string {
	// Check for OS-specific config files first
	osSpecific := getOSSpecificConfigPath()
	if _, err := os.Stat(osSpecific); err == nil {
		return osSpecific
	}

	// Fall back to generic config.yaml
	generic := "conf/config.yaml"
	if _, err := os.Stat(generic); err == nil {
		return generic
	}

	// Fall back to example config
	example := "conf/config.yaml.example"
	if _, err := os.Stat(example); err == nil {
		log.Printf("Warning: Using example config, please copy to config.yaml and customize")
		return example
	}

	// Last resort: return generic path (will use defaults)
	return generic
}

// getOSSpecificConfigPath returns OS-specific config file path
func getOSSpecificConfigPath() string {
	switch os := os.Getenv("GOOS"); os {
	case "windows":
		return "conf/config.windows.yaml"
	case "linux":
		return "conf/config.linux.yaml"
	case "darwin":
		return "conf/config.darwin.yaml"
	default:
		// Detect at runtime
		return getOSSpecificConfigPathRuntime()
	}
}

// getOSSpecificConfigPathRuntime detects OS at runtime
func getOSSpecificConfigPathRuntime() string {
	if isWindows() {
		return "conf/config.windows.yaml"
	}
	if isDarwin() {
		return "conf/config.darwin.yaml"
	}
	return "conf/config.linux.yaml"
}

// isWindows checks if running on Windows
func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

// isDarwin checks if running on macOS
func isDarwin() bool {
	// Simple check: macOS typically has /Applications directory
	if _, err := os.Stat("/Applications"); err == nil {
		return true
	}
	return false
}

// showAdminInfo displays admin account information
func showAdminInfo(db *gorm.DB) {
	var admins []struct {
		ID        string
		Username  string
		Email     string
		CreatedAt time.Time
		UpdatedAt time.Time
	}

	if err := db.Table("admins").Select("id, username, email, created_at, updated_at").Find(&admins).Error; err != nil {
		log.Fatalf("Failed to query admin accounts: %v", err)
	}

	if len(admins) == 0 {
		fmt.Println("No admin accounts found.")
		fmt.Println("Please start the server to create the initial admin account.")
		return
	}

	fmt.Println("========================================")
	fmt.Println("📋 管理员账户信息")
	fmt.Println("========================================")
	for i, admin := range admins {
		fmt.Printf("\n账户 #%d:\n", i+1)
		fmt.Printf("  用户名:   %s\n", admin.Username)
		if admin.Email != "" {
			fmt.Printf("  邮箱:     %s\n", admin.Email)
		}
	}
	fmt.Println("\n========================================")
	fmt.Println("💡 提示:")
	fmt.Println("  - 如需重置密码，使用: ./panel reset-password -username <用户名> -password <新密码>")
	fmt.Println("  - 如需重置密码，使用: ./panel reset-password -u <用户名> -p <新密码>")
	fmt.Println("  - 密码已加密存储，无法直接查看")
	fmt.Println("========================================")
}
