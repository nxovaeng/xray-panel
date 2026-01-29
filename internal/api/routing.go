package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

// CertificateInfo represents a discovered certificate
type CertificateInfo struct {
	Domain   string `json:"domain"`
	CertPath string `json:"cert_path"`
	KeyPath  string `json:"key_path"`
	Exists   bool   `json:"exists"`
}

// handleScanCertificates scans the certificate directory for available certificates
func (s *Server) handleScanCertificates(c *gin.Context) {
	certDir := s.config.Nginx.CertDir
	
	// Check if directory exists
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		jsonError(c, http.StatusNotFound, "Certificate directory not found: "+certDir)
		return
	}

	var certificates []CertificateInfo

	// Scan the certificate directory
	// Let's Encrypt structure: /etc/letsencrypt/live/domain.com/
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
		
		// Skip README directory
		if domain == "README" {
			continue
		}

		domainPath := filepath.Join(certDir, domain)
		certPath := filepath.Join(domainPath, "fullchain.pem")
		keyPath := filepath.Join(domainPath, "privkey.pem")

		// Check if both cert and key exist
		certExists := fileExists(certPath)
		keyExists := fileExists(keyPath)

		if certExists && keyExists {
			certificates = append(certificates, CertificateInfo{
				Domain:   domain,
				CertPath: certPath,
				KeyPath:  keyPath,
				Exists:   true,
			})
		}
	}

	// Also scan for custom certificate structures
	// Support: /etc/letsencrypt/domain.crt and domain.key
	customEntries, _ := os.ReadDir(certDir)
	for _, entry := range customEntries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		
		// Look for .crt files
		if strings.HasSuffix(name, ".crt") || strings.HasSuffix(name, ".pem") {
			domain := strings.TrimSuffix(strings.TrimSuffix(name, ".crt"), ".pem")
			certPath := filepath.Join(certDir, name)
			
			// Try to find corresponding key file
			keyPath := ""
			for _, keyExt := range []string{".key", ".pem", "-key.pem"} {
				testKeyPath := filepath.Join(certDir, domain+keyExt)
				if fileExists(testKeyPath) {
					keyPath = testKeyPath
					break
				}
			}

			if keyPath != "" && fileExists(certPath) {
				// Check if already added
				found := false
				for _, cert := range certificates {
					if cert.Domain == domain {
						found = true
						break
					}
				}

				if !found {
					certificates = append(certificates, CertificateInfo{
						Domain:   domain,
						CertPath: certPath,
						KeyPath:  keyPath,
						Exists:   true,
					})
				}
			}
		}
	}

	jsonOK(c, gin.H{
		"cert_dir":     certDir,
		"certificates": certificates,
		"count":        len(certificates),
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
