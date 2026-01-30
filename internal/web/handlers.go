package web

import (
	cryptorand "crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"xray-panel/internal/logger"
	"xray-panel/internal/models"
	"xray-panel/internal/system"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ============ Page Handlers ============

func (h *Handler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func (h *Handler) DashboardPage(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		user = nil
	}

	c.HTML(http.StatusOK, "dashboard", gin.H{
		"Title": "Dashboard",
		"Page":  "dashboard",
		"User":  user,
	})
}

func (h *Handler) UsersPage(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		user = nil
	}

	c.HTML(http.StatusOK, "users", gin.H{
		"Title": "Users",
		"Page":  "users",
		"User":  user,
	})
}

func (h *Handler) InboundsPage(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		user = nil
	}

	c.HTML(http.StatusOK, "inbounds", gin.H{
		"Title": "Inbounds",
		"Page":  "inbounds",
		"User":  user,
	})
}

func (h *Handler) OutboundsPage(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		user = nil
	}

	c.HTML(http.StatusOK, "outbounds", gin.H{
		"Title": "Outbounds",
		"Page":  "outbounds",
		"User":  user,
	})
}

func (h *Handler) RoutingPage(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		user = nil
	}

	c.HTML(http.StatusOK, "routing", gin.H{
		"Title": "Routing",
		"Page":  "routing",
		"User":  user,
	})
}

func (h *Handler) DomainsPage(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		user = nil
	}

	c.HTML(http.StatusOK, "domains", gin.H{
		"Title": "Domains",
		"Page":  "domains",
		"User":  user,
	})
}

