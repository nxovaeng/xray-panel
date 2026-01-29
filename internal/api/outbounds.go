package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"xray-panel/internal/models"
)

// CreateOutboundRequest represents the request to create an outbound
type CreateOutboundRequest struct {
	Tag            string              `json:"tag"`
	Type           models.OutboundType `json:"type" binding:"required"`
	Server         string              `json:"server"`
	Port           int                 `json:"port"`
	Username       string              `json:"username"`
	Password       string              `json:"password"`
	WGSecretKey    string              `json:"wg_secret_key"`
	WGPublicKey    string              `json:"wg_public_key"`
	WGReserved     string              `json:"wg_reserved"`
	WGLocalIPv4    string              `json:"wg_local_ipv4"`
	WGLocalIPv6    string              `json:"wg_local_ipv6"`
	WGMTU          int                 `json:"wg_mtu"`
	TrojanPassword string              `json:"trojan_password"`
	TrojanSNI      string              `json:"trojan_sni"`
	TrojanNetwork  string              `json:"trojan_network"`
	Priority       int                 `json:"priority"`
	Remark         string              `json:"remark"`
}

// handleListOutbounds returns all outbounds
func (s *Server) handleListOutbounds(c *gin.Context) {
	var outbounds []models.Outbound
	if err := s.db.Order("priority DESC, created_at DESC").Find(&outbounds).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to fetch outbounds")
		return
	}
	jsonOK(c, outbounds)
}

// handleCreateOutbound creates a new outbound
func (s *Server) handleCreateOutbound(c *gin.Context) {
	var req CreateOutboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	outbound := models.Outbound{
		Tag:            req.Tag,
		Type:           req.Type,
		Server:         req.Server,
		Port:           req.Port,
		Username:       req.Username,
		Password:       req.Password,
		WGSecretKey:    req.WGSecretKey,
		WGPublicKey:    req.WGPublicKey,
		WGReserved:     req.WGReserved,
		WGLocalIPv4:    req.WGLocalIPv4,
		WGLocalIPv6:    req.WGLocalIPv6,
		WGMTU:          req.WGMTU,
		TrojanPassword: req.TrojanPassword,
		TrojanSNI:      req.TrojanSNI,
		TrojanNetwork:  req.TrojanNetwork,
		Priority:       req.Priority,
		Remark:         req.Remark,
		Enabled:        true,
	}

	if err := s.db.Create(&outbound).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to create outbound")
		return
	}

	jsonCreated(c, outbound)
}

// handleGetOutbound returns a single outbound
func (s *Server) handleGetOutbound(c *gin.Context) {
	id := c.Param("id")

	var outbound models.Outbound
	if err := s.db.First(&outbound, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "Outbound not found")
		return
	}

	jsonOK(c, outbound)
}

// handleUpdateOutbound updates an outbound
func (s *Server) handleUpdateOutbound(c *gin.Context) {
	id := c.Param("id")

	var outbound models.Outbound
	if err := s.db.First(&outbound, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "Outbound not found")
		return
	}

	var req CreateOutboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	// Update fields
	outbound.Tag = req.Tag
	outbound.Type = req.Type
	outbound.Server = req.Server
	outbound.Port = req.Port
	outbound.Username = req.Username
	outbound.WGSecretKey = req.WGSecretKey
	outbound.WGPublicKey = req.WGPublicKey
	outbound.WGReserved = req.WGReserved
	outbound.WGLocalIPv4 = req.WGLocalIPv4
	outbound.WGLocalIPv6 = req.WGLocalIPv6
	outbound.WGMTU = req.WGMTU
	outbound.TrojanPassword = req.TrojanPassword
	outbound.TrojanSNI = req.TrojanSNI
	outbound.TrojanNetwork = req.TrojanNetwork
	outbound.Priority = req.Priority
	outbound.Remark = req.Remark

	// Only update password if provided
	if req.Password != "" {
		outbound.Password = req.Password
	}

	if err := s.db.Save(&outbound).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to update outbound")
		return
	}

	jsonOK(c, outbound)
}

// handleDeleteOutbound deletes an outbound
func (s *Server) handleDeleteOutbound(c *gin.Context) {
	id := c.Param("id")

	result := s.db.Delete(&models.Outbound{}, "id = ?", id)
	if result.Error != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to delete outbound")
		return
	}
	if result.RowsAffected == 0 {
		jsonError(c, http.StatusNotFound, "Outbound not found")
		return
	}

	jsonOK(c, gin.H{"deleted": true})
}
