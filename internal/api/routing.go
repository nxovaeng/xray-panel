package api

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xray-panel/internal/models"
)

// handleListRoutingRules returns all routing rules
func (s *Server) handleListRoutingRules(c *gin.Context) {
	var rules []models.RoutingRule
	if err := s.db.Order("priority ASC").Find(&rules).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to fetch routing rules")
		return
	}
	jsonOK(c, rules)
}

// handleCreateRoutingRule creates a new routing rule
func (s *Server) handleCreateRoutingRule(c *gin.Context) {
	var rule models.RoutingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	rule.Enabled = true

	if err := s.db.Create(&rule).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to create routing rule")
		return
	}

	jsonCreated(c, rule)
}

// handleUpdateRoutingRule updates a routing rule
func (s *Server) handleUpdateRoutingRule(c *gin.Context) {
	id := c.Param("id")

	var rule models.RoutingRule
	if err := s.db.First(&rule, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "Routing rule not found")
		return
	}

	var req models.RoutingRule
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	// Update fields
	rule.Name = req.Name
	rule.Type = req.Type
	rule.Domains = req.Domains
	rule.IPs = req.IPs
	rule.GeoIPCodes = req.GeoIPCodes
	rule.GeoSiteTags = req.GeoSiteTags
	rule.Protocols = req.Protocols
	rule.OutboundTag = req.OutboundTag
	rule.Priority = req.Priority
	rule.Enabled = req.Enabled
	rule.Remark = req.Remark

	if err := s.db.Save(&rule).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to update routing rule")
		return
	}

	jsonOK(c, rule)
}

// handleDeleteRoutingRule deletes a routing rule
func (s *Server) handleDeleteRoutingRule(c *gin.Context) {
	id := c.Param("id")

	result := s.db.Delete(&models.RoutingRule{}, "id = ?", id)
	if result.Error != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to delete routing rule")
		return
	}
	if result.RowsAffected == 0 {
		jsonError(c, http.StatusNotFound, "Routing rule not found")
		return
	}

	jsonOK(c, gin.H{"deleted": true})
}

// handleImportPresetRules imports preset routing rules
func (s *Server) handleImportPresetRules(c *gin.Context) {
	presetName := c.Param("preset")

	presets := models.PresetRoutingRules()
	rules, exists := presets[presetName]
	if !exists {
		c.String(http.StatusNotFound, "预设模板不存在: "+presetName)
		return
	}

	// Import rules
	imported := 0
	for _, rule := range rules {
		// Check if rule with same name already exists
		var existing models.RoutingRule
		if err := s.db.Where("name = ?", rule.Name).First(&existing).Error; err == nil {
			// Rule exists, skip
			continue
		}

		// Create new rule
		if err := s.db.Create(&rule).Error; err != nil {
			// Log error but continue with other rules
			continue
		}
		imported++
	}

	// Return updated routing table HTML (same as RoutingTable handler)
	var allRules []models.RoutingRule
	if err := s.db.Order("priority ASC").Find(&allRules).Error; err != nil {
		c.String(http.StatusInternalServerError, "加载路由规则失败")
		return
	}

	c.HTML(http.StatusOK, "components/routing-table.html", gin.H{
		"Rules": allRules,
	})
}

// handleListDomains returns all domains
func (s *Server) handleListDomains(c *gin.Context) {
	var domains []models.Domain
	if err := s.db.Order("created_at DESC").Find(&domains).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to fetch domains")
		return
	}
	jsonOK(c, domains)
}

// CertificateInfo represents a discovered certificate with parsed info
type CertificateInfo struct {
	Domain       string    `json:"domain"`
	CertPath     string    `json:"cert_path"`
	KeyPath      string    `json:"key_path"`
	ExpiryDate   time.Time `json:"expiry_date"`
	DaysToExpiry int       `json:"days_to_expiry"`
	Status       string    `json:"status"` // "正常" | "即将过期" | "已过期"
	Issuer       string    `json:"issuer"`
	IsWildcard   bool      `json:"is_wildcard"`
	Exists       bool      `json:"exists"`
}

// parseCertificate reads and parses certificate info from a PEM file
func parseCertificate(certPath string) (issuer string, expiry time.Time, err error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return "", time.Time{}, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return "", time.Time{}, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", time.Time{}, err
	}

	return cert.Issuer.CommonName, cert.NotAfter, nil
}

