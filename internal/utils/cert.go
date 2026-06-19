package utils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// ParseCertificateExpiry reads and parses certificate info from a PEM file to get expiration date
func ParseCertificateExpiry(certPath string) (time.Time, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return time.Time{}, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return time.Time{}, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}

	return cert.NotAfter, nil
}
