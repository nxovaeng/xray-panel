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

// Generator generates Xray configuration
type Generator struct {
	users     []models.User
	inbounds  []models.Inbound
	outbounds []models.Outbound
	rules     []models.RoutingRule
	domains   map[string]models.Domain
	apiPort   int
	logLevel  string
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
		DNS: &DNSConfig{
			Servers: []interface{}{
				"https+local://1.1.1.1/dns-query",
				"https+local://8.8.8.8/dns-query",
				"localhost",
			},
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

	// Generate API inbound
	config.Inbounds = append(config.Inbounds, g.generateAPIInbound())

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

	// Generate routing
	config.Routing = g.generateRouting()

	return config, nil
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

// splitCSV splits a comma-separated string into a slice
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
