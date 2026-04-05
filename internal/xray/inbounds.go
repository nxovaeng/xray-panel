package xray

import (
	"fmt"
	"strings"

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
	Dest        string   `json:"dest,omitempty"`
	Xver        int      `json:"xver,omitempty"`
	ServerNames []string `json:"serverNames,omitempty"` // For inbound
	ServerName  string   `json:"serverName,omitempty"`  // For outbound
	PrivateKey  string   `json:"privateKey,omitempty"`  // For inbound
	PublicKey   string   `json:"publicKey,omitempty"`   // For outbound
	ShortIds    []string `json:"shortIds,omitempty"`    // For inbound
	ShortId     string   `json:"shortId,omitempty"`     // For outbound
	Fingerprint string   `json:"fingerprint,omitempty"`
	SpiderX     string   `json:"spiderX,omitempty"`
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
	// WireGuard is handled separately — no stream settings, no users
	if inbound.Protocol == models.ProtocolWireGuard {
		return g.generateWireGuardInbound(inbound)
	}

	// Determine listen address and port
	listen := inbound.Listen
	var port interface{} = inbound.Port
	if inbound.UseUDS {
		listen = inbound.SocketPath(g.socketDir)
		port = 0
	}

	config := &InboundConfig{
		Tag:      inbound.Tag,
		Listen:   listen,
		Port:     port,
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
			Mode: "auto", // 服务端固定 auto，接受客户端所有上传模式
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
			"flow":  "",
			"email": user.StatsKey(), // 用稳定的 stats key，不依赖可选的 Email 字段
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
		client := map[string]interface{}{
			"password": user.UUID,
			"email":    user.StatsKey(), // 用稳定的 stats key，不依赖可选的 Email 字段
			"level":    0,
		}
		clients = append(clients, client)
	}

	return map[string]interface{}{
		"clients": clients,
	}
}

// generateWireGuardInbound generates a WireGuard inbound configuration.
// WireGuard in Xray acts as a "freedom" tunnel — it receives traffic from
// another Xray node's WireGuard outbound and routes it locally.
// The peer is the remote xray-panel node that forwards traffic to this server.
func (g *Generator) generateWireGuardInbound(inbound models.Inbound) (*InboundConfig, error) {
	if inbound.WGSecretKey == "" {
		return nil, fmt.Errorf("wireguard inbound %q: secret key is required", inbound.Tag)
	}
	if inbound.WGPeerPubKey == "" {
		return nil, fmt.Errorf("wireguard inbound %q: peer public key is required", inbound.Tag)
	}

	mtu := inbound.WGMTU
	if mtu == 0 {
		mtu = 1420
	}

	localIP := inbound.WGLocalIP
	if localIP == "" {
		localIP = "10.0.0.1"
	}

	// Xray requires the interface address to be /32 for IPv4 and /128 for IPv6.
	// We strip any user-provided subnet and append the correct one.
	if idx := strings.Index(localIP, "/"); idx != -1 {
		localIP = localIP[:idx]
	}
	if strings.Contains(localIP, ":") {
		localIP = localIP + "/128"
	} else {
		localIP = localIP + "/32"
	}

	settings := map[string]interface{}{
		"secretKey": inbound.WGSecretKey,
		"address":   []string{localIP},
		"peers": []map[string]interface{}{
			{
				"publicKey":  inbound.WGPeerPubKey,
				"allowedIPs": []string{"0.0.0.0/0", "::/0"},
			},
		},
		"mtu": mtu,
	}

	listen := inbound.Listen
	// WireGuard must listen on all interfaces, unlike TCP proxies that sit behind Nginx
	if listen == "" || listen == "127.0.0.1" {
		listen = "0.0.0.0"
	}

	return &InboundConfig{
		Tag:      inbound.Tag,
		Listen:   listen,
		Port:     inbound.Port,
		Protocol: "wireguard",
		Settings: settings,
		// WireGuard 不需要 StreamSettings 和 Sniffing
	}, nil
}
