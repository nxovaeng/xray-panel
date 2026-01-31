package api

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"

	"xray-panel/internal/models"
)

// CreateInboundRequest represents the request to create an inbound
type CreateInboundRequest struct {
	Tag           string           `json:"tag"`
	Protocol      models.Protocol  `json:"protocol"`
	Transport     models.Transport `json:"transport" binding:"required"`
	Port          int              `json:"port"` // 可选，为0时自动生成随机端口
	Listen        string           `json:"listen"`
	DomainID      string           `json:"domain_id"`
	Path          string           `json:"path"` // 可选，为空时自动生成随机路径
	ServiceName   string           `json:"service_name"`
	Host          string           `json:"host"`
	Mode          string           `json:"mode"`
	Remark        string           `json:"remark"`
	ConnectDomain string           `json:"connect_domain"` // 连接目标域名（CDN域名或父域名）
	RandomPort    bool             `json:"random_port"`    // 是否生成随机端口
	RandomPath    bool             `json:"random_path"`    // 是否生成随机路径
}

// generateRandomPort generates a random port between 10000 and 60000
func generateRandomPort() (int, error) {
	// 生成 10000-60000 之间的随机端口
	n, err := rand.Int(rand.Reader, big.NewInt(50000))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 10000, nil
}

// generateRandomPath generates a random path for xhttp/ws
func generateRandomPath() string {
	// 生成 8 字节的随机十六进制字符串
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "/" + hex.EncodeToString(bytes)
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

	// 处理随机端口
	port := req.Port
	if req.RandomPort || port == 0 {
		randomPort, err := generateRandomPort()
		if err != nil {
			jsonError(c, http.StatusInternalServerError, "Failed to generate random port")
			return
		}
		port = randomPort
	}

	// 处理随机路径（仅对 xhttp 和 ws 有效）
	path := req.Path
	if req.RandomPath || (path == "" && (req.Transport == models.TransportXHTTP || req.Transport == models.TransportWS)) {
		path = generateRandomPath()
	}

	inbound := models.Inbound{
		Tag:           req.Tag,
		Protocol:      req.Protocol,
		Transport:     req.Transport,
		Port:          port,
		Listen:        req.Listen,
		DomainID:      req.DomainID,
		Path:          path,
		ServiceName:   req.ServiceName,
		Host:          req.Host,
		Mode:          req.Mode,
		Remark:        req.Remark,
		ConnectDomain: req.ConnectDomain,
		Enabled:       true,
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

	// 处理随机端口
	port := req.Port
	if req.RandomPort {
		randomPort, err := generateRandomPort()
		if err != nil {
			jsonError(c, http.StatusInternalServerError, "Failed to generate random port")
			return
		}
		port = randomPort
	}

	// 处理随机路径（仅对 xhttp 和 ws 有效）
	path := req.Path
	if req.RandomPath {
		path = generateRandomPath()
	}

	// Update fields
	inbound.Tag = req.Tag
	inbound.Transport = req.Transport
	inbound.Port = port
	inbound.Listen = req.Listen
	inbound.DomainID = req.DomainID
	inbound.Path = path
	inbound.ServiceName = req.ServiceName
	inbound.Host = req.Host
	inbound.Mode = req.Mode
	inbound.Remark = req.Remark
	inbound.ConnectDomain = req.ConnectDomain

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
