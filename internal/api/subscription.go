package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"xray-panel/internal/models"
)

// handleSubscription generates aggregated subscription links for a user
// Supports multiple inbounds in a single subscription (KEY FEATURE)
func (s *Server) handleSubscription(c *gin.Context) {
	path := c.Param("path")
	format := c.Param("format")
	if format == "" {
		format = "base64" // default format
	}

	// Find user by subscription path
	var user models.User
	if err := s.db.Where("sub_path = ?", path).First(&user).Error; err != nil {
		c.String(http.StatusNotFound, "Subscription not found")
		return
	}

	// Check if user is active
	if !user.IsActive() {
		c.String(http.StatusForbidden, "Subscription expired or disabled")
		return
	}

	// Get all enabled inbounds with domains (AGGREGATED SUBSCRIPTION)
	var inbounds []models.Inbound
	if err := s.db.Preload("Domain").Where("enabled = ?", true).Find(&inbounds).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate subscription")
		return
	}

	// Generate links for all inbounds
	var links []string
	for _, inbound := range inbounds {
		link := generateVLESSLink(user, inbound)
		if link != "" {
			links = append(links, link)
		}
	}

	// Calculate user info
	uploadBytes := int64(0) // TODO: implement upload tracking
	downloadBytes := user.TrafficUsed
	totalBytes := user.TrafficLimit
	expireTime := int64(0)
	if !user.ExpiryDate.IsZero() {
		expireTime = user.ExpiryDate.Unix()
	}

	// Set subscription info header (for clients that support it)
	subInfo := fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d",
		uploadBytes, downloadBytes, totalBytes, expireTime)
	c.Header("Subscription-Userinfo", subInfo)
	c.Header("Profile-Update-Interval", "24") // Update every 24 hours
	c.Header("Profile-Title", fmt.Sprintf("Xray - %s", user.Name))

	// Generate safe filename
	filename := user.Name
	if filename == "" {
		filename = user.Email
	}
	if filename == "" {
		filename = "subscription"
	}
	// Remove unsafe characters from filename
	filename = strings.Map(func(r rune) rune {
		if r == ' ' {
			return '_'
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, filename)

	result := strings.Join(links, "\n")

	switch format {
	case "base64", "":
		// Base64 encoded (standard format)
		encoded := base64.StdEncoding.EncodeToString([]byte(result))
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.txt", filename))
		c.String(http.StatusOK, encoded)

	case "plain", "txt":
		// Plain text format
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.txt", filename))
		c.String(http.StatusOK, result)

	case "json":
		// JSON format with detailed info
		jsonOK(c, gin.H{
			"links": links,
			"count": len(links),
			"user": gin.H{
				"name":              user.Name,
				"email":             user.Email,
				"uuid":              user.UUID,
				"upload":            uploadBytes,
				"download":          downloadBytes,
				"total":             totalBytes,
				"expire":            expireTime,
				"traffic_used":      user.TrafficUsed,
				"traffic_limit":     user.TrafficLimit,
				"remaining_days":    user.RemainingDays(),
				"remaining_traffic": user.RemainingTraffic(),
				"is_active":         user.IsActive(),
			},
		})

	case "clash":
		// Clash format (YAML)
		clashConfig := generateClashConfig(user, inbounds)
		c.Header("Content-Type", "text/yaml; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.yaml", filename))
		c.String(http.StatusOK, clashConfig)

	default:
		c.String(http.StatusBadRequest, "Unknown format. Supported: base64, plain, json, clash")
	}
}

