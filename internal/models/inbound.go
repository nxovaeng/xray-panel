package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Protocol represents the proxy protocol type
type Protocol string

const (
	ProtocolVLESS     Protocol = "vless"
	ProtocolTrojan    Protocol = "trojan"
	ProtocolWireGuard Protocol = "wireguard" // WireGuard 入站（用于接收其他节点转发的流量）
)

// Transport represents the transport layer type
type Transport string

const (
	TransportWS    Transport = "ws"
	TransportGRPC  Transport = "grpc"
	TransportXHTTP Transport = "xhttp"
	TransportRAW   Transport = "raw" // WireGuard 使用 raw/UDP
)

// Security represents the TLS/security type
type Security string

const (
	SecurityNone Security = "none"
)

// Inbound represents an Xray inbound configuration
type Inbound struct {
	ID        string    `json:"id" form:"id" gorm:"primaryKey"`
	Tag       string    `json:"tag" form:"tag" gorm:"uniqueIndex;not null"`
	Protocol  Protocol  `json:"protocol" form:"protocol" gorm:"default:vless"`
	Transport Transport `json:"transport" form:"transport" gorm:"not null"`
	Port      int       `json:"port" form:"port" gorm:"not null"`
	Listen    string    `json:"listen" form:"listen" gorm:"default:127.0.0.1"`

	// Domain for TLS (handled by Nginx)
	DomainID string  `json:"domain_id" form:"domain_id"`
	Domain   *Domain `json:"domain,omitempty" gorm:"foreignKey:DomainID"`

	// Transport-specific settings
	Path        string `json:"path" form:"path"`
	ServiceName string `json:"service_name" form:"service_name"`
	Host        string `json:"host" form:"host"`

	// XHTTP specific (unused: server mode is always "auto", kept for DB compatibility)
	Mode string `json:"mode" form:"mode"`

	// Wildcard certificate support
	ActualDomain string `json:"actual_domain" form:"actual_domain"`

	// SNI for Cloudflare Tunnel / custom scenarios
	CustomSNI string `json:"custom_sni" form:"custom_sni"`

	// CDN connect domain
	ConnectDomain string `json:"connect_domain" form:"connect_domain"`

	// WireGuard specific fields
	WGSecretKey  string `json:"-" form:"wg_secret_key"`              // 服务端私钥（不暴露到 JSON）
	WGPublicKey  string `json:"wg_public_key" form:"wg_public_key"`  // 服务端公钥（展示用）
	WGPeerPubKey string `json:"wg_peer_pub_key" form:"wg_peer_pub_key"` // 对端（出站节点）公钥
	WGMTU        int    `json:"wg_mtu" form:"wg_mtu"`                // MTU，默认 1420
	WGLocalIP    string `json:"wg_local_ip" form:"wg_local_ip"`      // 本端 WireGuard 虚拟 IP，如 10.0.0.1/24

	// 是否排除在订阅链接之外（WireGuard 入站等内部中转节点不应出现在用户订阅中）
	ExcludeFromSub bool `json:"exclude_from_sub" form:"exclude_from_sub" gorm:"default:false"`

	Enabled   bool      `json:"enabled" form:"enabled" gorm:"default:true;index"`
	UseUDS    bool      `json:"use_uds" form:"use_uds" gorm:"default:true"`
	Remark    string    `json:"remark" form:"remark"`
	CreatedAt time.Time `json:"created_at" form:"created_at" gorm:"index"`
	UpdatedAt time.Time `json:"updated_at" form:"updated_at"`
}

// BeforeCreate generates UUID for new inbound
func (i *Inbound) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	if i.Tag == "" {
		i.Tag = "inbound-" + i.ID[:8]
	}
	return nil
}

// IsGRPC returns true if transport is gRPC
func (i *Inbound) IsGRPC() bool { return i.Transport == TransportGRPC }

// IsXHTTP returns true if transport is XHTTP
func (i *Inbound) IsXHTTP() bool { return i.Transport == TransportXHTTP }

// IsWS returns true if transport is WebSocket
func (i *Inbound) IsWS() bool { return i.Transport == TransportWS }

// IsTrojan returns true if protocol is Trojan
func (i *Inbound) IsTrojan() bool { return i.Protocol == ProtocolTrojan }

// IsVLESS returns true if protocol is VLESS
func (i *Inbound) IsVLESS() bool { return i.Protocol == ProtocolVLESS }

// IsWireGuard returns true if protocol is WireGuard
func (i *Inbound) IsWireGuard() bool { return i.Protocol == ProtocolWireGuard }

// SocketPath returns the Unix Domain Socket path for this inbound.
func (i *Inbound) SocketPath(socketDir string) string {
	if socketDir == "" {
		socketDir = "/dev/shm"
	}
	return fmt.Sprintf("%s/xray-%s.sock", socketDir, i.Tag)
}
