package api

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xray-panel/internal/models"
	"xray-panel/internal/xray"
)

// handleGetOutbound returns a single outbound as JSON (used by edit forms)
func (s *Server) handleGetOutbound(c *gin.Context) {
	id := c.Param("id")

	var outbound models.Outbound
	if err := s.db.First(&outbound, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "Outbound not found")
		return
	}

	jsonOK(c, outbound)
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

	result := s.testOutboundViaXray(outbound)
	c.JSON(http.StatusOK, result)
}

// OutboundTestResult represents the result of an outbound connectivity test
type OutboundTestResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Latency  int64  `json:"latency_ms,omitempty"` // Latency in milliseconds
	Endpoint string `json:"endpoint,omitempty"`
}

// testOutboundViaXray runs an ephemeral Xray proxy process and routes HTTP traffic through it.
// This executes a "True Dial Test" which verifies authentic reachability instead of false-positive Pings.
func (s *Server) testOutboundViaXray(outbound models.Outbound) OutboundTestResult {
	endpoint := outbound.Server
	if outbound.Port > 0 {
		endpoint = fmt.Sprintf("%s:%d", outbound.Server, outbound.Port)
	}

	// 1. Get a random free port for our local HTTP proxy
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return OutboundTestResult{Success: false, Message: "Failed to allocate local port: " + err.Error(), Endpoint: endpoint}
	}
	proxyPort := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// 2. Build a minimal Xray config JSON directly
	//    (The Generator only handles vless/trojan/wireguard inbounds,
	//     so we construct the test config by hand.)
	outbound.Enabled = true // Force enable for test

	gen := xray.NewGenerator()
	gen.SetOutbounds([]models.Outbound{outbound})

	// Use the generator only for the outbound section
	configJSON, err := gen.GenerateTestJSON(proxyPort)
	if err != nil {
		return OutboundTestResult{Success: false, Message: "Config build failed: " + err.Error(), Endpoint: endpoint}
	}

	// 3. Write temporary config file
	tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("xray_test_%s.json", outbound.ID))
	err = os.WriteFile(tempPath, configJSON, 0600)
	if err != nil {
		return OutboundTestResult{Success: false, Message: "Write temp file failed: " + err.Error(), Endpoint: endpoint}
	}
	defer os.Remove(tempPath)

	// 4. Start Xray background process
	binaryPath := s.config.Xray.BinaryPath
	if binaryPath == "" {
		binaryPath = "/usr/local/bin/xray"
	}
	cmd := exec.Command(binaryPath, "run", "-c", tempPath)
	err = cmd.Start()
	if err != nil {
		return OutboundTestResult{Success: false, Message: "Start Xray failed: " + err.Error(), Endpoint: endpoint}
	}

	// Ensure we kill the Xray child process when done
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Give Xray time to initialize
	time.Sleep(1000 * time.Millisecond)

	// 5. Test HTTP Proxy
	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", proxyPort))
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get("http://cp.cloudflare.com/generate_204")
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "deadline exceeded") || strings.Contains(errMsg, "timeout") {
			errMsg = "Connection timed out (Check server/firewall/password)"
		} else if strings.Contains(errMsg, "connection refused") {
			errMsg = "Connection refused by proxy server"
		}
		return OutboundTestResult{Success: false, Message: "Test failed: " + errMsg, Endpoint: endpoint, Latency: latency}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return OutboundTestResult{Success: true, Message: "Connected successfully (True Dial Test)", Latency: latency, Endpoint: endpoint}
	}

	return OutboundTestResult{Success: false, Message: fmt.Sprintf("Unexpected HTTP status: %d", resp.StatusCode), Latency: latency, Endpoint: endpoint}
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
