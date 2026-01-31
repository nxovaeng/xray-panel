package xray

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// APIClient represents a client for Xray API
type APIClient struct {
	baseURL    string
	client     *http.Client
	xrayBinary string
	apiPort    int
}

// NewAPIClient creates a new Xray API client
func NewAPIClient(host string, port int) *APIClient {
	return &APIClient{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		xrayBinary: "/usr/local/bin/xray",
		apiPort:    port,
	}
}

// NewAPIClientWithBinary creates a new Xray API client with custom binary path
func NewAPIClientWithBinary(host string, port int, binaryPath string) *APIClient {
	c := NewAPIClient(host, port)
	if binaryPath != "" {
		c.xrayBinary = binaryPath
	}
	return c
}

// AddUser adds a user to an inbound via API
func (c *APIClient) AddUser(inboundTag string, user map[string]interface{}) error {
	payload := map[string]interface{}{
		"tag":  inboundTag,
		"user": user,
	}

	return c.request("POST", "/handler/add/user", payload)
}

// RemoveUser removes a user from an inbound via API
func (c *APIClient) RemoveUser(inboundTag string, email string) error {
	payload := map[string]interface{}{
		"tag":   inboundTag,
		"email": email,
	}

	return c.request("POST", "/handler/remove/user", payload)
}

// AddInbound adds a new inbound via API
func (c *APIClient) AddInbound(inbound InboundConfig) error {
	return c.request("POST", "/handler/add/inbound", inbound)
}

// RemoveInbound removes an inbound via API
func (c *APIClient) RemoveInbound(tag string) error {
	payload := map[string]interface{}{
		"tag": tag,
	}

	return c.request("POST", "/handler/remove/inbound", payload)
}

// GetStats gets statistics for a user or inbound using xray api command
// name format: "user>>>email>>>traffic>>>downlink" or "user>>>email>>>traffic>>>uplink"
func (c *APIClient) GetStats(name string, reset bool) (int64, error) {
	args := []string{
		"api", "stats",
		"--server=127.0.0.1:" + strconv.Itoa(c.apiPort),
		"-name", name,
	}
	if reset {
		args = append(args, "-reset")
	}

	cmd := exec.Command(c.xrayBinary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the stat doesn't exist, xray returns an error, but that's okay
		return 0, nil
	}

	// Parse output - format is like:
	// stat: <
	//   name: "user>>>test@example.com>>>traffic>>>downlink"
	//   value: 12345
	// >
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "value:") {
			valueStr := strings.TrimPrefix(line, "value:")
			valueStr = strings.TrimSpace(valueStr)
			value, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return 0, nil
			}
			return value, nil
		}
	}

	return 0, nil
}

// RestartXray restarts Xray process (requires external script)
func (c *APIClient) RestartXray() error {
	// This would typically call a system command or script
	// For now, we'll just return nil as restart is handled externally
	return nil
}

// request makes an API request without expecting a response body
func (c *APIClient) request(method, path string, payload interface{}) error {
	_, err := c.requestWithResponse(method, path, payload)
	return err
}

// requestWithResponse makes an API request and returns the response body
func (c *APIClient) requestWithResponse(method, path string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// IsHealthy checks if Xray API is responding using xray api command
func (c *APIClient) IsHealthy() bool {
	// Try to query stats - if xray is running and API is enabled, this should work
	cmd := exec.Command(c.xrayBinary, "api", "stats", "--server=127.0.0.1:"+strconv.Itoa(c.apiPort))
	err := cmd.Run()
	// Even if there are no stats, the command should succeed if xray is running
	return err == nil
}
