package models

import (
	"time"

	"gorm.io/gorm"
)

// Setting represents a key-value system setting
type Setting struct {
	Key       string    `json:"key" gorm:"primaryKey"`
	Value     string    `json:"value"`
	Type      string    `json:"type" gorm:"default:string"` // string, int, bool, json
	Remark    string    `json:"remark"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeSave updates the timestamp
func (s *Setting) BeforeSave(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return nil
}

// DefaultSettings returns default system settings
func DefaultSettings() []Setting {
	return []Setting{
		{Key: "panel_title", Value: "Xray Panel", Type: "string", Remark: "Panel title"},
		{Key: "panel_mode", Value: "server", Type: "string", Remark: "Panel working mode (server / client)"},
		{Key: "client_routing_mode", Value: "white", Type: "string", Remark: "Client routing mode (white / black / custom)"},
		{Key: "sub_domain", Value: "", Type: "string", Remark: "Subscription domain"},
		{Key: "sub_path", Value: "/d", Type: "string", Remark: "Subscription URL path prefix"},
		{Key: "xray_log_level", Value: "warning", Type: "string", Remark: "Xray log level"},
		{Key: "enable_traffic_stats", Value: "true", Type: "bool", Remark: "Enable traffic statistics"},
		{Key: "enable_sniffing", Value: "true", Type: "bool", Remark: "Enable traffic sniffing"},
		{Key: "default_traffic_limit", Value: "0", Type: "int", Remark: "Default traffic limit (0=unlimited)"},
		{Key: "default_expire_days", Value: "30", Type: "int", Remark: "Default expiry days for new users"},
		{Key: "direct_domain_strategy", Value: "UseIPv4", Type: "string", Remark: "Domain strategy for direct outbound"},
	}
}

// GetPanelMode returns the current panel mode ("server" or "client")
func GetPanelMode(db *gorm.DB) string {
	var setting Setting
	if err := db.First(&setting, "key = ?", "panel_mode").Error; err != nil {
		return "server" // Default to server if not found
	}
	if setting.Value == "client" {
		return "client"
	}
	return "server"
}

// GetClientRoutingMode returns the current client routing mode ("white", "black" or "custom")
func GetClientRoutingMode(db *gorm.DB) string {
	var setting Setting
	if err := db.First(&setting, "key = ?", "client_routing_mode").Error; err != nil {
		return "white" // Default to white if not found
	}
	if setting.Value == "black" || setting.Value == "custom" {
		return setting.Value
	}
	return "white"
}

// GetDirectDomainStrategy returns the domain strategy for direct outbound
func GetDirectDomainStrategy(db *gorm.DB) string {
	var setting Setting
	if err := db.First(&setting, "key = ?", "direct_domain_strategy").Error; err != nil {
		return "UseIPv4"
	}
	if setting.Value != "" {
		return setting.Value
	}
	return "UseIPv4"
}

// GetSubPath returns the subscription URL path prefix (e.g. "/d")
func GetSubPath(db *gorm.DB) string {
	var setting Setting
	if err := db.First(&setting, "key = ?", "sub_path").Error; err != nil {
		return "/d"
	}
	if setting.Value != "" {
		return setting.Value
	}
	return "/d"
}