func (h *Handler) SettingsPage(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		user = nil
	}

	c.HTML(http.StatusOK, "settings", gin.H{
		"Title": "Settings",
		"Page":  "settings",
		"User":  user,
		"Time":  time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ============ Dashboard API ============

func (h *Handler) DashboardStats(c *gin.Context) {
	// Get user statistics
	var totalUsers, activeUsers int64
	h.db.Model(&models.User{}).Count(&totalUsers)
	h.db.Model(&models.User{}).Where("enabled = ?", true).Count(&activeUsers)

	// Get inbound statistics
	var totalInbounds int64
	h.db.Model(&models.Inbound{}).Where("enabled = ?", true).Count(&totalInbounds)

	// Calculate total traffic
	var users []models.User
	h.db.Find(&users)
	var totalTraffic int64
	for _, u := range users {
		totalTraffic += u.TrafficUsed
	}

	// Get system info
	sysInfo, err := system.GetSystemInfo()
	if err != nil {
		// If system info fails, use defaults
		sysInfo = &system.SystemInfo{}
	}

	stats := struct {
		// User stats
		TotalUsers   int64  `json:"total_users"`
		ActiveUsers  int64  `json:"active_users"`
		TotalInbounds int64 `json:"total_inbounds"`
		TotalTraffic string `json:"total_traffic"`
		
		// System stats
		CPUUsage    string `json:"cpu_usage"`
		CPUCores    int    `json:"cpu_cores"`
		MemUsage    string `json:"mem_usage"`
		MemPercent  string `json:"mem_percent"`
		DiskUsage   string `json:"disk_usage"`
		DiskPercent string `json:"disk_percent"`
		Uptime      string `json:"uptime"`
		OS          string `json:"os"`
	}{
		TotalUsers:    totalUsers,
		ActiveUsers:   activeUsers,
		TotalInbounds: totalInbounds,
		TotalTraffic:  system.FormatBytes(uint64(totalTraffic)),
		
		CPUUsage:    fmt.Sprintf("%.1f%%", sysInfo.CPUUsage),
		CPUCores:    sysInfo.CPUCores,
		MemUsage:    fmt.Sprintf("%s / %s", system.FormatBytes(sysInfo.MemUsed), system.FormatBytes(sysInfo.MemTotal)),
		MemPercent:  fmt.Sprintf("%.1f%%", sysInfo.MemPercent),
		DiskUsage:   fmt.Sprintf("%s / %s", system.FormatBytes(sysInfo.DiskUsed), system.FormatBytes(sysInfo.DiskTotal)),
		DiskPercent: fmt.Sprintf("%.1f%%", sysInfo.DiskPercent),
		Uptime:      system.FormatDuration(sysInfo.Uptime),
		OS:          sysInfo.Platform,
	}

	c.HTML(http.StatusOK, "components/dashboard-stats.html", stats)
}

// ============ Users API ============

func (h *Handler) UsersTable(c *gin.Context) {
	var users []models.User
	if err := h.db.Find(&users).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error loading users")
		return
	}

	type UserView struct {
		models.User
		CreatedAt  string
		SubURL     string
		ExpiryDate string
	}

	// Get base URL from request
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + c.Request.Host

	userViews := make([]UserView, len(users))
	for i, u := range users {
		subURL := baseURL + "/sub/" + u.SubPath
		expiryDate := ""
		if !u.ExpiryDate.IsZero() {
			expiryDate = u.ExpiryDate.Format("2006-01-02")
		}

		userViews[i] = UserView{
			User:       u,
			CreatedAt:  u.CreatedAt.Format("2006-01-02 15:04"),
			SubURL:     subURL,
			ExpiryDate: expiryDate,
		}
	}

	c.HTML(http.StatusOK, "components/users-table.html", gin.H{
		"Users": userViews,
	})
}

func (h *Handler) NewUserForm(c *gin.Context) {
	c.HTML(http.StatusOK, "components/user-form.html", gin.H{
		"GeneratedUUID": generateUUID(),
	})
}

func (h *Handler) EditUserForm(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	c.HTML(http.StatusOK, "components/user-form.html", gin.H{
		"User": user,
	})
}

func (h *Handler) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBind(&user); err != nil {
		c.String(http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Validate required fields
	if user.Name == "" {
		c.String(http.StatusBadRequest, "用户名不能为空")
		return
	}
	if user.Email == "" {
		c.String(http.StatusBadRequest, "邮箱不能为空")
		return
	}
	if user.UUID == "" {
		c.String(http.StatusBadRequest, "UUID 不能为空")
		return
	}

	// Convert traffic_limit from GB to bytes if provided
	if trafficGB := c.PostForm("traffic_limit"); trafficGB != "" {
		var gb int64
		if _, err := fmt.Sscanf(trafficGB, "%d", &gb); err == nil {
			user.TrafficLimit = gb * 1024 * 1024 * 1024
		}
	}

	// Set default values
	user.CreatedAt = time.Now()
	user.TrafficUsed = 0
	
	if err := h.db.Create(&user).Error; err != nil {
		logger.Error("Failed to create user %s: %v", user.Email, err)
		c.String(http.StatusInternalServerError, "Error creating user: "+err.Error())
		return
	}

	logger.Info("User created: %s (UUID: %s)", user.Email, user.UUID)
	h.UsersTable(c)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	
	// Get existing user
	var existingUser models.User
	if err := h.db.First(&existingUser, "id = ?", id).Error; err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	var user models.User
	if err := c.ShouldBind(&user); err != nil {
		c.String(http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Validate required fields
	if user.Name == "" {
		c.String(http.StatusBadRequest, "用户名不能为空")
		return
	}
	if user.Email == "" {
		c.String(http.StatusBadRequest, "邮箱不能为空")
		return
	}
	if user.UUID == "" {
		c.String(http.StatusBadRequest, "UUID 不能为空")
		return
	}

	// Convert traffic_limit from GB to bytes if provided
	if trafficGB := c.PostForm("traffic_limit"); trafficGB != "" {
		var gb int64
		if _, err := fmt.Sscanf(trafficGB, "%d", &gb); err == nil {
			user.TrafficLimit = gb * 1024 * 1024 * 1024
		}
	}

	// Preserve traffic used
	user.TrafficUsed = existingUser.TrafficUsed

	if err := h.db.Model(&models.User{}).Where("id = ?", id).Updates(&user).Error; err != nil {
		logger.Error("Failed to update user %s: %v", id, err)
		c.String(http.StatusInternalServerError, "Error updating user: "+err.Error())
		return
	}

	logger.Info("User updated: %s (UUID: %s)", user.Email, user.UUID)
	h.UsersTable(c)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	
	// Get user info before deletion for logging
	var user models.User
	if err := h.db.First(&user, "id = ?", id).Error; err == nil {
		logger.Info("User deleted: %s (UUID: %s)", user.Email, user.UUID)
	}
	
	if err := h.db.Delete(&models.User{}, "id = ?", id).Error; err != nil {
		logger.Error("Failed to delete user %s: %v", id, err)
		c.String(http.StatusInternalServerError, "Error deleting user")
		return
	}

	c.String(http.StatusOK, "")
}

func (h *Handler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	var users []models.User
	if err := h.db.Where("email LIKE ?", "%"+query+"%").Find(&users).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error searching users")
		return
	}

	type UserView struct {
		models.User
		CreatedAt string
	}

	userViews := make([]UserView, len(users))
	for i, u := range users {
		userViews[i] = UserView{
			User:      u,
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04"),
		}
	}

	c.HTML(http.StatusOK, "components/users-table.html", gin.H{
		"Users": userViews,
	})
}

// ============ Inbounds API ============

func (h *Handler) InboundsTable(c *gin.Context) {
	var inbounds []models.Inbound
	if err := h.db.Preload("Domain").Find(&inbounds).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error loading inbounds")
		return
	}

	c.HTML(http.StatusOK, "components/inbounds-table.html", gin.H{
		"Inbounds": inbounds,
	})
}

func (h *Handler) NewInboundForm(c *gin.Context) {
	var domains []models.Domain
	h.db.Find(&domains)
	c.HTML(http.StatusOK, "components/inbound-form.html", gin.H{
		"Domains": domains,
	})
}

func (h *Handler) EditInboundForm(c *gin.Context) {
	id := c.Param("id")
	var inbound models.Inbound
	if err := h.db.Preload("Domain").First(&inbound, "id = ?", id).Error; err != nil {
		c.String(http.StatusNotFound, "Inbound not found")
		return
	}

	var domains []models.Domain
	h.db.Find(&domains)
	c.HTML(http.StatusOK, "components/inbound-form.html", gin.H{
		"Inbound": inbound,
		"Domains": domains,
	})
}

func (h *Handler) CreateInbound(c *gin.Context) {
	var inbound models.Inbound
	if err := c.ShouldBind(&inbound); err != nil {
		c.String(http.StatusBadRequest, "Invalid input")
		return
	}

	// Handle wildcard certificate: generate random subdomain
	if inbound.DomainID != "" {
		var domain models.Domain
		if err := h.db.First(&domain, "id = ?", inbound.DomainID).Error; err == nil {
			// Check if domain is a wildcard (starts with *.)
			if strings.HasPrefix(domain.Domain, "*.") {
				baseDomain := strings.TrimPrefix(domain.Domain, "*.")
				subdomain := generateRandomSubdomain()
				inbound.ActualDomain = subdomain + "." + baseDomain
				logger.Info("Generated subdomain for wildcard cert: %s", inbound.ActualDomain)
			}
		}
	}

	if err := h.db.Create(&inbound).Error; err != nil {
		logger.Error("Failed to create inbound %s: %v", inbound.Tag, err)
		c.String(http.StatusInternalServerError, "Error creating inbound")
		return
	}

	logger.Info("Inbound created: %s (Protocol: %s, Port: %d)", inbound.Tag, inbound.Protocol, inbound.Port)
	h.InboundsTable(c)
}

func (h *Handler) UpdateInbound(c *gin.Context) {
	id := c.Param("id")
	var inbound models.Inbound
	if err := c.ShouldBind(&inbound); err != nil {
		c.String(http.StatusBadRequest, "Invalid input")
		return
	}

	if err := h.db.Model(&models.Inbound{}).Where("id = ?", id).Updates(&inbound).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error updating inbound")
		return
	}

	h.InboundsTable(c)
}

func (h *Handler) DeleteInbound(c *gin.Context) {
	id := c.Param("id")
	
	// Get inbound info before deletion for logging
	var inbound models.Inbound
	if err := h.db.First(&inbound, "id = ?", id).Error; err == nil {
		logger.Info("Inbound deleted: %s", inbound.Tag)
	}
	
	if err := h.db.Delete(&models.Inbound{}, "id = ?", id).Error; err != nil {
		logger.Error("Failed to delete inbound %s: %v", id, err)
		c.String(http.StatusInternalServerError, "Error deleting inbound")
		return
	}

	c.String(http.StatusOK, "")
}

// ============ Domains API ============

func (h *Handler) DomainsTable(c *gin.Context) {
	var domains []models.Domain
	if err := h.db.Find(&domains).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error loading domains")
		return
	}

	c.HTML(http.StatusOK, "components/domains-table.html", gin.H{
		"Domains": domains,
	})
}

