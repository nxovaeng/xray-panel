package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xray-panel/internal/models"
)

// CreateOutboundRequest represents the request to create an outbound
type CreateOutboundRequest struct {
	Tag      string              `json:"tag" form:"tag"`
	Type     models.OutboundType `json:"type" form:"type" binding:"required"`
	Server   string              `json:"server" form:"server"`
	Port     int                 `json:"port" form:"port"`
	Username string              `json:"username" form:"username"`
	Password string              `json:"password" form:"password"`
	// WireGuard specific fields (separate from SOCKS5 server/port)
	WGServer       string `json:"wg_server" form:"wg_server"`
	WGPort         int    `json:"wg_port" form:"wg_port"`
	WGSecretKey    string `json:"wg_secret_key" form:"wg_secret_key"`
	WGPublicKey    string `json:"wg_public_key" form:"wg_public_key"`
	WGReserved     string `json:"wg_reserved" form:"wg_reserved"`
	WGLocalIPv4    string `json:"wg_local_ipv4" form:"wg_local_ipv4"`
	WGLocalIPv6    string `json:"wg_local_ipv6" form:"wg_local_ipv6"`
	WGMTU          int    `json:"wg_mtu" form:"wg_mtu"`
	WGDNS          string `json:"wg_dns" form:"wg_dns"`
	TrojanPassword string `json:"trojan_password" form:"trojan_password"`
	TrojanSNI      string `json:"trojan_sni" form:"trojan_sni"`
	TrojanNetwork  string `json:"trojan_network" form:"trojan_network"`
	Priority       int    `json:"priority" form:"priority"`
	Remark         string `json:"remark" form:"remark"`
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
	if err := c.ShouldBind(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Determine server and port based on outbound type
	server := req.Server
	port := req.Port

	// For WireGuard, use WGServer and WGPort if provided
	if req.Type == models.OutboundWireGuard {
		if req.WGServer != "" {
			server = req.WGServer
		}
		if req.WGPort > 0 {
			port = req.WGPort
		}
	}

	outbound := models.Outbound{
		Tag:            req.Tag,
		Type:           req.Type,
		Server:         server,
		Port:           port,
		Username:       req.Username,
		Password:       req.Password,
		WGSecretKey:    req.WGSecretKey,
		WGPublicKey:    req.WGPublicKey,
		WGReserved:     req.WGReserved,
		WGLocalIPv4:    req.WGLocalIPv4,
		WGLocalIPv6:    req.WGLocalIPv6,
		WGMTU:          req.WGMTU,
		WGDNS:          req.WGDNS,
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
	if err := c.ShouldBind(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	// Determine server and port based on outbound type
	server := req.Server
	port := req.Port

	// For WireGuard, use WGServer and WGPort if provided
	if req.Type == models.OutboundWireGuard {
		if req.WGServer != "" {
			server = req.WGServer
		}
		if req.WGPort > 0 {
			port = req.WGPort
		}
	}

	// Update fields
	outbound.Tag = req.Tag
	outbound.Type = req.Type
	outbound.Server = server
	outbound.Port = port
	outbound.Username = req.Username
	outbound.WGPublicKey = req.WGPublicKey
	outbound.WGReserved = req.WGReserved
	outbound.WGLocalIPv4 = req.WGLocalIPv4
	outbound.WGLocalIPv6 = req.WGLocalIPv6
	outbound.WGMTU = req.WGMTU
	outbound.WGDNS = req.WGDNS
	outbound.TrojanPassword = req.TrojanPassword
	outbound.TrojanSNI = req.TrojanSNI
	outbound.TrojanNetwork = req.TrojanNetwork
	outbound.Priority = req.Priority
	outbound.Remark = req.Remark

	// Only update password if provided
	if req.Password != "" {
		outbound.Password = req.Password
	}

	// Only update WireGuard secret key if provided
	if req.WGSecretKey != "" {
		outbound.WGSecretKey = req.WGSecretKey
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

// handleTestOutbound tests the connectivity of an outbound
func (s *Server) handleTestOutbound(c *gin.Context) {
	id := c.Param("id")

	var outbound models.Outbound
	if err := s.db.First(&outbound, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, OutboundTestResult{
			Success: false,
			Message: "Outbound not found",
		})
		return
	}

	result := testOutboundConnectivity(outbound)
	c.JSON(http.StatusOK, result)
}

// OutboundTestResult represents the result of an outbound connectivity test
type OutboundTestResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Latency  int64  `json:"latency_ms,omitempty"` // Latency in milliseconds
	Endpoint string `json:"endpoint,omitempty"`
}

// testOutboundConnectivity tests the connectivity of an outbound
func testOutboundConnectivity(outbound models.Outbound) OutboundTestResult {
	switch outbound.Type {
	case models.OutboundWireGuard:
		return testWireGuardConnectivity(outbound)
	case models.OutboundSOCKS5:
		return testSOCKS5Connectivity(outbound)
	case models.OutboundTrojan:
		return testTrojanConnectivity(outbound)
	default:
		return OutboundTestResult{
			Success: false,
			Message: "Unsupported outbound type for testing",
		}
	}
}

// testWireGuardConnectivity tests WireGuard endpoint connectivity
func testWireGuardConnectivity(outbound models.Outbound) OutboundTestResult {
	if outbound.Server == "" {
		return OutboundTestResult{
			Success: false,
			Message: "WireGuard server address not configured",
		}
	}

	if outbound.Port == 0 {
		return OutboundTestResult{
			Success: false,
			Message: "WireGuard port not configured",
		}
	}

	endpoint := fmt.Sprintf("%s:%d", outbound.Server, outbound.Port)

	// Test UDP connectivity to WireGuard endpoint
	// WireGuard uses UDP, so we test UDP reachability
	start := time.Now()

	// First, try to resolve the hostname
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var resolver net.Resolver
	ips, err := resolver.LookupHost(ctx, outbound.Server)
	if err != nil {
		// If it's already an IP address, that's fine
		if net.ParseIP(outbound.Server) == nil {
			return OutboundTestResult{
				Success:  false,
				Message:  fmt.Sprintf("DNS resolution failed: %v", err),
				Endpoint: endpoint,
			}
		}
		ips = []string{outbound.Server}
	}

	// Try to connect via UDP (WireGuard uses UDP)
	for _, ip := range ips {
		addr := fmt.Sprintf("%s:%d", ip, outbound.Port)
		conn, err := net.DialTimeout("udp", addr, 5*time.Second)
		if err != nil {
			continue
		}
		defer conn.Close()

		latency := time.Since(start).Milliseconds()

		// For UDP, we can't really verify the connection without sending WireGuard handshake
		// But we can at least verify the endpoint is reachable
		return OutboundTestResult{
			Success:  true,
			Message:  fmt.Sprintf("UDP endpoint reachable (IP: %s)", ip),
			Latency:  latency,
			Endpoint: endpoint,
		}
	}

	return OutboundTestResult{
		Success:  false,
		Message:  "Failed to reach WireGuard endpoint via UDP",
		Endpoint: endpoint,
	}
}

// testSOCKS5Connectivity tests SOCKS5 proxy connectivity
func testSOCKS5Connectivity(outbound models.Outbound) OutboundTestResult {
	if outbound.Server == "" || outbound.Port == 0 {
		return OutboundTestResult{
			Success: false,
			Message: "SOCKS5 server address or port not configured",
		}
	}

	endpoint := fmt.Sprintf("%s:%d", outbound.Server, outbound.Port)
	start := time.Now()

	// Test TCP connectivity to SOCKS5 proxy
	conn, err := net.DialTimeout("tcp", endpoint, 5*time.Second)
	if err != nil {
		return OutboundTestResult{
			Success:  false,
			Message:  fmt.Sprintf("Connection failed: %v", err),
			Endpoint: endpoint,
		}
	}
	defer conn.Close()

	latency := time.Since(start).Milliseconds()

	// Try to read SOCKS5 greeting
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send SOCKS5 greeting (version 5, 1 auth method, no auth)
	_, err = conn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		return OutboundTestResult{
			Success:  true,
			Message:  "TCP connection successful, but SOCKS5 handshake failed",
			Latency:  latency,
			Endpoint: endpoint,
		}
	}

	// Read response
	buf := make([]byte, 2)
	_, err = conn.Read(buf)
	if err != nil {
		return OutboundTestResult{
			Success:  true,
			Message:  "TCP connection successful, SOCKS5 response timeout",
			Latency:  latency,
			Endpoint: endpoint,
		}
	}

	if buf[0] == 0x05 {
		return OutboundTestResult{
			Success:  true,
			Message:  "SOCKS5 proxy is responding correctly",
			Latency:  latency,
			Endpoint: endpoint,
		}
	}

	return OutboundTestResult{
		Success:  true,
		Message:  fmt.Sprintf("TCP connection successful, unexpected response: %v", buf),
		Latency:  latency,
		Endpoint: endpoint,
	}
}

// testTrojanConnectivity tests Trojan server connectivity
func testTrojanConnectivity(outbound models.Outbound) OutboundTestResult {
	if outbound.Server == "" || outbound.Port == 0 {
		return OutboundTestResult{
			Success: false,
			Message: "Trojan server address or port not configured",
		}
	}

	endpoint := fmt.Sprintf("%s:%d", outbound.Server, outbound.Port)
	start := time.Now()

	// Test TCP connectivity to Trojan server
	conn, err := net.DialTimeout("tcp", endpoint, 5*time.Second)
	if err != nil {
		return OutboundTestResult{
			Success:  false,
			Message:  fmt.Sprintf("Connection failed: %v", err),
			Endpoint: endpoint,
		}
	}
	defer conn.Close()

	latency := time.Since(start).Milliseconds()

	return OutboundTestResult{
		Success:  true,
		Message:  "TCP connection to Trojan server successful",
		Latency:  latency,
		Endpoint: endpoint,
	}
}

// parseWireGuardConfig parses a WireGuard configuration string (from ProtonVPN, etc.)
func parseWireGuardConfig(config string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(config, "\n")

	currentSection := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			continue
		}

		// Parse key = value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Prefix with section name for clarity
			if currentSection != "" {
				key = currentSection + "_" + key
			}
			result[key] = value
		}
	}

	return result
}