// getCertificateStatus returns status based on expiry date
func getCertificateStatus(expiry time.Time) (string, int) {
	now := time.Now()
	daysToExpiry := int(expiry.Sub(now).Hours() / 24)

	if daysToExpiry < 0 {
		return "已过期", daysToExpiry
	} else if daysToExpiry <= 30 {
		return "即将过期", daysToExpiry
	}
	return "正常", daysToExpiry
}

// handleScanCertificates scans the certificate directory for available certificates
// Supports both acme.sh (/root/.acme.sh) and Let's Encrypt (/etc/letsencrypt/live) structures
func (s *Server) handleScanCertificates(c *gin.Context) {
	certDir := s.config.Nginx.CertDir

	// Check if directory exists
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		jsonError(c, http.StatusNotFound, "Certificate directory not found: "+certDir)
		return
	}

	var certificates []CertificateInfo

	// Scan the certificate directory
	entries, err := os.ReadDir(certDir)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to read certificate directory: "+err.Error())
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		domain := entry.Name()

		// Skip hidden directories and special directories
		if strings.HasPrefix(domain, ".") || domain == "README" || domain == "ca" {
			continue
		}

		domainPath := filepath.Join(certDir, domain)

		// Try acme.sh structure first
		// acme.sh stores certs as: /root/.acme.sh/<domain>/<domain>.cer or fullchain.cer
		var certPath, keyPath string

		// Check for acme.sh structure
		acmeCertPath := filepath.Join(domainPath, domain+".cer")
		acmeFullchainPath := filepath.Join(domainPath, "fullchain.cer")
		acmeKeyPath := filepath.Join(domainPath, domain+".key")

		if fileExists(acmeFullchainPath) {
			certPath = acmeFullchainPath
		} else if fileExists(acmeCertPath) {
			certPath = acmeCertPath
		}

		if fileExists(acmeKeyPath) {
			keyPath = acmeKeyPath
		}

		// If not found, try Let's Encrypt structure
		// Let's Encrypt stores as: /etc/letsencrypt/live/<domain>/fullchain.pem
		if certPath == "" || keyPath == "" {
			leCertPath := filepath.Join(domainPath, "fullchain.pem")
			leKeyPath := filepath.Join(domainPath, "privkey.pem")

			if fileExists(leCertPath) && fileExists(leKeyPath) {
				certPath = leCertPath
				keyPath = leKeyPath
			}
		}

		// If no valid cert/key pair found, skip
		if certPath == "" || keyPath == "" {
			continue
		}

		// Parse certificate info
		issuer, expiry, parseErr := parseCertificate(certPath)
		status, daysToExpiry := getCertificateStatus(expiry)
		if parseErr != nil {
			// If parsing fails, still add with basic info
			issuer = "Unknown"
			expiry = time.Time{}
			status = "未知"
			daysToExpiry = 0
		}

		// Check if it's a wildcard certificate
		isWildcard := strings.HasPrefix(domain, "_wildcard") || strings.HasPrefix(domain, "*.")
		// Also check from certificate itself if we parsed it successfully
		if parseErr == nil && !isWildcard {
			// acme.sh uses _wildcard prefix for wildcard domains
			// e.g., _wildcard.example.com for *.example.com
			if strings.HasPrefix(domain, "_") {
				isWildcard = true
			}
		}

		certificates = append(certificates, CertificateInfo{
			Domain:       domain,
			CertPath:     certPath,
			KeyPath:      keyPath,
			ExpiryDate:   expiry,
			DaysToExpiry: daysToExpiry,
			Status:       status,
			Issuer:       issuer,
			IsWildcard:   isWildcard,
			Exists:       true,
		})
	}

	jsonOK(c, gin.H{
		"cert_dir":     certDir,
		"certificates": certificates,
		"count":        len(certificates),
	})
}

