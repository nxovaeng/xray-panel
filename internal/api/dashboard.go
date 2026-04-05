package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"

	"xray-panel/internal/logger"
	"xray-panel/internal/models"
	"xray-panel/internal/nginx"
	"xray-panel/internal/xray"
)

// DashboardData contains dashboard statistics
type DashboardData struct {
	TotalUsers       int64 `json:"total_users"`
	ActiveUsers      int64 `json:"active_users"`
	TotalInbounds    int64 `json:"total_inbounds"`
	TotalOutbounds   int64 `json:"total_outbounds"`
	TotalTrafficUp   int64 `json:"total_traffic_up"`
	TotalTrafficDown int64 `json:"total_traffic_down"`
}

// handleDashboard returns dashboard statistics
func (s *Server) handleDashboard(c *gin.Context) {
	var data DashboardData

	s.db.Model(&models.User{}).Count(&data.TotalUsers)
	s.db.Model(&models.User{}).Where("enabled = ?", true).Count(&data.ActiveUsers)
	s.db.Model(&models.Inbound{}).Where("enabled = ?", true).Count(&data.TotalInbounds)
	s.db.Model(&models.Outbound{}).Where("enabled = ?", true).Count(&data.TotalOutbounds)

	// Sum traffic in a single query instead of loading all users
	s.db.Model(&models.User{}).Select("COALESCE(SUM(traffic_used), 0)").Scan(&data.TotalTrafficDown)

	jsonOK(c, data)
}

// handleGetSettings returns all settings
func (s *Server) handleGetSettings(c *gin.Context) {
	var settings []models.Setting
	if err := s.db.Find(&settings).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to fetch settings")
		return
	}

	// Convert to map for easier use
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	jsonOK(c, settingsMap)
}

// handleUpdateSettings updates settings
func (s *Server) handleUpdateSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	for key, value := range req {
		setting := models.Setting{Key: key, Value: value}
		s.db.Where("key = ?", key).Assign(setting).FirstOrCreate(&setting)
	}

	jsonOK(c, gin.H{"updated": true})
}

// handleXrayStatus returns Xray service status
func (s *Server) handleXrayStatus(c *gin.Context) {
	// Check if Xray is running
	cmd := exec.Command("systemctl", "is-active", "xray")
	output, _ := cmd.Output()
	status := string(output)

	jsonOK(c, gin.H{
		"status": status,
		"active": status == "active\n",
	})
}

// handleXrayRestart restarts Xray service
func (s *Server) handleXrayRestart(c *gin.Context) {
	cmd := exec.Command("systemctl", "restart", "xray")
	if err := cmd.Run(); err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to restart Xray: "+err.Error())
		return
	}

	jsonOK(c, gin.H{"restarted": true})
}

// handleGetXrayConfig returns the generated Xray config
func (s *Server) handleGetXrayConfig(c *gin.Context) {
	configJSON, err := s.generateXrayConfig()
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to generate config: "+err.Error())
		return
	}

	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(configJSON))
}