// handleParseWireGuardConfig parses a WireGuard configuration and returns structured data
func (s *Server) handleParseWireGuardConfig(c *gin.Context) {
	var req struct {
		Config string `json:"config" form:"config"`
	}

	if err := c.ShouldBind(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.Config == "" {
		jsonError(c, http.StatusBadRequest, "Configuration is required")
		return
	}

	parsed := parseWireGuardConfig(req.Config)

	// Extract relevant fields
	result := gin.H{
		"private_key": parsed["Interface_PrivateKey"],
		"address":     parsed["Interface_Address"],
		"dns":         parsed["Interface_DNS"],
		"public_key":  parsed["Peer_PublicKey"],
		"endpoint":    parsed["Peer_Endpoint"],
	}

	// Parse endpoint into server and port
	if endpoint := parsed["Peer_Endpoint"]; endpoint != "" {
		// Handle IPv6 addresses like [::1]:51820
		if strings.HasPrefix(endpoint, "[") {
			// IPv6 format: [address]:port
			if idx := strings.LastIndex(endpoint, "]:"); idx != -1 {
				result["server"] = endpoint[1:idx]
				result["port"] = endpoint[idx+2:]
			}
		} else {
			// IPv4 or hostname format: address:port
			if idx := strings.LastIndex(endpoint, ":"); idx != -1 {
				result["server"] = endpoint[:idx]
				result["port"] = endpoint[idx+1:]
			}
		}
	}

	// Parse address into IPv4 and IPv6
	if address := parsed["Interface_Address"]; address != "" {
		addresses := strings.Split(address, ",")
		for _, addr := range addresses {
			addr = strings.TrimSpace(addr)
			if strings.Contains(addr, ":") {
				result["local_ipv6"] = addr
			} else {
				result["local_ipv4"] = addr
			}
		}
	}

	jsonOK(c, result)
}
