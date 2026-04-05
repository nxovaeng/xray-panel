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
	Port        string   `json:"port,omitempty"`
	Network     string   `json:"network,omitempty"`
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

		// DNS server IPs routing (Remote -> Proxy, Local -> Direct)
		routing.Rules = append(routing.Rules, RoutingRule{
			Type:        "field",
			IP:          []string{"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4"},
			OutboundTag: proxyTag,
		})
		routing.Rules = append(routing.Rules, RoutingRule{
			Type:        "field",
			IP:          []string{"223.5.5.5", "223.6.6.6"},
			OutboundTag: "direct",
		})

		// Local DNS (AliDNS) -> Direct
		routing.Rules = append(routing.Rules, RoutingRule{
			Type:        "field",
			IP:          []string{"223.5.5.5", "223.6.6.6"},
			OutboundTag: "direct",
		})

		// Handle specific client routing modes
		if g.clientRoutingMode == "white" {
			routing.Rules = append(routing.Rules, getWhiteRoutingRules()...)
			// White mode needs a proxy catch-all at the end
			routing.Rules = append(routing.Rules, RoutingRule{
				Type:        "field",
				Port:        "0-65535",
				OutboundTag: proxyTag,
			})
			return routing
		} else if g.clientRoutingMode == "black" {
			routing.Rules = append(routing.Rules, getBlackRoutingRules()...)
			// Black mode unmatched traffic naturally goes to "direct" because it's first in outbounds,
			// but we append a catch-all direct rule for safety
			routing.Rules = append(routing.Rules, RoutingRule{
				Type:        "field",
				Port:        "0-65535",
				OutboundTag: "direct",
			})
			return routing
		}
		// If "custom", fall back to processing g.rules below
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

// getBlackRoutingRules returns v2rayN custom_routing_black rules (Bypass Mainland / Proxy Blocked Sites)
func getBlackRoutingRules() []RoutingRule {
	return []RoutingRule{
		{Type: "field", Protocol: []string{"bittorrent"}, OutboundTag: "direct"},
		{Type: "field", Domain: []string{"api.ip.sb"}, OutboundTag: "proxy"}, // Typically proxy
		{Type: "field", Port: "443", Network: "udp", OutboundTag: "block"},
		{Type: "field", Domain: []string{"geosite:google"}, OutboundTag: "proxy"},
		{Type: "field", IP: []string{"geoip:private"}, OutboundTag: "direct"},
		{Type: "field", Domain: []string{"geosite:private"}, OutboundTag: "direct"},
		{
			Type: "field",
			IP: []string{
				"1.1.1.1", "1.0.0.1", "2606:4700:4700::1111", "2606:4700:4700::1001",
				"1.1.1.2", "1.0.0.2", "2606:4700:4700::1112", "2606:4700:4700::1002",
				"1.1.1.3", "1.0.0.3", "2606:4700:4700::1113", "2606:4700:4700::1003",
				"8.8.8.8", "8.8.4.4", "2001:4860:4860::8888", "2001:4860:4860::8844",
				"94.140.14.14", "94.140.15.15", "2a10:50c0::ad1:ff", "2a10:50c0::ad2:ff",
				"94.140.14.15", "94.140.15.16", "2a10:50c0::bad1:ff", "2a10:50c0::bad2:ff",
				"94.140.14.140", "94.140.14.141", "2a10:50c0::1:ff", "2a10:50c0::2:ff",
				"208.67.222.222", "208.67.220.220", "2620:119:35::35", "2620:119:53::53",
				"208.67.222.123", "208.67.220.123", "2620:119:35::123", "2620:119:53::123",
				"9.9.9.9", "149.112.112.112", "2620:fe::9", "2620:fe::fe",
				"9.9.9.11", "149.112.112.11", "2620:fe::11", "2620:fe::fe:11",
				"9.9.9.10", "149.112.112.10", "2620:fe::10", "2620:fe::fe:10",
				"77.88.8.8", "77.88.8.1", "2a02:6b8::feed:0ff", "2a02:6b8:0:1::feed:0ff",
				"77.88.8.88", "77.88.8.2", "2a02:6b8::feed:bad", "2a02:6b8:0:1::feed:bad",
				"77.88.8.7", "77.88.8.3", "2a02:6b8::feed:a11", "2a02:6b8:0:1::feed:a11",
			},
			OutboundTag: "proxy",
		},
		{
			Type: "field",
			Domain: []string{
				"domain:cloudflare-dns.com", "domain:one.one.one.one", "domain:dns.google",
				"domain:adguard-dns.com", "domain:opendns.com", "domain:umbrella.com",
				"domain:quad9.net", "domain:yandex.net",
			},
			OutboundTag: "proxy",
		},
		{
			Type: "field",
			IP: []string{
				"geoip:facebook", "geoip:fastly", "geoip:google", "geoip:netflix",
				"geoip:telegram", "geoip:twitter",
			},
			OutboundTag: "proxy",
		},
		{Type: "field", Domain: []string{"geosite:gfw", "geosite:greatfire"}, OutboundTag: "proxy"},
	}
}

// getWhiteRoutingRules returns v2rayN custom_routing_white rules (Proxy Default / Bypass Mainland)
func getWhiteRoutingRules() []RoutingRule {
	return []RoutingRule{
		{Type: "field", Port: "443", Network: "udp", OutboundTag: "block"},
		{Type: "field", Domain: []string{"geosite:google"}, OutboundTag: "proxy"},
		{Type: "field", IP: []string{"geoip:private"}, OutboundTag: "direct"},
		{Type: "field", Domain: []string{"geosite:private"}, OutboundTag: "direct"},
		{
			Type: "field",
			IP: []string{
				"223.5.5.5", "223.6.6.6", "2400:3200::1", "2400:3200:baba::1", "119.29.29.29",
				"1.12.12.12", "120.53.53.53", "2402:4e00::", "2402:4e00:1::", "180.76.76.76",
				"2400:da00::6666", "114.114.114.114", "114.114.115.115", "114.114.114.119",
				"114.114.115.119", "114.114.114.110", "114.114.115.110", "180.184.1.1",
				"180.184.2.2", "101.226.4.6", "218.30.118.6", "123.125.81.6", "140.207.198.6",
				"1.2.4.8", "210.2.4.8", "52.80.66.66", "117.50.22.22", "2400:7fc0:849e:200::4",
				"2404:c2c0:85d8:901::4", "117.50.10.10", "52.80.52.52", "2400:7fc0:849e:200::8",
				"2404:c2c0:85d8:901::8", "117.50.60.30", "52.80.60.30",
			},
			OutboundTag: "direct",
		},
		{
			Type: "field",
			Domain: []string{
				"domain:alidns.com", "domain:doh.pub", "domain:dot.pub",
				"domain:360.cn", "domain:onedns.net",
			},
			OutboundTag: "direct",
		},
		{Type: "field", IP: []string{"geoip:cn"}, OutboundTag: "direct"},
		{Type: "field", Domain: []string{"geosite:cn"}, OutboundTag: "direct"},
	}
}