// generateVLESSLink generates a VLESS share link for a specific inbound
func generateVLESSLink(user models.User, inbound models.Inbound) string {
	if inbound.Domain == nil {
		return ""
	}

	// SNI 域名：使用 ActualDomain（如果存在），否则使用 Domain.Domain
	// ActualDomain 是为通配符证书生成的实际子域名（如 abc123.example.com）
	// Domain.Domain 可能是通配符（如 *.example.com）或普通域名
	sniDomain := inbound.Domain.Domain
	if inbound.ActualDomain != "" {
		sniDomain = inbound.ActualDomain
	}

	// 连接目标域名：如果设置了 ConnectDomain 则使用它，否则使用 sniDomain
	// ConnectDomain 用于 CDN 场景，客户端连接到 CDN 域名，但 SNI 使用反代子域名
	connectDomain := sniDomain
	if inbound.ConnectDomain != "" {
		connectDomain = inbound.ConnectDomain
	}

	// 端口始终为 443（Nginx 反向代理）
	// Xray 本身监听在 inbound.Port，但客户端连接到 Nginx 的 443 端口
	port := 443

	// Build query parameters
	params := url.Values{}
	params.Set("type", string(inbound.Transport))
	params.Set("encryption", "none")

	// Security 始终为 TLS（客户端到 Nginx）
	// Nginx 到 Xray 的连接是 none（内部通信）
	params.Set("security", "tls")
	params.Set("sni", sniDomain) // SNI 使用反代子域名
	params.Set("alpn", "h2,http/1.1")
	params.Set("fp", "chrome")

	// Transport settings
	switch inbound.Transport {
	case models.TransportXHTTP:
		params.Set("path", inbound.Path)
		if inbound.Host != "" {
			params.Set("host", inbound.Host)
		}
		if inbound.Mode != "" {
			params.Set("mode", inbound.Mode)
		}

	case models.TransportGRPC:
		params.Set("serviceName", inbound.ServiceName)
		params.Set("mode", "multi")

	case models.TransportWS:
		params.Set("path", inbound.Path)
		if inbound.Host != "" {
			params.Set("host", inbound.Host)
		}
	}

	// Build remark (node name)
	remark := inbound.Remark
	if remark == "" {
		remark = fmt.Sprintf("%s-%s-%s", sniDomain, inbound.Protocol, inbound.Transport)
	}

	// Build VLESS URL
	// 连接目标使用 connectDomain（可能是 CDN 域名或父域名）
	// SNI 使用 sniDomain（反代子域名）
	link := fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		user.UUID,
		connectDomain, // 连接目标域名
		port,
		params.Encode(),
		url.PathEscape(remark),
	)

	return link
}

// generateClashConfig generates Clash configuration
func generateClashConfig(user models.User, inbounds []models.Inbound) string {
	var sb strings.Builder

	sb.WriteString("# Clash Configuration\n")
	sb.WriteString(fmt.Sprintf("# User: %s\n", user.Name))
	sb.WriteString("# Generated by Xray Panel\n\n")

	sb.WriteString("proxies:\n")

	for _, inbound := range inbounds {
		if inbound.Domain == nil {
			continue
		}

		// SNI 域名：使用 ActualDomain（如果存在），否则使用 Domain.Domain
		sniDomain := inbound.Domain.Domain
		if inbound.ActualDomain != "" {
			sniDomain = inbound.ActualDomain
		}

		// 连接目标域名：如果设置了 ConnectDomain 则使用它，否则使用 sniDomain
		connectDomain := sniDomain
		if inbound.ConnectDomain != "" {
			connectDomain = inbound.ConnectDomain
		}

		name := inbound.Remark
		if name == "" {
			name = fmt.Sprintf("%s-%s", sniDomain, inbound.Transport)
		}

		sb.WriteString(fmt.Sprintf("  - name: \"%s\"\n", name))
		sb.WriteString("    type: vless\n")
		sb.WriteString(fmt.Sprintf("    server: %s\n", connectDomain)) // 连接目标域名

		// Use port 443 for Nginx reverse proxy connections
		port := 443
		sb.WriteString(fmt.Sprintf("    port: %d\n", port))

		sb.WriteString(fmt.Sprintf("    uuid: %s\n", user.UUID))
		sb.WriteString("    cipher: none\n")

		// Network type
		sb.WriteString(fmt.Sprintf("    network: %s\n", inbound.Transport))

		// TLS 始终启用（客户端到 Nginx）
		sb.WriteString("    tls: true\n")
		sb.WriteString(fmt.Sprintf("    servername: %s\n", sniDomain)) // SNI 使用反代子域名

		// Transport options
		switch inbound.Transport {
		case models.TransportWS:
			sb.WriteString("    ws-opts:\n")
			sb.WriteString(fmt.Sprintf("      path: %s\n", inbound.Path))
		case models.TransportGRPC:
			sb.WriteString("    grpc-opts:\n")
			sb.WriteString(fmt.Sprintf("      grpc-service-name: %s\n", inbound.ServiceName))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
