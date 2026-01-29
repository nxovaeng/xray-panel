package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DomainType represents the type of domain usage
type DomainType string

const (
	DomainTypeDirect  DomainType = "direct"  // Direct connection, needs TLS cert
	DomainTypeCDN     DomainType = "cdn"     // Behind CDN (Cloudflare, etc.)
	DomainTypeReality DomainType = "reality" // Reality protocol, no cert needed
)

// Domain represents a domain used for proxy connections
type Domain struct {
	ID          string     `json:"id" form:"id" gorm:"primaryKey"`
	Domain      string     `json:"domain" form:"domain" gorm:"uniqueIndex;not null"`
	Type        DomainType `json:"type" form:"type" gorm:"default:direct"`
	ServerName  string     `json:"server_name" form:"server_name"` // SNI for Reality
	Fingerprint string     `json:"fingerprint" form:"fingerprint"` // TLS fingerprint for Reality
	ShortID     string     `json:"short_id" form:"short_id"`       // Reality short ID
	PrivateKey  string     `json:"-" form:"private_key"`           // Reality private key
	PublicKey   string     `json:"public_key" form:"public_key"`   // Reality public key
	CertPath    string     `json:"cert_path" form:"cert_path"`     // TLS certificate path
	KeyPath     string     `json:"key_path" form:"key_path"`       // TLS private key path
	Enabled     bool       `json:"enabled" form:"enabled" gorm:"default:true"`
	CreatedAt   time.Time  `json:"created_at" form:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" form:"updated_at"`
}

// BeforeCreate generates UUID for new domain
func (d *Domain) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// NeedsCert returns true if this domain type requires TLS certificate
func (d *Domain) NeedsCert() bool {
	return d.Type == DomainTypeDirect || d.Type == DomainTypeCDN
}

// IsReality returns true if this domain uses Reality protocol
func (d *Domain) IsReality() bool {
	return d.Type == DomainTypeReality
}
