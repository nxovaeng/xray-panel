package web

import (
	cryptorand "crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"xray-panel/internal/logger"
	"xray-panel/internal/models"
	"xray-panel/internal/nginx"
	"xray-panel/internal/system"
	"xray-panel/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db    *gorm.DB
	nginx *nginx.ConfigGenerator
}

func NewHandler(db *gorm.DB, nginxGen *nginx.ConfigGenerator) *Handler {
	return &Handler{db: db, nginx: nginxGen}
}

// ============ Page Handlers ============

func (h *Handler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func (h *Handler) renderPage(c *gin.Context, template string, data gin.H) {
	user, exists := c.Get("user")
	if !exists {
		user = nil
	}

	if data == nil {
		data = gin.H{}
	}
	data["User"] = user
	data["PanelMode"] = models.GetPanelMode(h.db)

	c.HTML(http.StatusOK, template, data)
}

func (h *Handler) DashboardPage(c *gin.Context) {
	h.renderPage(c, "dashboard", gin.H{
		"Title": "Dashboard",
		"Page":  "dashboard",
	})
}

func (h *Handler) UsersPage(c *gin.Context) {
	h.renderPage(c, "users", gin.H{
		"Title": "Users",
		"Page":  "users",
	})
}

func (h *Handler) InboundsPage(c *gin.Context) {
	h.renderPage(c, "inbounds", gin.H{
		"Title": "Inbounds",
		"Page":  "inbounds",
	})
}

func (h *Handler) OutboundsPage(c *gin.Context) {
	h.renderPage(c, "outbounds", gin.H{
		"Title": "Outbounds",
		"Page":  "outbounds",
	})
}

func (h *Handler) RoutingPage(c *gin.Context) {
	h.renderPage(c, "routing", gin.H{
		"Title": "Routing",
		"Page":  "routing",
	})
}

func (h *Handler) DomainsPage(c *gin.Context) {
	h.renderPage(c, "domains", gin.H{
		"Title": "Domains",
		"Page":  "domains",
	})
}

func (h *Handler) SettingsPage(c *gin.Context) {
	h.renderPage(c, "settings", gin.H{
		"Title": "Settings",
		"Page":  "settings",
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

	// Get system info (includes network traffic)
	sysInfo, err := system.GetSystemInfo()
	if err != nil {
		// If system info fails, use defaults
		sysInfo = &system.SystemInfo{}
	}

	stats := struct {
		// User stats
		TotalUsers    int64 `json:"total_users"`
		ActiveUsers   int64 `json:"active_users"`
		TotalInbounds int64 `json:"total_inbounds"`

		// Network traffic (system-wide)
		NetUpload   string `json:"net_upload"`
		NetDownload string `json:"net_download"`
		NetTotal    string `json:"net_total"`

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

		NetUpload:   system.FormatBytes(sysInfo.NetBytesSent),
		NetDownload: system.FormatBytes(sysInfo.NetBytesRecv),
		NetTotal:    system.FormatBytes(sysInfo.NetBytesSent + sysInfo.NetBytesRecv),

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
	// 优先检查 X-Forwarded-Proto 头（当面板通过 Nginx 反代时）
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	// 获取主机名，优先使用 X-Forwarded-Host
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	baseURL := scheme + "://" + host

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

	// Use Save to avoid GORM skipping zero-value bool fields (e.g. Enabled=false)
	user.ID = existingUser.ID
	if err := h.db.Save(&user).Error; err != nil {
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
	if err := h.db.Where("email LIKE ? OR name LIKE ?", "%"+query+"%", "%"+query+"%").Find(&users).Error; err != nil {
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

	// WireGuard specific fields (ShouldBind handles most, but secret key needs special care)
	// ExcludeFromSub is a bool from select, parse manually
	inbound.ExcludeFromSub = c.PostForm("exclude_from_sub") == "true"
	if mtuStr := c.PostForm("wg_mtu"); mtuStr != "" {
		if mtu, err := strconv.Atoi(mtuStr); err == nil && mtu > 0 {
			inbound.WGMTU = mtu
		}
	}

	// WireGuard inbound: force UseUDS=false and skip domain logic
	if inbound.Protocol == models.ProtocolWireGuard {
		inbound.UseUDS = false
		inbound.DomainID = ""
		inbound.ActualDomain = ""
		if err := h.db.Create(&inbound).Error; err != nil {
			logger.Error("Failed to create wireguard inbound %s: %v", inbound.Tag, err)
			c.String(http.StatusInternalServerError, "Error creating inbound")
			return
		}
		logger.Info("WireGuard inbound created: %s (Port: %d)", inbound.Tag, inbound.Port)
		h.InboundsTable(c)
		return
	}

	// Get domain_id from form (handle empty string)
	domainID := c.PostForm("domain_id")
	if domainID != "" {
		inbound.DomainID = domainID
		logger.Info("CreateInbound: DomainID from form: %s", domainID)

		var domain models.Domain
		if err := h.db.First(&domain, "id = ?", domainID).Error; err == nil {
			logger.Info("CreateInbound: Found domain: %s (ID: %s)", domain.Domain, domain.ID)

			// Check if domain is a wildcard certificate
			// Use IsWildcard field from database (set during certificate import)
			// or fallback to checking domain prefix for backward compatibility
			isWildcard := domain.IsWildcard || strings.HasPrefix(domain.Domain, "*.")
			if isWildcard {
				baseDomain := strings.TrimPrefix(domain.Domain, "*.")
				subdomain := generateRandomSubdomain()
				inbound.ActualDomain = subdomain + "." + baseDomain
				logger.Info("CreateInbound: Generated subdomain for wildcard cert: %s (base: %s, wildcard: %s, IsWildcard: %v)",
					inbound.ActualDomain, baseDomain, domain.Domain, domain.IsWildcard)
			} else {
				// For non-wildcard domains, use the domain directly
				inbound.ActualDomain = domain.Domain
				logger.Info("CreateInbound: Domain is not wildcard: %s", domain.Domain)
			}
		} else {
			logger.Warn("CreateInbound: Failed to find domain with ID %s: %v", domainID, err)
		}
	} else {
		logger.Info("CreateInbound: No DomainID provided")
	}

	if err := h.db.Create(&inbound).Error; err != nil {
		logger.Error("Failed to create inbound %s: %v", inbound.Tag, err)
		c.String(http.StatusInternalServerError, "Error creating inbound")
		return
	}

	logger.Info("Inbound created: %s (Protocol: %s, Port: %d, DomainID: %s, ActualDomain: %s)",
		inbound.Tag, inbound.Protocol, inbound.Port, inbound.DomainID, inbound.ActualDomain)

	// Generate Nginx config if domain is set
	if inbound.DomainID != "" && h.nginx != nil {
		if err := h.generateNginxConfigForInbound(&inbound); err != nil {
			logger.Error("Failed to generate Nginx config for inbound %s: %v", inbound.Tag, err)
			// Don't fail the request, just log the error
		} else {
			logger.Info("Nginx config generated for inbound %s", inbound.Tag)
		}
	}

	h.InboundsTable(c)
}

func (h *Handler) UpdateInbound(c *gin.Context) {
	id := c.Param("id")

	// Get existing inbound first
	var existingInbound models.Inbound
	if err := h.db.First(&existingInbound, "id = ?", id).Error; err != nil {
		c.String(http.StatusNotFound, "Inbound not found")
		return
	}

	// Manually parse form fields to avoid GORM zero-value skipping issue
	// and bool-from-string binding issues with ShouldBind
	existingInbound.Tag = c.PostForm("tag")
	existingInbound.Protocol = models.Protocol(c.PostForm("protocol"))
	existingInbound.Transport = models.Transport(c.PostForm("transport"))
	existingInbound.Path = c.PostForm("path")
	existingInbound.ServiceName = c.PostForm("service_name")
	existingInbound.Host = c.PostForm("host")
	existingInbound.Mode = c.PostForm("mode")
	existingInbound.Remark = c.PostForm("remark")
	existingInbound.ConnectDomain = c.PostForm("connect_domain")
	existingInbound.CustomSNI = c.PostForm("custom_sni")

	// WireGuard specific fields
	if c.PostForm("wg_secret_key") != "" {
		existingInbound.WGSecretKey = c.PostForm("wg_secret_key")
	}
	existingInbound.WGPublicKey = c.PostForm("wg_public_key")
	existingInbound.WGPeerPubKey = c.PostForm("wg_peer_pub_key")
	existingInbound.WGLocalIP = c.PostForm("wg_local_ip")
	if mtuStr := c.PostForm("wg_mtu"); mtuStr != "" {
		if mtu, err := strconv.Atoi(mtuStr); err == nil && mtu > 0 {
			existingInbound.WGMTU = mtu
		}
	}

	// ExcludeFromSub
	existingInbound.ExcludeFromSub = c.PostForm("exclude_from_sub") == "true"

	// Parse port (keep existing if not provided)
	if portStr := c.PostForm("port"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			existingInbound.Port = port
		}
	}

	// WireGuard inbound: skip UDS and domain logic
	if existingInbound.Protocol == models.ProtocolWireGuard {
		existingInbound.UseUDS = false
		existingInbound.DomainID = ""
		existingInbound.ActualDomain = ""
		if err := h.db.Save(&existingInbound).Error; err != nil {
			c.String(http.StatusInternalServerError, "Error updating inbound")
			return
		}
		logger.Info("WireGuard inbound updated: %s", existingInbound.Tag)
		h.InboundsTable(c)
		return
	}

	// Parse use_uds (string "true"/"false" from HTML select)
	existingInbound.UseUDS = c.PostForm("use_uds") == "true"

	// Set defaults
	if existingInbound.Protocol == "" {
		existingInbound.Protocol = models.ProtocolVLESS
	}
	if existingInbound.Listen == "" {
		existingInbound.Listen = "127.0.0.1"
	}

	// Handle domain_id and ActualDomain
	domainID := c.PostForm("domain_id")
	if domainID != "" {
		if domainID != existingInbound.DomainID {
			// Domain changed — recalculate ActualDomain
			var domain models.Domain
			if err := h.db.First(&domain, "id = ?", domainID).Error; err == nil {
				isWildcard := domain.IsWildcard || strings.HasPrefix(domain.Domain, "*.")
				if isWildcard {
					baseDomain := strings.TrimPrefix(domain.Domain, "*.")
					existingInbound.ActualDomain = generateRandomSubdomain() + "." + baseDomain
				} else {
					existingInbound.ActualDomain = domain.Domain
				}
			}
		}
		existingInbound.DomainID = domainID
	} else {
		existingInbound.DomainID = ""
		existingInbound.ActualDomain = ""
	}

	if err := h.db.Save(&existingInbound).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error updating inbound")
		return
	}

	inbound := existingInbound
	logger.Info("UpdateInbound: Inbound updated: %s (DomainID: %s, ActualDomain: %s)", id, inbound.DomainID, inbound.ActualDomain)

	// Regenerate Nginx config if domain is set
	if h.nginx != nil {
		// First cleanup old configs
		if err := h.nginx.CleanupInboundConfigs(id); err != nil {
			logger.Warn("Failed to cleanup old Nginx configs for inbound %s: %v", id, err)
		}

		// Then generate new config if domain is set
		if inbound.DomainID != "" {
			// Reload inbound with domain preloaded
			var updatedInbound models.Inbound
			if err := h.db.Preload("Domain").First(&updatedInbound, "id = ?", id).Error; err == nil {
				if err := h.generateNginxConfigForInbound(&updatedInbound); err != nil {
					logger.Error("Failed to generate Nginx config for inbound %s: %v", updatedInbound.Tag, err)
				} else {
					logger.Info("Nginx config regenerated for inbound %s", updatedInbound.Tag)
				}
			}
		}
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

	// Cleanup Nginx configs before deleting inbound
	if h.nginx != nil {
		if err := h.nginx.CleanupInboundConfigs(id); err != nil {
			logger.Warn("Failed to cleanup Nginx configs for inbound %s: %v", id, err)
		} else {
			logger.Info("Nginx configs cleaned up for inbound %s", id)
		}
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

func (h *Handler) ImportOutboundForm(c *gin.Context) {
	c.HTML(http.StatusOK, "components/import-outbound-form.html", nil)
}

func (h *Handler) ImportShareLink(c *gin.Context) {
	link := c.PostForm("link")
	if link == "" {
		c.String(http.StatusBadRequest, "链接不能为空")
		return
	}

	outbound, err := utils.ParseShareLink(link)
	if err != nil {
		logger.Error("解析分享链接失败: %v", err)
		c.String(http.StatusBadRequest, "解析失败: "+err.Error())
		return
	}

	outbound.CreatedAt = time.Now()
	outbound.UpdatedAt = time.Now()

	if err := h.db.Create(&outbound).Error; err != nil {
		logger.Error("导入出站失败: %v", err)
		c.String(http.StatusInternalServerError, "保存记录失败: "+err.Error())
		return
	}

	logger.Info("导入出站成功: %s", outbound.Tag)
	h.OutboundsTable(c)
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

	// Handle WireGuard-specific fields (wg_server -> Server, wg_port -> Port)
	if outbound.Type == models.OutboundWireGuard {
		wgServer := c.PostForm("wg_server")
		wgPortStr := c.PostForm("wg_port")
		if wgServer != "" {
			outbound.Server = wgServer
		}
		if wgPortStr != "" {
			if port, err := strconv.Atoi(wgPortStr); err == nil && port > 0 {
				outbound.Port = port
			}
		}
	}

	// Handle Trojan-specific fields (trojan_server -> Server, trojan_port -> Port)
	if outbound.Type == models.OutboundTrojan {
		trojanServer := c.PostForm("trojan_server")
		trojanPortStr := c.PostForm("trojan_port")
		if trojanServer != "" {
			outbound.Server = trojanServer
		}
		if trojanPortStr != "" {
			if port, err := strconv.Atoi(trojanPortStr); err == nil && port > 0 {
				outbound.Port = port
			}
		}
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

	// Handle WireGuard-specific fields (wg_server -> Server, wg_port -> Port)
	if outbound.Type == models.OutboundWireGuard {
		wgServer := c.PostForm("wg_server")
		wgPortStr := c.PostForm("wg_port")
		if wgServer != "" {
			outbound.Server = wgServer
		}
		if wgPortStr != "" {
			if port, err := strconv.Atoi(wgPortStr); err == nil && port > 0 {
				outbound.Port = port
			}
		}
	}

	// Handle Trojan-specific fields (trojan_server -> Server, trojan_port -> Port)
	if outbound.Type == models.OutboundTrojan {
		trojanServer := c.PostForm("trojan_server")
		trojanPortStr := c.PostForm("trojan_port")
		if trojanServer != "" {
			outbound.Server = trojanServer
		}
		if trojanPortStr != "" {
			if port, err := strconv.Atoi(trojanPortStr); err == nil && port > 0 {
				outbound.Port = port
			}
		}
	}

	// Set update timestamp
	outbound.UpdatedAt = time.Now()

	// Use Save to avoid GORM skipping zero-value bool fields (e.g. Enabled=false)
	var existing models.Outbound
	if err := h.db.First(&existing, "id = ?", id).Error; err != nil {
		c.String(http.StatusNotFound, "Outbound not found")
		return
	}
	outbound.ID = existing.ID
	if err := h.db.Save(&outbound).Error; err != nil {
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

	// Use Save to avoid GORM skipping zero-value bool fields (e.g. Enabled=false)
	var existing models.RoutingRule
	if err := h.db.First(&existing, "id = ?", id).Error; err != nil {
		c.String(http.StatusNotFound, "Routing rule not found")
		return
	}
	rule.ID = existing.ID
	if err := h.db.Save(&rule).Error; err != nil {
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

// ============ Nginx Config Helper ============

// generateNginxConfigForInbound generates Nginx reverse proxy config for a single inbound
func (h *Handler) generateNginxConfigForInbound(inbound *models.Inbound) error {
	if h.nginx == nil {
		return fmt.Errorf("nginx generator not configured")
	}

	if inbound.Domain == nil {
		// Try to load domain if not preloaded
		if inbound.DomainID != "" {
			var domain models.Domain
			if err := h.db.First(&domain, "id = ?", inbound.DomainID).Error; err != nil {
				return fmt.Errorf("failed to load domain: %v", err)
			}
			inbound.Domain = &domain
		} else {
			return fmt.Errorf("inbound has no domain configured")
		}
	}

	// Generate config for this single inbound
	inbounds := []models.Inbound{*inbound}
	if err := h.nginx.GenerateHTTPConfig(inbounds); err != nil {
		return fmt.Errorf("failed to generate HTTP config: %v", err)
	}

	// Reload Nginx
	if err := h.nginx.Reload(); err != nil {
		logger.Warn("Failed to reload Nginx: %v", err)
		// Don't return error, config was generated successfully
	}

	return nil
}
