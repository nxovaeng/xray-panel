package xray

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"xray-panel/internal/models"
)

// OutboundConfig represents an Xray outbound configuration
type OutboundConfig struct {
	Tag            string                 `json:"tag"`
	Protocol       string                 `json:"protocol"`
	Settings       map[string]interface{} `json:"settings,omitempty"`
	StreamSettings *StreamSettings        `json:"streamSettings,omitempty"`
}

// generateOutbounds generates all outbound configurations
func (g *Generator) generateOutbounds() []OutboundConfig {
	outbounds := make([]OutboundConfig, 0)

	// Always add direct outbound first
	outbounds = append(outbounds, OutboundConfig{
		Tag:      "direct",
		Protocol: "freedom",
		Settings: map[string]interface{}{
			"domainStrategy": "UseIPv4",
		},
	})

	// Add block/blackhole outbound
	outbounds = append(outbounds, OutboundConfig{
		Tag:      "block",
		Protocol: "blackhole",
		Settings: map[string]interface{}{
			"response": map[string]interface{}{
				"type": "http",
			},
		},
	})

	// Add configured outbounds
	for _, outbound := range g.outbounds {
		if !outbound.Enabled {
			continue
		}

		switch outbound.Type {
		case models.OutboundWireGuard:
			outbounds = append(outbounds, g.generateWireGuardOutbound(outbound))
		case models.OutboundSOCKS5:
			outbounds = append(outbounds, g.generateSOCKS5Outbound(outbound))
		case models.OutboundTrojan:
			outbounds = append(outbounds, g.generateTrojanOutbound(outbound))
		case models.OutboundVLESS:
			outbounds = append(outbounds, g.generateVLESSOutbound(outbound))
		case models.OutboundVMess:
			outbounds = append(outbounds, g.generateVMessOutbound(outbound))
		}
	}

	return outbounds
}

// generateWireGuardOutbound generates a WireGuard outbound (WARP, Proton VPN, etc.)
func (g *Generator) generateWireGuardOutbound(outbound models.Outbound) OutboundConfig {
	// Parse reserved bytes from JSON array format [0,0,0]
	reserved := []int{0, 0, 0}
	if outbound.WGReserved != "" {
		// Try to parse as JSON array
		var parsedReserved []int
		if err := json.Unmarshal([]byte(outbound.WGReserved), &parsedReserved); err == nil && len(parsedReserved) == 3 {
			reserved = parsedReserved
		} else {
			// Try to parse as comma-separated values: "0,0,0"
			parts := strings.Split(strings.Trim(outbound.WGReserved, "[] "), ",")
			if len(parts) == 3 {
				for i, part := range parts {
					if val, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
						reserved[i] = val
					}
				}
			}
		}
	}

	// Build endpoint from Server and Port
	endpoint := outbound.Server
	if outbound.Port > 0 {
		endpoint = fmt.Sprintf("%s:%d", outbound.Server, outbound.Port)
	}

	peers := []map[string]interface{}{
		{
			"publicKey": outbound.WGPublicKey,
			"endpoint":  endpoint,
		},
	}

	addresses := []string{}
	if outbound.WGLocalIPv4 != "" {
		addresses = append(addresses, outbound.WGLocalIPv4)
	}
	if outbound.WGLocalIPv6 != "" {
		addresses = append(addresses, outbound.WGLocalIPv6)
	}

	mtu := outbound.WGMTU
	if mtu == 0 {
		mtu = 1420 // Default MTU
	}

	return OutboundConfig{
		Tag:      outbound.Tag,
		Protocol: "wireguard",
		Settings: map[string]interface{}{
			"secretKey": outbound.WGSecretKey,
			"address":   addresses,
			"peers":     peers,
			"reserved":  reserved,
			"mtu":       mtu,
		},
	}
}

// generateSOCKS5Outbound generates a SOCKS5 outbound
func (g *Generator) generateSOCKS5Outbound(outbound models.Outbound) OutboundConfig {
	servers := []map[string]interface{}{
		{
			"address": outbound.Server,
			"port":    outbound.Port,
		},
	}

	// Add auth if configured
	if outbound.Username != "" {
		servers[0]["users"] = []map[string]interface{}{
			{
				"user": outbound.Username,
				"pass": outbound.Password,
			},
		}
	}

	return OutboundConfig{
		Tag:      outbound.Tag,
		Protocol: "socks",
		Settings: map[string]interface{}{
			"servers": servers,
		},
	}
}

