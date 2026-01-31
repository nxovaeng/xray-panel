package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NginxConfig represents a generated Nginx configuration file
type NginxConfig struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	InboundID  string    `json:"inbound_id" gorm:"index"`           // 关联的 Inbound ID
	Inbound    *Inbound  `json:"inbound,omitempty" gorm:"foreignKey:InboundID"`
	Domain     string    `json:"domain"`                            // 配置的域名（可能是生成的子域名）
	ConfigPath string    `json:"config_path"`                       // Nginx 配置文件路径
	ConfigType string    `json:"config_type" gorm:"default:http"`   // http, stream, panel
	IsManaged  bool      `json:"is_managed" gorm:"default:true"`    // 是否由面板管理
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BeforeCreate generates UUID for new nginx config
func (n *NginxConfig) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}
