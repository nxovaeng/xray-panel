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
	if outbound.TrojanSNI != "" || outbound.TrojanNetwork != "" {
		streamSettings := &StreamSettings{
			Network:  "tcp",
			Security: "tls",
		}

		if outbound.TrojanNetwork != "" {
			streamSettings.Network = outbound.TrojanNetwork
		}

		if outbound.TrojanSNI != "" {
			streamSettings.TLSSettings = &TLSSettings{
				ServerName: outbound.TrojanSNI,
			}
		}

		config.StreamSettings = streamSettings
	}

	return config
}
