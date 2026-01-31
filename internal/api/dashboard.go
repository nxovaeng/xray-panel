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

	// Sum all user traffic
	var users []models.User
	s.db.Find(&users)
	for _, u := range users {
		data.TotalTrafficDown += u.TrafficUsed // Simplified: assuming all traffic is download
	}

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

	// 3. Generate Nginx configs
	var inbounds []models.Inbound
	s.db.Preload("Domain").Where("enabled = ?", true).Find(&inbounds)

	nginxGen := nginx.NewGenerator(s.config.Nginx.ConfigDir, s.config.Nginx.StreamDir)

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

// applyXrayConfigHot applies configuration changes via Xray API (hot reload)
// NOTE: Xray uses gRPC API, not HTTP. Full implementation requires:
// 1. Add google.golang.org/grpc dependency
// 2. Import Xray protobuf definitions from github.com/xtls/xray-core
// 3. Implement gRPC client calls for HandlerService
//
// For now, this function returns an error suggesting to use the restart method.
func (s *Server) applyXrayConfigHot() error {
	// Hot reload via gRPC API is not implemented yet.
	// The Xray API uses gRPC protocol (not HTTP REST), which requires:
	// - gRPC client library
	// - Xray protobuf definitions
	// - Connection to Xray's dokodemo-door API inbound
	//
	// Current workaround: Use the restart method which writes config to file
	// and restarts the Xray service via systemctl.

	return fmt.Errorf("热更新功能暂未实现，请使用重启方式应用配置")
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

	generator := xray.NewGenerator()
	generator.SetUsers(users)
	generator.SetInbounds(inbounds)
	generator.SetOutbounds(outbounds)
	generator.SetRoutingRules(rules)
	generator.SetDomains(domains)
	generator.SetAPIPort(s.config.Xray.APIPort)

	return generator.GenerateJSON()
}
