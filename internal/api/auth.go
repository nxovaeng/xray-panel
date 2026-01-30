package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"xray-panel/internal/logger"
	"xray-panel/internal/models"
)

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse contains the JWT token
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// Claims represents JWT claims
type Claims struct {
	AdminID  string `json:"admin_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// handleLogin authenticates admin and returns JWT
func (s *Server) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	// Find admin by username
	var admin models.Admin
	if err := s.db.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		jsonError(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Verify password
	if !admin.CheckPassword(req.Password) {
		jsonError(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate JWT token
	expiresAt := time.Now().Add(time.Duration(s.config.JWT.ExpireHour) * time.Hour)
	claims := &Claims{
		AdminID:  admin.ID,
		Username: admin.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "xray-panel",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	jsonOK(c, LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Unix(),
	})
}

// authMiddleware validates JWT tokens (supports both Bearer token and cookie)
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try to get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Extract token from "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// If no Authorization header, try to get token from cookie
		if tokenString == "" {
			var err error
			tokenString, err = c.Cookie("session_token")
			if err != nil || tokenString == "" {
				jsonError(c, http.StatusUnauthorized, "Authorization required")
				c.Abort()
				return
			}
		}

		// Parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.config.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			jsonError(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		// Store admin info in context
		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// webAuthMiddleware validates session for web pages
func (s *Server) webAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check session cookie
		sessionToken, err := c.Cookie("session_token")
		if err != nil || sessionToken == "" {
			// Redirect to login page
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(sessionToken, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.config.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			// Clear invalid cookie and redirect to login
			c.SetCookie("session_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Store admin info in context
		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// handleWebLogin handles web-based login (form submission)
func (s *Server) handleWebLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"Error": "用户名和密码不能为空",
		})
		return
	}

	// Find admin by username
	var admin models.Admin
	if err := s.db.Where("username = ?", username).First(&admin).Error; err != nil {
		logger.Warn("Failed login attempt for username: %s", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Error": "用户名或密码错误",
		})
		return
	}

	// Verify password
	if !admin.CheckPassword(password) {
		logger.Warn("Failed login attempt for username: %s (invalid password)", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Error": "用户名或密码错误",
		})
		return
	}

	// Generate JWT token
	expiresAt := time.Now().Add(time.Duration(s.config.JWT.ExpireHour) * time.Hour)
	claims := &Claims{
		AdminID:  admin.ID,
		Username: admin.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "xray-panel",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"Error": "登录失败，请重试",
		})
		return
	}

	// Set session cookie
	c.SetCookie(
		"session_token",
		tokenString,
		int(s.config.JWT.ExpireHour)*3600, // seconds
		"/",
		"",
		false, // secure (set to true in production with HTTPS)
		true,  // httpOnly
	)

	logger.Info("Admin logged in: %s", username)
	// Redirect to dashboard
	c.Redirect(http.StatusFound, "/dashboard")
}

// handleWebLogout handles web-based logout
func (s *Server) handleWebLogout(c *gin.Context) {
	// Get username for logging if available
	if username, exists := c.Get("username"); exists {
		logger.Info("Admin logged out: %s", username)
	}
	
	// Clear session cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

// handleLogout handles API logout (clears cookie and returns JSON)
func (s *Server) handleLogout(c *gin.Context) {
	// Get username for logging if available
	if username, exists := c.Get("username"); exists {
		logger.Info("Admin logged out via API: %s", username)
	}

	// Clear session cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)
	jsonOK(c, gin.H{"message": "Logged out successfully"})
}

// getAdminID returns the current admin ID from context
func getAdminID(c *gin.Context) string {
	adminID, _ := c.Get("admin_id")
	return adminID.(string)
}