// handleScanAndImportCertificates scans certificates and imports them as domains
func (s *Server) handleScanAndImportCertificates(c *gin.Context) {
	certDir := s.config.Nginx.CertDir

	// Check if directory exists
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		c.String(http.StatusNotFound, "证书目录不存在: "+certDir)
		return
	}

	entries, err := os.ReadDir(certDir)
	if err != nil {
		c.String(http.StatusInternalServerError, "读取证书目录失败: "+err.Error())
		return
	}

	importedCount := 0
	skippedCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		domain := entry.Name()

		// Skip hidden directories and special directories
		if strings.HasPrefix(domain, ".") || domain == "README" || domain == "ca" {
			continue
		}

		domainPath := filepath.Join(certDir, domain)

		// Find certificate files (acme.sh or Let's Encrypt structure)
		var certPath, keyPath string

		// Check for acme.sh structure
		acmeFullchainPath := filepath.Join(domainPath, "fullchain.cer")
		acmeCertPath := filepath.Join(domainPath, domain+".cer")
		acmeKeyPath := filepath.Join(domainPath, domain+".key")

		if fileExists(acmeFullchainPath) {
			certPath = acmeFullchainPath
		} else if fileExists(acmeCertPath) {
			certPath = acmeCertPath
		}

		if fileExists(acmeKeyPath) {
			keyPath = acmeKeyPath
		}

		// Try Let's Encrypt structure
		if certPath == "" || keyPath == "" {
			leCertPath := filepath.Join(domainPath, "fullchain.pem")
			leKeyPath := filepath.Join(domainPath, "privkey.pem")

			if fileExists(leCertPath) && fileExists(leKeyPath) {
				certPath = leCertPath
				keyPath = leKeyPath
			}
		}

		if certPath == "" || keyPath == "" {
			continue
		}

		// Convert acme.sh domain format to actual domain
		// acme.sh uses _wildcard.example.com for *.example.com
		actualDomain := domain
		if strings.HasPrefix(domain, "_wildcard.") {
			actualDomain = "*." + strings.TrimPrefix(domain, "_wildcard.")
		}

		// Check if domain already exists
		var existingDomain models.Domain
		if err := s.db.Where("domain = ?", actualDomain).First(&existingDomain).Error; err == nil {
			// Domain exists, update cert paths if different
			if existingDomain.CertPath != certPath || existingDomain.KeyPath != keyPath {
				existingDomain.CertPath = certPath
				existingDomain.KeyPath = keyPath
				s.db.Save(&existingDomain)
			}
			skippedCount++
			continue
		}

		// Create new domain
		newDomain := models.Domain{
			Domain:   actualDomain,
			Type:     models.DomainTypeDirect,
			CertPath: certPath,
			KeyPath:  keyPath,
			Enabled:  true,
		}

		if err := s.db.Create(&newDomain).Error; err != nil {
			continue
		}

		importedCount++
	}

	// Return updated domains table
	var domains []models.Domain
	if err := s.db.Order("created_at DESC").Find(&domains).Error; err != nil {
		c.String(http.StatusInternalServerError, "加载域名列表失败")
		return
	}

	// Set response header to notify about import results
	c.Header("HX-Trigger", fmt.Sprintf(`{"showNotification": {"type": "success", "message": "导入完成：新增 %d 个，跳过 %d 个"}}`, importedCount, skippedCount))

	c.HTML(http.StatusOK, "components/domains-table.html", gin.H{
		"Domains": domains,
	})
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// handleCreateDomain creates a new domain
func (s *Server) handleCreateDomain(c *gin.Context) {
	var domain models.Domain
	if err := c.ShouldBindJSON(&domain); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	domain.Enabled = true

	if err := s.db.Create(&domain).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to create domain")
		return
	}

	jsonCreated(c, domain)
}

// handleUpdateDomain updates a domain
func (s *Server) handleUpdateDomain(c *gin.Context) {
	id := c.Param("id")

	var domain models.Domain
	if err := s.db.First(&domain, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "Domain not found")
		return
	}

	var req models.Domain
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	// Update fields
	domain.Domain = req.Domain
	domain.Type = req.Type
	domain.ServerName = req.ServerName
	domain.Fingerprint = req.Fingerprint
	domain.ShortID = req.ShortID
	domain.CertPath = req.CertPath
	domain.KeyPath = req.KeyPath
	domain.Enabled = req.Enabled

	// Only update private key if provided
	if req.PrivateKey != "" {
		domain.PrivateKey = req.PrivateKey
	}
	if req.PublicKey != "" {
		domain.PublicKey = req.PublicKey
	}

	if err := s.db.Save(&domain).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to update domain")
		return
	}

	jsonOK(c, domain)
}

// handleDeleteDomain deletes a domain
func (s *Server) handleDeleteDomain(c *gin.Context) {
	id := c.Param("id")

	result := s.db.Delete(&models.Domain{}, "id = ?", id)
	if result.Error != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to delete domain")
		return
	}
	if result.RowsAffected == 0 {
		jsonError(c, http.StatusNotFound, "Domain not found")
		return
	}

	jsonOK(c, gin.H{"deleted": true})
}
