package xray

import (
	"fmt"

	"xray-panel/internal/models"
)

// InboundConfig represents an Xray inbound configuration
type InboundConfig struct {
	Tag            string                 `json:"tag"`
	Listen         string                 `json:"listen,omitempty"`
	Port           interface{}            `json:"port"`
	Protocol       string                 `json:"protocol"`
	Settings       map[string]interface{} `json:"settings,omitempty"`
	StreamSettings *StreamSettings        `json:"streamSettings,omitempty"`
	Sniffing       *SniffingConfig        `json:"sniffing,omitempty"`
}

// StreamSettings represents stream settings
type StreamSettings struct {
	Network         string           `json:"network"`
	Security        string           `json:"security,omitempty"`
	TLSSettings     *TLSSettings     `json:"tlsSettings,omitempty"`
	RealitySettings *RealitySettings `json:"realitySettings,omitempty"`
	XHTTPSettings   *XHTTPSettings   `json:"xhttpSettings,omitempty"`
	GRPCSettings    *GRPCSettings    `json:"grpcSettings,omitempty"`
	TCPSettings     *TCPSettings     `json:"tcpSettings,omitempty"`
	WSSettings      *WSSettings      `json:"wsSettings,omitempty"`
}

// TLSSettings represents TLS configuration
type TLSSettings struct {
	ServerName   string    `json:"serverName,omitempty"`
	ALPN         []string  `json:"alpn,omitempty"`
	Certificates []TLSCert `json:"certificates,omitempty"`
}

// TLSCert represents a TLS certificate
type TLSCert struct {
	CertificateFile string `json:"certificateFile,omitempty"`
	KeyFile         string `json:"keyFile,omitempty"`
}

// RealitySettings represents Reality protocol settings
type RealitySettings struct {
	Show        bool     `json:"show,omitempty"`
	Dest        string   `json:"dest"`
	Xver        int      `json:"xver,omitempty"`
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIds    []string `json:"shortIds"`
}

// XHTTPSettings represents XHTTP transport settings
type XHTTPSettings struct {
	Path  string            `json:"path,omitempty"`
	Host  string            `json:"host,omitempty"`
	Mode  string            `json:"mode,omitempty"`
	Extra map[string]string `json:"extra,omitempty"`
}

// GRPCSettings represents gRPC transport settings
type GRPCSettings struct {
	ServiceName        string `json:"serviceName"`
	MultiMode          bool   `json:"multiMode,omitempty"`
	IdleTimeout        int    `json:"idle_timeout,omitempty"`
	HealthCheckTimeout int    `json:"health_check_timeout,omitempty"`
}

// TCPSettings represents TCP transport settings
type TCPSettings struct {
	AcceptProxyProtocol bool `json:"acceptProxyProtocol,omitempty"`
}

// WSSettings represents WebSocket transport settings
type WSSettings struct {
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// SniffingConfig represents sniffing configuration
type SniffingConfig struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride,omitempty"`
	RouteOnly    bool     `json:"routeOnly,omitempty"`
}

// generateInbound generates a single inbound configuration
func (g *Generator) generateInbound(inbound models.Inbound) (*InboundConfig, error) {
	config := &InboundConfig{
		Tag:      inbound.Tag,
		Listen:   inbound.Listen,
		Port:     inbound.Port,
		Protocol: string(inbound.Protocol),
		Sniffing: &SniffingConfig{
			Enabled:      true,
			DestOverride: []string{"http", "tls", "quic", "fakedns"},
			RouteOnly:    true,
		},
	}

	// Generate protocol-specific settings
	switch inbound.Protocol {
	case models.ProtocolVLESS:
		config.Settings = g.generateVLESSSettings()
	case models.ProtocolTrojan:
		config.Settings = g.generateTrojanSettings()
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", inbound.Protocol)
	}

	// Generate stream settings
	streamSettings := &StreamSettings{}

	switch inbound.Transport {
	case models.TransportXHTTP:
		streamSettings.Network = "xhttp"
		streamSettings.XHTTPSettings = &XHTTPSettings{
			Path: inbound.Path,
			Host: inbound.Host,
			Mode: inbound.Mode,
		}
		if inbound.Mode == "" {
			streamSettings.XHTTPSettings.Mode = "auto"
		}

	case models.TransportGRPC:
		streamSettings.Network = "grpc"
		streamSettings.GRPCSettings = &GRPCSettings{
			ServiceName: inbound.ServiceName,
			MultiMode:   true,
		}

	case models.TransportWS:
		streamSettings.Network = "ws"
		streamSettings.WSSettings = &WSSettings{
			Path: inbound.Path,
		}
		if inbound.Host != "" {
			streamSettings.WSSettings.Headers = map[string]string{
				"Host": inbound.Host,
			}
		}

	default:
		return nil, fmt.Errorf("unsupported transport: %s", inbound.Transport)
	}

	// Security 始终为 none，由 Nginx 处理 TLS
	streamSettings.Security = "none"

	config.StreamSettings = streamSettings
	return config, nil
}

// generateVLESSSettings generates VLESS protocol settings
func (g *Generator) generateVLESSSettings() map[string]interface{} {
	clients := make([]map[string]interface{}, 0)
	for _, user := range g.getActiveUsers() {
		client := map[string]interface{}{
			"id":    user.UUID,
			"email": user.Email,
			"level": 0,
		}
		clients = append(clients, client)
	}

	return map[string]interface{}{
		"clients":    clients,
		"decryption": "none",
	}
}

// generateTrojanSettings generates Trojan protocol settings
func (g *Generator) generateTrojanSettings() map[string]interface{} {
	clients := make([]map[string]interface{}, 0)
	for _, user := range g.getActiveUsers() {
		// Trojan 使用 password 而不是 UUID
		client := map[string]interface{}{
			"password": user.UUID, // 使用 UUID 作为 Trojan 密码
			"email":    user.Email,
			"level":    0,
		}
		clients = append(clients, client)
	}

	return map[string]interface{}{
		"clients": clients,
	}
}
