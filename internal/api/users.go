package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"xray-panel/internal/models"
)

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email"`
	TrafficLimit int64  `json:"traffic_limit"` // bytes, 0 = unlimited
	ExpireDays   int    `json:"expire_days"`   // 0 = never expires
	Note         string `json:"note"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	TrafficLimit int64  `json:"traffic_limit"`
	ExpiryDate   string `json:"expiry_date"` // RFC3339 format
	Enabled      bool   `json:"enabled"`
	Note         string `json:"note"`
}

// handleListUsers returns all users
func (s *Server) handleListUsers(c *gin.Context) {
	var users []models.User
	if err := s.db.Order("created_at DESC").Find(&users).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	jsonOK(c, users)
}

// handleCreateUser creates a new user
func (s *Server) handleCreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	user := models.User{
		ID:           uuid.New().String(),
		UUID:         uuid.New().String(),
		Name:         req.Name,
		Email:        req.Email,
		TrafficLimit: req.TrafficLimit,
		Enabled:      true,
		SubPath:      uuid.New().String()[:8],
		Note:         req.Note,
	}

	// Set expiry date if specified
	if req.ExpireDays > 0 {
		user.ExpiryDate = time.Now().AddDate(0, 0, req.ExpireDays)
	}

	if err := s.db.Create(&user).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	jsonCreated(c, user)
}

// handleGetUser returns a single user
func (s *Server) handleGetUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "User not found")
		return
	}

	jsonOK(c, user)
}

// handleUpdateUser updates a user
func (s *Server) handleUpdateUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		jsonError(c, http.StatusNotFound, "User not found")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	// Update fields
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	user.TrafficLimit = req.TrafficLimit
	user.Enabled = req.Enabled
	user.Note = req.Note

	// Parse expiry date
	if req.ExpiryDate != "" {
		expiryDate, err := time.Parse(time.RFC3339, req.ExpiryDate)
		if err == nil {
			user.ExpiryDate = expiryDate
		}
	}

	if err := s.db.Save(&user).Error; err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to update user")
		return
	}

	jsonOK(c, user)
}

// handleDeleteUser deletes a user
func (s *Server) handleDeleteUser(c *gin.Context) {
	id := c.Param("id")

	result := s.db.Delete(&models.User{}, "id = ?", id)
	if result.Error != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to delete user")
		return
	}
	if result.RowsAffected == 0 {
		jsonError(c, http.StatusNotFound, "User not found")
		return
	}

	jsonOK(c, gin.H{"deleted": true})
}

// handleResetUserTraffic resets a user's traffic usage
func (s *Server) handleResetUserTraffic(c *gin.Context) {
	id := c.Param("id")

	result := s.db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"traffic_used":  0,
		"traffic_reset": time.Now(),
	})

	if result.Error != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to reset traffic")
		return
	}
	if result.RowsAffected == 0 {
		jsonError(c, http.StatusNotFound, "User not found")
		return
	}

	jsonOK(c, gin.H{"reset": true})
}
