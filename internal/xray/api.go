package xray

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient represents a client for Xray API
type APIClient struct {
	baseURL string
	client  *http.Client
}

// NewAPIClient creates a new Xray API client
func NewAPIClient(host string, port int) *APIClient {
	return &APIClient{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
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

// GetStats gets statistics for a user or inbound
func (c *APIClient) GetStats(name string, reset bool) (int64, error) {
	payload := map[string]interface{}{
		"name":  name,
		"reset": reset,
	}

	resp, err := c.requestWithResponse("POST", "/stats/query", payload)
	if err != nil {
		return 0, err
	}

	// Parse response
	var result struct {
		Stat struct {
			Value int64 `json:"value"`
		} `json:"stat"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, err
	}

	return result.Stat.Value, nil
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

// IsHealthy checks if Xray API is responding
func (c *APIClient) IsHealthy() bool {
	req, err := http.NewRequest("GET", c.baseURL+"/stats/query", nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest
}