func (h *Handler) NewDomainForm(c *gin.Context) {
	c.HTML(http.StatusOK, "components/domain-form.html", nil)
}

func (h *Handler) EditDomainForm(c *gin.Context) {
	id := c.Param("id")
	var domain models.Domain
	if err := h.db.First(&domain, "id = ?", id).Error; err != nil {
		c.String(http.StatusNotFound, "Domain not found")
		return
	}

	c.HTML(http.StatusOK, "components/domain-form.html", gin.H{
		"Domain": domain,
	})
}

func (h *Handler) CreateDomain(c *gin.Context) {
	var domain models.Domain
	if err := c.ShouldBind(&domain); err != nil {
		c.String(http.StatusBadRequest, "输入无效")
		return
	}

	// Validate domain format
	if !validateDomain(domain.Domain) {
		c.String(http.StatusBadRequest, "域名格式无效")
		return
	}

	// Validate certificate paths exist
	if valid, errMsg := validateCertificatePaths(domain.CertPath, domain.KeyPath); !valid {
		c.String(http.StatusBadRequest, errMsg)
		return
	}

	if err := h.db.Create(&domain).Error; err != nil {
		c.String(http.StatusInternalServerError, "创建域名失败")
		return
	}

	logger.Info("Domain created: %s", domain.Domain)
	h.DomainsTable(c)
}