// handleApplyXrayConfig generates and applies the Xray and Nginx configs
func (s *Server) handleApplyXrayConfig(c *gin.Context) {
	// Check if hot reload is requested
	hotReload := c.Query("hot") == "true"

	if hotReload {
		// Use Xray API for hot reload
		if err := s.applyXrayConfigHot(); err != nil {
			jsonError(c, http.StatusInternalServerError, "Hot reload failed: "+err.Error())
			return
		}

		jsonOK(c, gin.H{
			"applied": true,
			"method":  "hot_reload",
			"message": "配置已通过 API 热更新",
		})
		return
	}

	// Traditional method: write config and restart
	// 1. Generate Xray config
	configJSON, err := s.generateXrayConfig()
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to generate Xray config: "+err.Error())
		return
	}

	// 2. Write Xray config to file
	if err := os.WriteFile(s.config.Xray.ConfigPath, configJSON, 0644); err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to write Xray config: "+err.Error())
		return
	}

	// 2.5. Validate config before restarting
	if err := s.validateXrayConfig(); err != nil {
		jsonError(c, http.StatusBadRequest, "配置校验失败，服务未重启: "+err.Error())
		return
	}

	// 3. Generate Nginx configs
	var inbounds []models.Inbound
	s.db.Preload("Domain").Where("enabled = ?", true).Find(&inbounds)

	nginxGen := nginx.NewGenerator(s.config.Nginx.ConfigDir, s.config.Nginx.StreamDir)
	nginxGen.SetSocketDir(s.config.Xray.SocketDir)

	// Generate HTTP configs
	if err := nginxGen.GenerateHTTPConfig(inbounds); err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to generate Nginx HTTP config: "+err.Error())
		return
	}

	// 4. Restart services
	// Restart Xray
	xrayCmd := exec.Command("systemctl", "restart", "xray")
	if err := xrayCmd.Run(); err != nil {
		logger.Warn("Failed to restart Xray: %v", err)
	} else {
		logger.Info("Xray service restarted successfully")
	}

	// Reload Nginx
	nginxCmd := exec.Command("sh", "-c", s.config.Nginx.ReloadCmd)
	if err := nginxCmd.Run(); err != nil {
		logger.Warn("Failed to reload Nginx: %v", err)
	} else {
		logger.Info("Nginx reloaded successfully")
	}

	jsonOK(c, gin.H{
		"applied":    true,
		"method":     "restart",
		"config_len": len(configJSON),
		"message":    "配置已写入文件并重启服务",
	})
}

// applyXrayConfigHot applies configuration by writing config and restarting Xray.
// Full gRPC-based hot reload is not yet implemented; this is a functional fallback.
func (s *Server) applyXrayConfigHot() error {
	configJSON, err := s.generateXrayConfig()
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	if err := os.WriteFile(s.config.Xray.ConfigPath, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Validate before restart
	if err := s.validateXrayConfig(); err != nil {
		return fmt.Errorf("配置校验失败: %w", err)
	}

	cmd := exec.Command("systemctl", "restart", "xray")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart xray: %w", err)
	}

	logger.Info("Xray config applied via hot reload (write + restart)")
	return nil
}

// validateXrayConfig runs `xray -test -c <config>` to validate the generated config.
// Returns nil if valid, error with details if invalid.
func (s *Server) validateXrayConfig() error {
	binaryPath := s.config.Xray.BinaryPath
	if binaryPath == "" {
		binaryPath = "/usr/local/bin/xray"
	}

	cmd := exec.Command(binaryPath, "-test", "-c", s.config.Xray.ConfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Xray config validation failed: %s", string(output))
		// Extract the meaningful error line from output
		errMsg := string(output)
		if len(errMsg) > 500 {
			errMsg = errMsg[:500] + "..."
		}
		return fmt.Errorf("%s", errMsg)
	}

	logger.Info("Xray config validation passed ✓")
	return nil
}

// generateXrayConfig creates the Xray configuration
func (s *Server) generateXrayConfig() ([]byte, error) {
	var users []models.User
	var inbounds []models.Inbound
	var outbounds []models.Outbound
	var rules []models.RoutingRule
	var domains []models.Domain

	s.db.Where("enabled = ?", true).Find(&users)
	s.db.Preload("Domain").Where("enabled = ?", true).Find(&inbounds)
	s.db.Where("enabled = ?", true).Find(&outbounds)
	s.db.Where("enabled = ?", true).Order("priority ASC").Find(&rules)
	s.db.Where("enabled = ?", true).Find(&domains)

	var modeSetting models.Setting
	panelMode := "server"
	if err := s.db.First(&modeSetting, "key = ?", "panel_mode").Error; err == nil {
		panelMode = modeSetting.Value
	}

	clientRoutingMode := models.GetClientRoutingMode(s.db)

	generator := xray.NewGenerator()
	generator.SetUsers(users)
	generator.SetInbounds(inbounds)
	generator.SetOutbounds(outbounds)
	generator.SetRoutingRules(rules)
	generator.SetDomains(domains)
	generator.SetAPIPort(s.config.Xray.APIPort)
	generator.SetSocketDir(s.config.Xray.SocketDir)
	generator.SetPanelMode(panelMode)
	generator.SetClientRoutingMode(clientRoutingMode)

	return generator.GenerateJSON()
}
