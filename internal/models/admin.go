package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Admin represents an administrator account
type Admin struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BeforeCreate generates UUID for new admin
func (a *Admin) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// SetPassword hashes and sets the password
func (a *Admin) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies a password against the stored hash
func (a *Admin) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password))
	return err == nil
}
