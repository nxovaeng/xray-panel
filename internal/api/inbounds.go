package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"xray-panel/internal/models"
)

// CreateInboundRequest represents the request to create an inbound
type CreateInboundRequest struct {
	Tag         string           `json:"tag"`
	Protocol    models.Protocol  `json:"protocol"`
	Transport   models.Transport `json:"transport" binding:"required"`
	Port        int              `json:"port" binding:"required"`
	Listen      string           `json:"listen"`
	DomainID    string           `json:"domain_id"`
	Path        string           `json:"path"`
	ServiceName string           `json:"service_name"`
	Host        string           `json:"host"`
	Mode        string           `json:"mode"`
	Remark      string           `json:"remark"`
}

// handleListInbounds returns all inbounds
func (s *Server) handleListInbounds(c *gin.Context) {
	var inbounds []models.Inbound
	if err := s.db.Preload("Domain").Order("created_at DESC").Find(&inbounds).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to fetch inbounds")
		return
	}
	jsonOK(c, inbounds)
}

// handleCreateInbound creates a new inbound
func (s *Server) handleCreateInbound(c *gin.Context) {
	var req CreateInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	inbound := models.Inbound{
		Tag:         req.Tag,
		Protocol:    req.Protocol,
		Transport:   req.Transport,
		Port:        req.Port,
		Listen:      req.Listen,
		DomainID:    req.DomainID,
		Path:        req.Path,
		ServiceName: req.ServiceName,
		Host:        req.Host,
		Mode:        req.Mode,
		Remark:      req.Remark,
		Enabled:     true,
	}

	// Set defaults
	if inbound.Protocol == "" {
		inbound.Protocol = models.ProtocolVLESS
	}
	if inbound.Listen == "" {
		inbound.Listen = "127.0.0.1"
	}

	if err := s.db.Create(&inbound).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to create inbound")
		return
	}

	jsonCreated(c, inbound)
}

// handleGetInbound returns a single inbound
func (s *Server) handleGetInbound(c *gin.Context) {
	id := c.Param("id")

	var inbound models.Inbound
	if err := s.db.Preload("Domain").First(&inbound, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "Inbound not found")
		return
	}

	jsonOK(c, inbound)
}

// handleUpdateInbound updates an inbound
func (s *Server) handleUpdateInbound(c *gin.Context) {
	id := c.Param("id")

	var inbound models.Inbound
	if err := s.db.First(&inbound, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "Inbound not found")
		return
	}

	var req CreateInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	// Update fields
	inbound.Tag = req.Tag
	inbound.Transport = req.Transport
	inbound.Port = req.Port
	inbound.Listen = req.Listen
	inbound.DomainID = req.DomainID
	inbound.Path = req.Path
	inbound.ServiceName = req.ServiceName
	inbound.Host = req.Host
	inbound.Mode = req.Mode
	inbound.Remark = req.Remark

	if err := s.db.Save(&inbound).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to update inbound")
		return
	}

	jsonOK(c, inbound)
}

// handleDeleteInbound deletes an inbound
func (s *Server) handleDeleteInbound(c *gin.Context) {
	id := c.Param("id")

	result := s.db.Delete(&models.Inbound{}, "id = ?", id)
	if result.Error != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to delete inbound")
		return
	}
	if result.RowsAffected == 0 {
		jsonError(c, http.StatusNotFound, "Inbound not found")
		return
	}

	jsonOK(c, gin.H{"deleted": true})
}
