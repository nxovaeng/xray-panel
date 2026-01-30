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
	version   = "0.1.0"
	buildTime = "unknown"
)

func main() {
	// Command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	showAdmin := flag.Bool("show-admin", false, "Show admin account information")
	resetPassword := flag.Bool("reset-password", false, "Reset admin password")
	username := flag.String("username", "", "Admin username (for reset-password)")
	newPassword := flag.String("password", "", "New password (for reset-password)")
	configPath := flag.String("config", "", "Path to configuration file (auto-detect if empty)")
	nginxAction := flag.String("nginx", "", "Nginx action: 'sync' (regenerate all), 'reload', 'panel' (generate panel config)")
	domain := flag.String("domain", "", "Domain for panel config (required for -nginx=panel)")
	certPath := flag.String("cert", "", "Cert path for panel config (required for -nginx=panel)")
	keyPath := flag.String("key", "", "Key path for panel config (required for -nginx=panel)")
	
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("xray-panel version %s (built: %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Auto-detect config file based on OS if not specified
	if *configPath == "" {
		*configPath = getDefaultConfigPath()
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Warning: Could not load config file, using defaults: %v", err)
		cfg = config.Default()
	}

	// Initialize logger
	if err := logger.Init(&cfg.Log); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize database
	db, err := database.Init(cfg.Database.Path)
	if err != nil {
		logger.Fatal("Failed to initialize database: %v", err)
	}

	// Handle Nginx Actions
	if *nginxAction != "" {
		// Initialize Nginx generator
		// Note: removed StreamDir as it's deprecated in our new design
		ng := nginx.NewGenerator(cfg.Nginx.ConfigDir, cfg.Nginx.ReloadCmd)

		switch *nginxAction {
		case "sync":
			logger.Info("Syncing Nginx configurations...")
			// 1. Fetch inbounds from DB
			// We need a way to get inbounds. Since we changed models but didn't check the DB logic, 
			// let's assume we can traverse inbounds. 
			// Wait, the main package doesn't have direct access to 'GetAllInbounds' easy function without initializing more.
			// Let's check api.go or similar but simpler:
			// Just use GORM to query.
			var inbounds []models.Inbound
			if err := db.Preload("Domain").Find(&inbounds).Error; err != nil {
				logger.Fatal("Failed to query inbounds: %v", err)
			}
			if err := ng.GenerateHTTPConfig(inbounds); err != nil {
				logger.Fatal("Failed to generate configs: %v", err)
			}
			logger.Info("Nginx configurations generated successfully.")

		case "reload":
			logger.Info("Reloading Nginx...")
			if err := ng.Reload(); err != nil {
				logger.Fatal("Failed to reload Nginx: %v", err)
			}
			logger.Info("Nginx reloaded successfully.")

		case "panel":
			if *domain == "" || *certPath == "" || *keyPath == "" {
				logger.Fatal("Error: -domain, -cert, and -key are required for panel config generation")
			}
			logger.Info("Generating Nginx config for Panel (%s)...", *domain)
			if err := ng.GeneratePanelConfig(*domain, *certPath, *keyPath, cfg.Server.Listen); err != nil {
				logger.Fatal("Failed to generate panel config: %v", err)
			}
			logger.Info("Panel config generated. Please reload Nginx to apply.")

		default:
			logger.Fatal("Unknown nginx action: %s", *nginxAction)
		}
		os.Exit(0)
	}

	// Run auto migrations
	if err := database.Migrate(db); err != nil {
		logger.Fatal("Failed to run database migrations: %v", err)
	}

	// Show admin info
	if *showAdmin {
		showAdminInfo(db)
		os.Exit(0)
	}

	// Reset password
	if *resetPassword {
		if *username == "" {
			logger.Fatal("Error: -username is required for password reset")
		}
		if *newPassword == "" {
			logger.Fatal("Error: -password is required for password reset")
		}
		if err := database.ResetAdminPassword(db, *username, *newPassword); err != nil {
			logger.Fatal("Failed to reset password: %v", err)
		}
		os.Exit(0)
	}

	// Seed default data (only when running server)
	if err := database.Seed(db, cfg); err != nil {
		logger.Fatal("Failed to seed database: %v", err)
	}

	// Initialize and run API server
	logger.Info("Using config file: %s", *configPath)
	server := api.NewServer(cfg, db)
	logger.Info("Starting xray-panel on %s", cfg.Server.Listen)
	if err := server.Run(); err != nil {
		logger.Fatal("Server error: %v", err)
	}
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
		fmt.Printf("  创建时间: %s\n", admin.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  更新时间: %s\n", admin.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("\n========================================")
	fmt.Println("💡 提示:")
	fmt.Println("  - 如需重置密码，使用: ./panel -reset-password -username=<用户名> -password=<新密码>")
	fmt.Println("  - 密码已加密存储，无法直接查看")
	fmt.Println("========================================")
}
