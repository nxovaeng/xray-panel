package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OutboundType represents the type of outbound connection
type OutboundType string

const (
	OutboundDirect    OutboundType = "direct"    // Direct internet access
	OutboundSOCKS5    OutboundType = "socks5"    // SOCKS5 proxy (转到本地其他代理)
	OutboundWireGuard OutboundType = "wireguard" // WireGuard (WARP, Proton VPN 等)
	OutboundTrojan    OutboundType = "trojan"    // Trojan (自己的其他服务器，落地IP用途)
	OutboundBlackhole OutboundType = "blackhole" // Block traffic
)

// Outbound represents an Xray outbound configuration
type Outbound struct {
	ID   string       `json:"id" form:"id" gorm:"primaryKey"`
	Tag  string       `json:"tag" form:"tag" gorm:"uniqueIndex;not null"`
	Type OutboundType `json:"type" form:"type" gorm:"not null"`

	// SOCKS5 specific settings (转到本地其他代理)
	Server   string `json:"server" form:"server"`     // SOCKS5 server address
	Port     int    `json:"port" form:"port"`         // SOCKS5 server port
	Username string `json:"username" form:"username"` // SOCKS5 auth username
	Password string `json:"-" form:"password"`        // SOCKS5 auth password

	// WireGuard specific settings (WARP, Proton VPN 等)
	WGSecretKey string `json:"-" form:"wg_secret_key"`             // WireGuard private key
	WGPublicKey string `json:"wg_public_key" form:"wg_public_key"` // WireGuard server public key
	WGEndpoint  string `json:"wg_endpoint" form:"wg_endpoint"`     // WireGuard endpoint (host:port)
	WGReserved  string `json:"wg_reserved" form:"wg_reserved"`     // Reserved bytes (WARP only, optional)
	WGLocalIPv4 string `json:"wg_local_ipv4" form:"wg_local_ipv4"` // Assigned IPv4
	WGLocalIPv6 string `json:"wg_local_ipv6" form:"wg_local_ipv6"` // Assigned IPv6
	WGMTU       int    `json:"wg_mtu" form:"wg_mtu"`               // MTU (default: 1420)
	WGDNS       string `json:"wg_dns" form:"wg_dns"`               // DNS servers (optional, for reference)

	// Trojan specific settings (自己的其他服务器，落地IP用途)
	TrojanPassword string `json:"-" form:"trojan_password"`             // Trojan password
	TrojanSNI      string `json:"trojan_sni" form:"trojan_sni"`         // Server Name Indication
	TrojanNetwork  string `json:"trojan_network" form:"trojan_network"` // tcp/ws/grpc

	Enabled   bool      `json:"enabled" form:"enabled" gorm:"default:true"`
	Priority  int       `json:"priority" form:"priority" gorm:"default:0"` // Higher = preferred
	Remark    string    `json:"remark" form:"remark"`
	CreatedAt time.Time `json:"created_at" form:"created_at"`
	UpdatedAt time.Time `json:"updated_at" form:"updated_at"`
}

// BeforeCreate generates UUID for new outbound
func (o *Outbound) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	if o.Tag == "" {
		o.Tag = string(o.Type) + "-" + o.ID[:8]
	}
	return nil
}

// IsDirect returns true if this is a direct outbound
func (o *Outbound) IsDirect() bool {
	return o.Type == OutboundDirect
}

// IsWireGuard returns true if this is a WireGuard outbound
func (o *Outbound) IsWireGuard() bool {
	return o.Type == OutboundWireGuard
}

// IsTrojan returns true if this is a Trojan outbound
func (o *Outbound) IsTrojan() bool {
	return o.Type == OutboundTrojan
}

// IsSOCKS5 returns true if this is a SOCKS5 outbound
func (o *Outbound) IsSOCKS5() bool {
	return o.Type == OutboundSOCKS5
}

// IsBlackhole returns true if this is a blackhole outbound
func (o *Outbound) IsBlackhole() bool {
	return o.Type == OutboundBlackhole
}
