package api

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
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

	c.JSON(http.StatusOK, gin.H{
		"private_key": privB64,
		"public_key":  pubB64,
	})
}
