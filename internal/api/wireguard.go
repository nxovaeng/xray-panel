package api

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"

	"xray-panel/internal/models"
	"xray-panel/internal/utils"
	"xray-panel/internal/wireguard"
)

// handleGenerateWGKeys generates a WireGuard (Curve25519) key pair.
// Returns JSON: { "private_key": "...", "public_key": "..." }
func (s *Server) handleGenerateWGKeys(c *gin.Context) {
	privKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to generate key: "+err.Error())
		return
	}

	privB64 := base64.StdEncoding.EncodeToString(privKey.Bytes())
	pubB64 := base64.StdEncoding.EncodeToString(privKey.PublicKey().Bytes())

	// Sanity-check via shared utility (also validates round-trip)
	if _, err := utils.DeriveWGPublicKey(privB64); err != nil {
		jsonError(c, http.StatusInternalServerError, "Key derivation check failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"private_key": privB64,
		"public_key":  pubB64,
	})
}

// handleWGStatus returns overall service and interface status
func (s *Server) handleWGStatus(c *gin.Context) {
	mgr := wireguard.NewManager(s.db)
	status, err := mgr.GetStatus()
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to get WireGuard status: "+err.Error())
		return
	}
	jsonOK(c, status)
}

// handleWGServiceAction starts, stops, restarts, enables or disables wg-quick@wg0
func (s *Server) handleWGServiceAction(c *gin.Context) {
	action := c.Param("action")
	mgr := wireguard.NewManager(s.db)
	if err := mgr.ServiceAction(action); err != nil {
		jsonError(c, http.StatusInternalServerError, "Service action failed: "+err.Error())
		return
	}
	jsonOK(c, gin.H{"action": action, "success": true})
}

// handleGetWGServer returns current server configuration
func (s *Server) handleGetWGServer(c *gin.Context) {
	serverCfg, err := models.GetWGServerConfig(s.db)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "Failed to get server config: "+err.Error())
		return
	}
	jsonOK(c, serverCfg)
}

// handleWGSetupEnv automatically installs wireguard-tools and configures IPv4/IPv6 forwarding
func (s *Server) handleWGSetupEnv(c *gin.Context) {
	msg, err := wireguard.SetupEnvironment()
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "环境初始化失败: "+err.Error()+"\n"+msg)
		return
	}
	jsonOK(c, gin.H{
		"success": true,
		"message": msg,
	})
}
