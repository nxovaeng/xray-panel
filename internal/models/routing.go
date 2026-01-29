package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RuleType represents the type of routing rule
type RuleType string

const (
	RuleTypeInbound  RuleType = "inbound"  // Inbound-based routing
	RuleTypeDomain   RuleType = "domain"   // Domain list routing
	RuleTypeIP       RuleType = "ip"       // IP list routing
	RuleTypeGeoSite  RuleType = "geosite"  // GeoSite category routing
	RuleTypeGeoIP    RuleType = "geoip"    // GeoIP country routing
	RuleTypeProtocol RuleType = "protocol" // Protocol-based (e.g., bittorrent)
)

// RoutingRule represents a routing rule for Xray
// Each rule matches ONLY ONE type of condition
type RoutingRule struct {
	ID   string   `json:"id" form:"id" gorm:"primaryKey"`
	Name string   `json:"name" form:"name" gorm:"not null"`
	Type RuleType `json:"type" form:"type" gorm:"not null"`

	// Rule matching value (only one field is used based on Type)
	// Type=inbound:  InboundTag (single inbound tag)
	// Type=domain:   Domains (comma-separated domain rules)
	// Type=ip:       IPs (comma-separated IP/CIDR)
	// Type=geosite:  GeoSiteTags (comma-separated geosite tags)
	// Type=geoip:    GeoIPCodes (comma-separated country codes)
	// Type=protocol: Protocols (comma-separated protocols)
	InboundTag  string `json:"inbound_tag" form:"inbound_tag"`   // For Type=inbound
	Domains     string `json:"domains" form:"domains"`           // For Type=domain: domain:example.com, full:www.example.com
	IPs         string `json:"ips" form:"ips"`                   // For Type=ip: 192.168.0.0/16, 8.8.8.8
	GeoSiteTags string `json:"geosite_tags" form:"geosite_tags"` // For Type=geosite: category-ads, cn, geolocation-cn
	GeoIPCodes  string `json:"geoip_codes" form:"geoip_codes"`   // For Type=geoip: cn, us, private
	Protocols   string `json:"protocols" form:"protocols"`       // For Type=protocol: bittorrent, http, tls

	// Target outbound
	OutboundTag string `json:"outbound_tag" form:"outbound_tag" gorm:"not null"` // direct, warp, block, etc.

	// Rule order (lower = higher priority)
	Priority int  `json:"priority" form:"priority" gorm:"default:100"`
	Enabled  bool `json:"enabled" form:"enabled" gorm:"default:true"`

	Remark    string    `json:"remark" form:"remark"`
	CreatedAt time.Time `json:"created_at" form:"created_at"`
	UpdatedAt time.Time `json:"updated_at" form:"updated_at"`
}

// BeforeCreate generates UUID for new rule
func (r *RoutingRule) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

// DefaultRoutingRules returns the default basic routing rules
func DefaultRoutingRules() []RoutingRule {
	return []RoutingRule{
		// Block Ads
		{
			Name:        "屏蔽广告域名",
			Type:        RuleTypeGeoSite,
			GeoSiteTags: "category-ads,category-ads-all",
			OutboundTag: "block",
			Priority:    10,
			Enabled:     true,
			Remark:      "屏蔽常见广告和追踪域名",
		},
		// Block BitTorrent
		{
			Name:        "屏蔽 BT 协议",
			Type:        RuleTypeProtocol,
			Protocols:   "bittorrent",
			OutboundTag: "block",
			Priority:    20,
			Enabled:     true,
			Remark:      "阻止 BitTorrent 流量",
		},
		// Private/LAN IPs - Direct
		{
			Name:        "私有网络直连",
			Type:        RuleTypeGeoIP,
			GeoIPCodes:  "private",
			OutboundTag: "direct",
			Priority:    90,
			Enabled:     true,
			Remark:      "局域网和私有IP地址直接连接",
		},
	}
}

// PresetRoutingRules returns preset rule templates for quick import
// These can be imported after user creates corresponding outbounds
func PresetRoutingRules() map[string][]RoutingRule {
	return map[string][]RoutingRule{
		"warp-china": {
			{
				Name:        "中国网站经 WARP",
				Type:        RuleTypeGeoSite,
				GeoSiteTags: "cn,geolocation-cn",
				OutboundTag: "warp", // User needs to create a WARP outbound first
				Priority:    50,
				Enabled:     true,
				Remark:      "中国大陆网站通过 WARP 访问（需要先创建 WARP 出站）",
			},
			{
				Name:        "中国 IP 经 WARP",
				Type:        RuleTypeGeoIP,
				GeoIPCodes:  "cn",
				OutboundTag: "warp",
				Priority:    51,
				Enabled:     true,
				Remark:      "中国大陆 IP 通过 WARP 访问（需要先创建 WARP 出站）",
			},
		},
		"warp-streaming": {
			{
				Name:        "流媒体经 WARP",
				Type:        RuleTypeGeoSite,
				GeoSiteTags: "netflix,disney,youtube,spotify,hulu,hbo,primevideo",
				OutboundTag: "warp",
				Priority:    60,
				Enabled:     true,
				Remark:      "流媒体服务通过 WARP 访问（需要先创建 WARP 出站）",
			},
		},
		"china-direct": {
			{
				Name:        "中国网站直连",
				Type:        RuleTypeGeoSite,
				GeoSiteTags: "cn,geolocation-cn",
				OutboundTag: "direct",
				Priority:    50,
				Enabled:     true,
				Remark:      "中国大陆网站直接连接",
			},
			{
				Name:        "中国 IP 直连",
				Type:        RuleTypeGeoIP,
				GeoIPCodes:  "cn",
				OutboundTag: "direct",
				Priority:    51,
				Enabled:     true,
				Remark:      "中国大陆 IP 直接连接",
			},
		},
	}
}
