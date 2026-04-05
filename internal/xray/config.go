package xray

import (
	"encoding/json"
	"strings"

	"xray-panel/internal/models"
)

// Config represents the complete Xray configuration
type Config struct {
	Log       *LogConfig       `json:"log,omitempty"`
	API       *APIConfig       `json:"api,omitempty"`
	DNS       *DNSConfig       `json:"dns,omitempty"`
	Inbounds  []InboundConfig  `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	Routing   *RoutingConfig   `json:"routing,omitempty"`
	Policy    *PolicyConfig    `json:"policy,omitempty"`
	Stats     *StatsConfig     `json:"stats,omitempty"`
}

// LogConfig represents Xray logging configuration
type LogConfig struct {
	Access   string `json:"access,omitempty"`
	Error    string `json:"error,omitempty"`
	LogLevel string `json:"loglevel,omitempty"`
}

// APIConfig represents Xray API configuration
type APIConfig struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

// DNSConfig represents DNS configuration
type DNSConfig struct {
	Servers []interface{} `json:"servers,omitempty"`
	Tag     string        `json:"tag,omitempty"`
}

// StatsConfig enables statistics
type StatsConfig struct{}

// PolicyConfig represents policy configuration
type PolicyConfig struct {
	Levels map[string]PolicyLevel `json:"levels,omitempty"`
	System *SystemPolicy          `json:"system,omitempty"`
}

// PolicyLevel represents a policy level
type PolicyLevel struct {
	Handshake         int  `json:"handshake,omitempty"`
	ConnIdle          int  `json:"connIdle,omitempty"`
	UplinkOnly        int  `json:"uplinkOnly,omitempty"`
	DownlinkOnly      int  `json:"downlinkOnly,omitempty"`
	StatsUserUplink   bool `json:"statsUserUplink,omitempty"`
	StatsUserDownlink bool `json:"statsUserDownlink,omitempty"`
}

// SystemPolicy represents system-wide policies
type SystemPolicy struct {
	StatsInboundUplink    bool `json:"statsInboundUplink,omitempty"`
	StatsInboundDownlink  bool `json:"statsInboundDownlink,omitempty"`
	StatsOutboundUplink   bool `json:"statsOutboundUplink,omitempty"`
	StatsOutboundDownlink bool `json:"statsOutboundDownlink,omitempty"`
}

type Generator struct {
	users     []models.User
	inbounds  []models.Inbound
	outbounds []models.Outbound
	rules     []models.RoutingRule
	domains   map[string]models.Domain
	apiPort   int
	logLevel  string
	socketDir         string
	panelMode         string
	clientRoutingMode string
}

// NewGenerator creates a new configuration generator
func NewGenerator() *Generator {
	return &Generator{
		domains:  make(map[string]models.Domain),
		apiPort:  10085,
		logLevel: "warning",
	}
}

// SetUsers sets the users for configuration
func (g *Generator) SetUsers(users []models.User) *Generator {
	g.users = users
	return g
}

// SetInbounds sets the inbound configurations
func (g *Generator) SetInbounds(inbounds []models.Inbound) *Generator {
	g.inbounds = inbounds
	return g
}

// SetOutbounds sets the outbound configurations
func (g *Generator) SetOutbounds(outbounds []models.Outbound) *Generator {
	g.outbounds = outbounds
	return g
}

// SetRoutingRules sets the routing rules
func (g *Generator) SetRoutingRules(rules []models.RoutingRule) *Generator {
	g.rules = rules
	return g
}

// SetDomains sets the domain configurations
func (g *Generator) SetDomains(domains []models.Domain) *Generator {
	for _, d := range domains {
		g.domains[d.ID] = d
	}
	return g
}

// SetAPIPort sets the API port
func (g *Generator) SetAPIPort(port int) *Generator {
	g.apiPort = port
	return g
}

// SetLogLevel sets the log level
func (g *Generator) SetLogLevel(level string) *Generator {
	g.logLevel = level
	return g
}

// SetSocketDir sets the directory for Unix Domain Sockets
func (g *Generator) SetSocketDir(dir string) *Generator {
	g.socketDir = dir
	return g
}

// SetPanelMode sets the working mode of the panel (server / client)
func (g *Generator) SetPanelMode(mode string) *Generator {
	g.panelMode = mode
	return g
}

// SetClientRoutingMode sets the client routing mode (white / black / custom)
func (g *Generator) SetClientRoutingMode(mode string) *Generator {
	g.clientRoutingMode = mode
	return g
}

// Generate creates the complete Xray configuration
func (g *Generator) Generate() (*Config, error) {
	config := &Config{
		Log: &LogConfig{
			LogLevel: g.logLevel,
		},
		API: &APIConfig{
			Tag:      "api",
			Services: []string{"HandlerService", "LoggerService", "StatsService"},
		},
		Stats: &StatsConfig{},
		Policy: &PolicyConfig{
			Levels: map[string]PolicyLevel{
				"0": {
					Handshake:         4,
					ConnIdle:          300,
					UplinkOnly:        2,
					DownlinkOnly:      5,
					StatsUserUplink:   true,
					StatsUserDownlink: true,
				},
			},
			System: &SystemPolicy{
				StatsInboundUplink:    true,
				StatsInboundDownlink:  true,
				StatsOutboundUplink:   true,
				StatsOutboundDownlink: true,
			},
		},
	}

	if g.panelMode == "client" {
		config.DNS = g.generateDNSForClient()
	} else {
		config.DNS = &DNSConfig{
			Servers: []interface{}{
				"1.1.1.1",
				"8.8.8.8",
				"localhost",
			},
		}
	}

	// Generate API inbound
	config.Inbounds = append(config.Inbounds, g.generateAPIInbound())

	if g.panelMode == "client" {
		config.Inbounds = append(config.Inbounds, InboundConfig{
			Tag:      "dns-in",
			Listen:   "127.0.0.1",
			Port:     53,
			Protocol: "dokodemo-door",
			Settings: map[string]interface{}{
				"address": "1.1.1.1",
				"port":    53,
				"network": "udp",
			},
		})

		// Add SOCKS5 inbound
		config.Inbounds = append(config.Inbounds, InboundConfig{
			Tag:      "socks-in",
			Listen:   "127.0.0.1",
			Port:     10808,
			Protocol: "socks",
			Settings: map[string]interface{}{
				"auth": "noauth",
				"udp":  true,
			},
			Sniffing: &SniffingConfig{
				Enabled:      true,
				DestOverride: []string{"http", "tls", "quic", "fakedns"},
				RouteOnly:    true,
			},
		})

		// Add HTTP inbound
		config.Inbounds = append(config.Inbounds, InboundConfig{
			Tag:      "http-in",
			Listen:   "127.0.0.1",
			Port:     10809,
			Protocol: "http",
			Sniffing: &SniffingConfig{
				Enabled:      true,
				DestOverride: []string{"http", "tls", "quic", "fakedns"},
				RouteOnly:    true,
			},
		})
	}

	// Generate proxy inbounds
	for _, inbound := range g.inbounds {
		if !inbound.Enabled {
			continue
		}
		inboundConfig, err := g.generateInbound(inbound)
		if err != nil {
			return nil, err
		}
		config.Inbounds = append(config.Inbounds, *inboundConfig)
	}

	// Generate outbounds
	config.Outbounds = g.generateOutbounds()

	if g.panelMode == "client" {
		config.Outbounds = append(config.Outbounds, OutboundConfig{
			Protocol: "dns",
			Tag:      "dns-out",
		})
	}

	// Generate routing
	config.Routing = g.generateRouting()

	return config, nil
}

// generateDNSForClient generates comprehensive anti-poison DNS arrays needed for the proxy client
func (g *Generator) generateDNSForClient() *DNSConfig {
	return &DNSConfig{
		Servers: []interface{}{
			// Remote DNS (Anti-poison) - Use tcp:// to ensure it goes through ROUTING and thus PROXY
			map[string]interface{}{
				"address": "tcp://1.1.1.1",
				"domains": []string{
					"geosite:geolocation-!cn",
				},
				"expectIPs": []string{
					"geoip:!cn",
				},
			},
			// Domestic DNS - Use DOHL (https+local) to BYPASS ROUTING for better performance
			map[string]interface{}{
				"address": "https+local://223.5.5.5/dns-query",
				"domains": []string{
					"geosite:cn",
				},
				"expectIPs": []string{
					"geoip:cn",
				},
			},
			"localhost",
		},
	}
}

// GenerateJSON generates the configuration as JSON
func (g *Generator) GenerateJSON() ([]byte, error) {
	config, err := g.Generate()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(config, "", "  ")
}

// generateAPIInbound creates the API inbound
func (g *Generator) generateAPIInbound() InboundConfig {
	return InboundConfig{
		Tag:      "api",
		Listen:   "127.0.0.1",
		Port:     g.apiPort,
		Protocol: "dokodemo-door",
		Settings: map[string]interface{}{
			"address": "127.0.0.1",
		},
	}
}

// getActiveUsers returns only active users
func (g *Generator) getActiveUsers() []models.User {
	var active []models.User
	for _, u := range g.users {
		if u.IsActive() {
			active = append(active, u)
		}
	}
	return active
}

// splitCSV splits a comma or newline-separated string into a slice
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	
	// Split by comma or newline
	f := func(c rune) bool {
		return c == ',' || c == '\n' || c == '\r'
	}
	parts := strings.FieldsFunc(s, f)
	
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
