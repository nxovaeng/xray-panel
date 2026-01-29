package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a proxy user with traffic and time limits
type User struct {
	ID           string    `json:"id" form:"id" gorm:"primaryKey"`
	UUID         string    `json:"uuid" form:"uuid" gorm:"uniqueIndex;not null"` // Xray user UUID
	Name         string    `json:"name" form:"name" gorm:"not null"`
	Email        string    `json:"email" form:"email"`
	TrafficLimit int64     `json:"traffic_limit" form:"traffic_limit"` // bytes, 0 = unlimited
	TrafficUsed  int64     `json:"traffic_used" form:"traffic_used"`
	TrafficReset time.Time `json:"traffic_reset" form:"traffic_reset"` // when to reset traffic
	ExpiryDate   time.Time `json:"expiry_date" form:"expiry_date"`     // zero = never expires
	Enabled      bool      `json:"enabled" form:"enabled" gorm:"default:true"`
	SubPath      string    `json:"sub_path" form:"sub_path" gorm:"uniqueIndex"` // subscription URL path
	Note         string    `json:"note" form:"note"`
	CreatedAt    time.Time `json:"created_at" form:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" form:"updated_at"`
}

// BeforeCreate generates UUID and subscription path for new user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.UUID == "" {
		u.UUID = uuid.New().String()
	}
	if u.SubPath == "" {
		u.SubPath = uuid.New().String()[:8] // short path for subscription URL
	}
	return nil
}

// IsActive checks if user is enabled and not expired
func (u *User) IsActive() bool {
	if !u.Enabled {
		return false
	}
	// Check expiry
	if !u.ExpiryDate.IsZero() && time.Now().After(u.ExpiryDate) {
		return false
	}
	// Check traffic limit
	if u.TrafficLimit > 0 && u.TrafficUsed >= u.TrafficLimit {
		return false
	}
	return true
}

// RemainingTraffic returns remaining traffic in bytes, -1 if unlimited
func (u *User) RemainingTraffic() int64 {
	if u.TrafficLimit == 0 {
		return -1
	}
	remaining := u.TrafficLimit - u.TrafficUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RemainingDays returns days until expiry, -1 if never expires
func (u *User) RemainingDays() int {
	if u.ExpiryDate.IsZero() {
		return -1
	}
	days := int(time.Until(u.ExpiryDate).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
