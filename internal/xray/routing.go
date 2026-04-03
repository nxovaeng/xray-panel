package xray

import (
	"sort"

	"xray-panel/internal/models"
)

// RoutingConfig represents Xray routing configuration
type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy"`
	DomainMatcher  string        `json:"domainMatcher,omitempty"`
	Rules          []RoutingRule `json:"rules"`
}

// RoutingRule represents a single routing rule
type RoutingRule struct {
	Type        string   `json:"type"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
	Protocol    []string `json:"protocol,omitempty"`
	InboundTag  []string `json:"inboundTag,omitempty"`
	OutboundTag string   `json:"outboundTag"`
}

// generateRouting generates the routing configuration
func (g *Generator) generateRouting() *RoutingConfig {
	routing := &RoutingConfig{
		DomainStrategy: "IPIfNonMatch",
		DomainMatcher:  "hybrid",
		Rules:          make([]RoutingRule, 0),
	}

	// Add API routing rule first
	routing.Rules = append(routing.Rules, RoutingRule{
		Type:        "field",
		InboundTag:  []string{"api"},
		OutboundTag: "api",
	})

	if g.panelMode == "client" {
		// Route upstream proxy DNS to the primary configured wireguard/trojan/socks proxy
		// This uses string literal to avoid depending heavily on user input formatting.
		proxyTag := "proxy"
		if len(g.outbounds) > 0 {
			proxyTag = g.outbounds[0].Tag
		}

		// Route intercepted 53 DNS to Xray internal DNS process
		routing.Rules = append(routing.Rules, RoutingRule{
			Type:        "field",
			InboundTag:  []string{"dns-in"},
			OutboundTag: "dns-out",
		})

		// Route DNS server IPs
		// Remote DNS (Cloudflare) -> Proxy
		routing.Rules = append(routing.Rules, RoutingRule{
			Type:        "field",
			IP:          []string{"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4"},
			OutboundTag: proxyTag,
		})

		// Local DNS (AliDNS) -> Direct
		routing.Rules = append(routing.Rules, RoutingRule{
			Type:        "field",
			IP:          []string{"223.5.5.5", "223.6.6.6"},
			OutboundTag: "direct",
		})

		// Route client inbounds through proxy
		routing.Rules = append(routing.Rules, RoutingRule{
			Type:        "field",
			InboundTag:  []string{"socks-in", "http-in"},
			OutboundTag: proxyTag,
		})
	}

	// Sort rules by priority
	sortedRules := make([]models.RoutingRule, len(g.rules))
	copy(sortedRules, g.rules)
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority < sortedRules[j].Priority
	})

	// Convert model rules to Xray rules
	for _, rule := range sortedRules {
		if !rule.Enabled {
			continue
		}

		xrayRule := RoutingRule{
			Type:        "field",
			OutboundTag: rule.OutboundTag,
		}

		switch rule.Type {
		case models.RuleTypeInbound:
			// Inbound-based routing
			if rule.InboundTag != "" {
				xrayRule.InboundTag = []string{rule.InboundTag}
			} else {
				continue // Skip if no inbound tag specified
			}

		case models.RuleTypeDomain:
			domains := splitCSV(rule.Domains)
			if len(domains) > 0 {
				xrayRule.Domain = domains
			} else {
				continue // Skip empty domain rules
			}

		case models.RuleTypeIP:
			ips := splitCSV(rule.IPs)
			if len(ips) > 0 {
				xrayRule.IP = ips
			} else {
				continue
			}

		case models.RuleTypeGeoSite:
			tags := splitCSV(rule.GeoSiteTags)
			if len(tags) > 0 {
				domains := make([]string, len(tags))
				for i, tag := range tags {
					domains[i] = "geosite:" + tag
				}
				xrayRule.Domain = domains
			} else {
				continue
			}

		case models.RuleTypeGeoIP:
			codes := splitCSV(rule.GeoIPCodes)
			if len(codes) > 0 {
				ips := make([]string, len(codes))
				for i, code := range codes {
					ips[i] = "geoip:" + code
				}
				xrayRule.IP = ips
			} else {
				continue
			}

		case models.RuleTypeProtocol:
			protocols := splitCSV(rule.Protocols)
			if len(protocols) > 0 {
				xrayRule.Protocol = protocols
			} else {
				continue
			}
		}

		routing.Rules = append(routing.Rules, xrayRule)
	}

	return routing
}
