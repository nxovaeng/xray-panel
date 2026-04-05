package api

import (
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
	CustomSNI     string           `json:"custom_sni"`     // 手动填写的 SNI（免 Nginx 配置）
	RandomPort    bool             `json:"random_port"`    // 是否生成随机端口
	RandomPath    bool             `json:"random_path"`    // 是否生成随机路径
	UseUDS        *bool            `json:"use_uds"`        // 使用 Unix Domain Socket（默认 true）
}

// handleGetInbound returns a single inbound as JSON (used by edit forms)
func (s *Server) handleGetInbound(c *gin.Context) {
	id := c.Param("id")

	var inbound models.Inbound
	if err := s.db.Preload("Domain").First(&inbound, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "Inbound not found")
		return
	}

	jsonOK(c, inbound)
}
