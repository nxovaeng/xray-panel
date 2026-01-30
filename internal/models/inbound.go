package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Protocol represents the proxy protocol type
type Protocol string

const (
	ProtocolVLESS  Protocol = "vless"  // VLESS
	ProtocolTrojan Protocol = "trojan" // Trojan
)

// Transport represents the transport layer type
type Transport string

const (
	TransportWS    Transport = "ws"    // WebSocket (可被 Nginx 反代)
	TransportGRPC  Transport = "grpc"  // gRPC (可被 Nginx 反代)
	TransportXHTTP Transport = "xhttp" // XHTTP (可被 Nginx 反代)
)

// Security represents the TLS/security type
type Security string

const (
	SecurityNone Security = "none" // 始终为 none，由 Nginx 处理 TLS
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
	Path        string `json:"path" form:"path"`                 // For WS/XHTTP: /path
	ServiceName string `json:"service_name" form:"service_name"` // For gRPC: service name
	Host        string `json:"host" form:"host"`                 // For WS/XHTTP: Host header

	// XHTTP specific
	Mode string `json:"mode" form:"mode"` // auto, packet-up, stream-up (default: auto)

	// Wildcard certificate support
	ActualDomain string `json:"actual_domain" form:"actual_domain"` // Generated subdomain for wildcard certs (e.g., abc123.example.com)

	Enabled   bool      `json:"enabled" form:"enabled" gorm:"default:true"`
	Remark    string    `json:"remark" form:"remark"`
	CreatedAt time.Time `json:"created_at" form:"created_at"`
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
func (i *Inbound) IsGRPC() bool {
	return i.Transport == TransportGRPC
}

// IsXHTTP returns true if transport is XHTTP
func (i *Inbound) IsXHTTP() bool {
	return i.Transport == TransportXHTTP
}

// IsWS returns true if transport is WebSocket
func (i *Inbound) IsWS() bool {
	return i.Transport == TransportWS
}

// IsTrojan returns true if protocol is Trojan
func (i *Inbound) IsTrojan() bool {
	return i.Protocol == ProtocolTrojan
}

// IsVLESS returns true if protocol is VLESS
func (i *Inbound) IsVLESS() bool {
	return i.Protocol == ProtocolVLESS
}