// generateTrojanOutbound generates a Trojan outbound
func (g *Generator) generateTrojanOutbound(outbound models.Outbound) OutboundConfig {
	servers := []map[string]interface{}{
		{
			"address":  outbound.Server,
			"port":     outbound.Port,
			"password": outbound.TrojanPassword,
		},
	}

	config := OutboundConfig{
		Tag:      outbound.Tag,
		Protocol: "trojan",
		Settings: map[string]interface{}{
			"servers": servers,
		},
	}

	// Add stream settings for TLS and transport
	config.StreamSettings = g.generateOutboundStreamSettings(outbound)

	return config
}

// generateVLESSOutbound generates a VLESS outbound
func (g *Generator) generateVLESSOutbound(outbound models.Outbound) OutboundConfig {
	vnext := []map[string]interface{}{
		{
			"address": outbound.Server,
			"port":    outbound.Port,
			"users": []map[string]interface{}{
				{
					"id":         outbound.UUID,
					"encryption": "none",
					"flow":       outbound.Flow,
				},
			},
		},
	}

	config := OutboundConfig{
		Tag:      outbound.Tag,
		Protocol: "vless",
		Settings: map[string]interface{}{
			"vnext": vnext,
		},
	}

	config.StreamSettings = g.generateOutboundStreamSettings(outbound)
	return config
}

// generateVMessOutbound generates a VMess outbound
func (g *Generator) generateVMessOutbound(outbound models.Outbound) OutboundConfig {
	vnext := []map[string]interface{}{
		{
			"address": outbound.Server,
			"port":    outbound.Port,
			"users": []map[string]interface{}{
				{
					"id":       outbound.UUID,
					"security": outbound.Security,
				},
			},
		},
	}

	config := OutboundConfig{
		Tag:      outbound.Tag,
		Protocol: "vmess",
		Settings: map[string]interface{}{
			"vnext": vnext,
		},
	}

	config.StreamSettings = g.generateOutboundStreamSettings(outbound)
	return config
}

// generateOutboundStreamSettings generates stream settings for outbounds
func (g *Generator) generateOutboundStreamSettings(outbound models.Outbound) *StreamSettings {
	// If no transport or security is configured, return nil
	if outbound.Network == "" && !outbound.TLS && !outbound.Reality && outbound.TrojanNetwork == "" && outbound.TrojanSNI == "" {
		return nil
	}

	streamSettings := &StreamSettings{
		Network: "tcp",
	}

	// Determine network
	network := outbound.Network
	if network == "" && outbound.Type == models.OutboundTrojan {
		network = outbound.TrojanNetwork
	}
	if network != "" {
		streamSettings.Network = network
	}

	// Transport settings
	switch streamSettings.Network {
	case "ws":
		streamSettings.WSSettings = &WSSettings{
			Path: outbound.Path,
		}
		if outbound.RequestHost != "" {
			streamSettings.WSSettings.Headers = map[string]string{
				"Host": outbound.RequestHost,
			}
		}
	case "grpc":
		streamSettings.GRPCSettings = &GRPCSettings{
			ServiceName: outbound.ServiceName,
			MultiMode:   true,
		}
	case "xhttp":
		streamSettings.XHTTPSettings = &XHTTPSettings{
			Path: outbound.Path,
			Host: outbound.RequestHost,
			Mode: "auto",
		}
	}

	// Security settings
	if outbound.Reality {
		streamSettings.Security = "reality"
		streamSettings.RealitySettings = &RealitySettings{
			Show:        false,
			Fingerprint: "chrome",
			ServerName:  outbound.RealitySNI,
			PublicKey:   outbound.RealityPubKey,
			ShortId:     outbound.RealityShortID,
			SpiderX:     "/",
		}
	} else if outbound.TLS || outbound.Type == models.OutboundTrojan {
		streamSettings.Security = "tls"
		sni := outbound.TLSServerName
		if sni == "" && outbound.Type == models.OutboundTrojan {
			sni = outbound.TrojanSNI
		}
		streamSettings.TLSSettings = &TLSSettings{
			ServerName: sni,
		}
		if outbound.TLSALPN != "" {
			streamSettings.TLSSettings.ALPN = strings.Split(outbound.TLSALPN, ",")
		}
	}

	return streamSettings
}