func (h *Handler) UpdateDomain(c *gin.Context) {
	id := c.Param("id")
	var domain models.Domain
	if err := c.ShouldBind(&domain); err != nil {
		c.String(http.StatusBadRequest, "输入无效")
		return
	}

	// Validate domain format
	if !validateDomain(domain.Domain) {
		c.String(http.StatusBadRequest, "域名格式无效")
		return
	}

	// Validate certificate paths exist
	if valid, errMsg := validateCertificatePaths(domain.CertPath, domain.KeyPath); !valid {
		c.String(http.StatusBadRequest, errMsg)
		return
	}

	if err := h.db.Model(&models.Domain{}).Where("id = ?", id).Updates(&domain).Error; err != nil {
		c.String(http.StatusInternalServerError, "更新域名失败")
		return
	}

	logger.Info("Domain updated: %s", domain.Domain)
	h.DomainsTable(c)
}

func (h *Handler) DeleteDomain(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&models.Domain{}, "id = ?", id).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error deleting domain")
		return
	}

	c.String(http.StatusOK, "")
}

// ============ Helper Functions ============

// generateUUID generates a cryptographically secure UUID v4
func generateUUID() string {
	return uuid.New().String()
}

// generateRandomSubdomain generates a cryptographically secure random 6-8 character subdomain
func generateRandomSubdomain() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	// Generate random length between 6 and 8
	lengthBig, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(3))
	length := int(lengthBig.Int64()) + 6 // 6, 7, or 8 characters

	b := make([]byte, length)
	for i := range b {
		num, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

// domainRegex validates domain name format
var domainRegex = regexp.MustCompile(`^(\*\.)?([a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// validateDomain checks if a domain name is valid
func validateDomain(domain string) bool {
	if domain == "" {
		return false
	}
	// Allow wildcard domains like *.example.com
	return domainRegex.MatchString(domain)
}

// validateCertificatePaths checks if certificate and key files exist
func validateCertificatePaths(certPath, keyPath string) (bool, string) {
	if certPath == "" || keyPath == "" {
		return false, "证书路径和密钥路径不能为空"
	}

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return false, fmt.Sprintf("证书文件不存在: %s", certPath)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return false, fmt.Sprintf("密钥文件不存在: %s", keyPath)
	}

	return true, ""
}

// ============ Outbounds API ============

func (h *Handler) OutboundsTable(c *gin.Context) {
	var outbounds []models.Outbound
	if err := h.db.Find(&outbounds).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error loading outbounds")
		return
	}

	c.HTML(http.StatusOK, "components/outbounds-table.html", gin.H{
		"Outbounds": outbounds,
	})
}

func (h *Handler) NewOutboundForm(c *gin.Context) {
	c.HTML(http.StatusOK, "components/outbound-form.html", nil)
}

func (h *Handler) EditOutboundForm(c *gin.Context) {
	id := c.Param("id")
	var outbound models.Outbound
	if err := h.db.First(&outbound, "id = ?", id).Error; err != nil {
		c.String(http.StatusNotFound, "Outbound not found")
		return
	}

	c.HTML(http.StatusOK, "components/outbound-form.html", gin.H{
		"Outbound": outbound,
	})
}

func (h *Handler) CreateOutbound(c *gin.Context) {
	var outbound models.Outbound
	if err := c.ShouldBind(&outbound); err != nil {
		c.String(http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Set defaults
	if outbound.Tag == "" {
		c.String(http.StatusBadRequest, "Tag is required")
		return
	}
	
	// Set timestamps
	outbound.CreatedAt = time.Now()
	outbound.UpdatedAt = time.Now()
	outbound.Enabled = true // Default to enabled

	if err := h.db.Create(&outbound).Error; err != nil {
		logger.Error("Failed to create outbound %s: %v", outbound.Tag, err)
		c.String(http.StatusInternalServerError, "Error creating outbound: "+err.Error())
		return
	}

	logger.Info("Outbound created: %s (Type: %s)", outbound.Tag, outbound.Type)
	h.OutboundsTable(c)
}

func (h *Handler) UpdateOutbound(c *gin.Context) {
	id := c.Param("id")
	var outbound models.Outbound
	if err := c.ShouldBind(&outbound); err != nil {
		c.String(http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Set update timestamp
	outbound.UpdatedAt = time.Now()

	if err := h.db.Model(&models.Outbound{}).Where("id = ?", id).Updates(&outbound).Error; err != nil {
		logger.Error("Failed to update outbound %s: %v", id, err)
		c.String(http.StatusInternalServerError, "Error updating outbound: "+err.Error())
		return
	}

	logger.Info("Outbound updated: %s", outbound.Tag)
	h.OutboundsTable(c)
}

func (h *Handler) DeleteOutbound(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&models.Outbound{}, "id = ?", id).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error deleting outbound")
		return
	}

	c.String(http.StatusOK, "")
}

// ============ Routing API ============

func (h *Handler) RoutingTable(c *gin.Context) {
	var rules []models.RoutingRule
	if err := h.db.Find(&rules).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error loading routing rules")
		return
	}

	c.HTML(http.StatusOK, "components/routing-table.html", gin.H{
		"Rules": rules,
	})
}

func (h *Handler) NewRoutingForm(c *gin.Context) {
	var outbounds []models.Outbound
	h.db.Find(&outbounds)

	var inbounds []models.Inbound
	h.db.Find(&inbounds)

	c.HTML(http.StatusOK, "components/routing-form.html", gin.H{
		"Outbounds": outbounds,
		"Inbounds":  inbounds,
	})
}

func (h *Handler) EditRoutingForm(c *gin.Context) {
	id := c.Param("id")
	var rule models.RoutingRule
	if err := h.db.First(&rule, "id = ?", id).Error; err != nil {
		c.String(http.StatusNotFound, "Routing rule not found")
		return
	}

	var outbounds []models.Outbound
	h.db.Find(&outbounds)

	var inbounds []models.Inbound
	h.db.Find(&inbounds)

	c.HTML(http.StatusOK, "components/routing-form.html", gin.H{
		"Rule":      rule,
		"Outbounds": outbounds,
		"Inbounds":  inbounds,
	})
}

func (h *Handler) CreateRouting(c *gin.Context) {
	var rule models.RoutingRule
	if err := c.ShouldBind(&rule); err != nil {
		c.String(http.StatusBadRequest, "Invalid input")
		return
	}

	if err := h.db.Create(&rule).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error creating routing rule")
		return
	}

	h.RoutingTable(c)
}

func (h *Handler) UpdateRouting(c *gin.Context) {
	id := c.Param("id")
	var rule models.RoutingRule
	if err := c.ShouldBind(&rule); err != nil {
		c.String(http.StatusBadRequest, "Invalid input")
		return
	}

	if err := h.db.Model(&models.RoutingRule{}).Where("id = ?", id).Updates(&rule).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error updating routing rule")
		return
	}

	h.RoutingTable(c)
}

func (h *Handler) DeleteRouting(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&models.RoutingRule{}, "id = ?", id).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error deleting routing rule")
		return
	}

	c.String(http.StatusOK, "")
}
