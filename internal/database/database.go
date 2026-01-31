package database

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"xray-panel/internal/config"
	applogger "xray-panel/internal/logger"
	"xray-panel/internal/models"
)

// Init initializes the SQLite database connection
func Init(dbPath string) (*gorm.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Migrate runs auto-migrations for all models
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Admin{},
		&models.User{},
		&models.Domain{},
		&models.Inbound{},
		&models.Outbound{},
		&models.RoutingRule{},
		&models.NginxConfig{},
		&models.Setting{})
}

// Seed creates default admin and settings if they don't exist
func Seed(db *gorm.DB, cfg *config.Config) error {
	// 1. Check if admin exists
	var count int64
	db.Model(&models.Admin{}).Count(&count)
	if count == 0 {
		username := cfg.Admin.Username
		password := cfg.Admin.Password
		email := cfg.Admin.Email

		// Generate random username if not specified
		if username == "" {
			username = "admin_" + generateRandomString(6)
		}

		// Generate random password if not specified
		if password == "" {
			password = generateRandomPassword(16)
		}

		admin := &models.Admin{
			Username: username,
			Email:    email,
		}
		if err := admin.SetPassword(password); err != nil {
			return err
		}
		if err := db.Create(admin).Error; err != nil {
			return err
		}

		// Log the credentials
		applogger.Info("========================================")
		applogger.Info("🔐 初始管理员账户已创建")
		applogger.Info("========================================")
		applogger.Info("用户名: %s", username)
		applogger.Info("密码:   %s", password)
		if email != "" {
			applogger.Info("邮箱:   %s", email)
		}
		applogger.Info("========================================")
		applogger.Info("⚠️  请立即登录并修改密码！")
		applogger.Info("⚠️  请妥善保存这些凭据，它们不会再次显示！")
		applogger.Info("========================================")
	}

	// 2. Default settings
	for _, s := range models.DefaultSettings() {
		db.Where("key = ?", s.Key).FirstOrCreate(&s)
	}

	// 3. Default routing rules
	db.Model(&models.RoutingRule{}).Count(&count)
	if count == 0 {
		for _, r := range models.DefaultRoutingRules() {
			db.Create(&r)
		}
	}

	return nil
}

// generateRandomString generates a random alphanumeric string
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// generateRandomPassword generates a secure random password
func generateRandomPassword(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to simple random if crypto/rand fails
		return generateRandomString(length)
	}
	// Use base64 encoding for better character variety
	password := base64.URLEncoding.EncodeToString(b)
	// Trim to desired length
	if len(password) > length {
		password = password[:length]
	}
	return password
}

// ResetAdminPassword resets admin password (for recovery)
func ResetAdminPassword(db *gorm.DB, username string, newPassword string) error {
	var admin models.Admin
	if err := db.Where("username = ?", username).First(&admin).Error; err != nil {
		return fmt.Errorf("admin not found: %w", err)
	}

	if err := admin.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	if err := db.Save(&admin).Error; err != nil {
		return fmt.Errorf("failed to save admin: %w", err)
	}

	applogger.Info("✅ Password reset successful for user: %s", username)
	return nil
}
