package utils

import (
	"crypto/ecdh"
	"encoding/base64"
	"fmt"
)

// DeriveWGPublicKey derives the WireGuard (X25519) public key from a
// base64-encoded private key. Returns the base64-encoded public key.
func DeriveWGPublicKey(privKeyB64 string) (string, error) {
	privBytes, err := base64.StdEncoding.DecodeString(privKeyB64)
	if err != nil {
		return "", fmt.Errorf("invalid base64 private key: %w", err)
	}
	privKey, err := ecdh.X25519().NewPrivateKey(privBytes)
	if err != nil {
		return "", fmt.Errorf("invalid WireGuard private key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(privKey.PublicKey().Bytes()), nil
}
