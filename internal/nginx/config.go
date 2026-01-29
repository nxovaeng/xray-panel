package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"xray-panel/internal/models"
)

// ConfigGenerator handles generating Nginx configurations
type ConfigGenerator struct {
	configDir string
	streamDir string
}

// NewGenerator creates a new Nginx config generator
func NewGenerator(configDir, streamDir string) *ConfigGenerator {
	return &ConfigGenerator{
		configDir: configDir,
		streamDir: streamDir,
	}
}

// GenerateHTTPConfig generates Nginx HTTP server blocks for inbounds
// All inbounds use Nginx for TLS termination on port 443
// Supports WebSocket, gRPC, and XHTTP transports
func (g *ConfigGenerator) GenerateHTTPConfig(inbounds []models.Inbound) error {
	// Group inbounds by domain
	domainInbounds := make(map[string][]models.Inbound)
	for _, i := range inbounds {
		if i.Domain != nil {
			// All inbounds with domains need HTTP config
			// Nginx handles TLS termination on port 443
			domainInbounds[i.Domain.Domain] = append(domainInbounds[i.Domain.Domain], i)
		}
	}

	// Generate config for each domain
	for domain, inbounds := range domainInbounds {
		conf := g.buildServerBlock(domain, inbounds)
		filename := filepath.Join(g.configDir, fmt.Sprintf("%s.conf", domain))
		if err := os.WriteFile(filename, []byte(conf), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (g *ConfigGenerator) buildServerBlock(domain string, inbounds []models.Inbound) string {
	var sb strings.Builder
	
	// HTTP to HTTPS redirect
	sb.WriteString("# HTTP to HTTPS redirect\n")
	sb.WriteString("server {\n")
	sb.WriteString("    listen 80;\n")
	sb.WriteString("    listen [::]:80;\n")
	sb.WriteString(fmt.Sprintf("    server_name %s;\n", domain))
	sb.WriteString("    return 301 https://$host$request_uri;\n")
	sb.WriteString("}\n\n")

	// HTTPS server block
	// Listen on 443 - Nginx handles TLS termination
	sb.WriteString("# HTTPS server block\n")
	sb.WriteString("server {\n")
	sb.WriteString("    listen 443 ssl http2;\n")
	sb.WriteString("    listen [::]:443 ssl http2;\n")
	sb.WriteString(fmt.Sprintf("    server_name %s;\n\n", domain))

	// SSL configuration
	if len(inbounds) > 0 && inbounds[0].Domain != nil {
		d := inbounds[0].Domain
		sb.WriteString("    # SSL Configuration\n")
		sb.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", d.CertPath))
		sb.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", d.KeyPath))
		sb.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
		sb.WriteString("    ssl_ciphers HIGH:!aNULL:!MD5;\n")
		sb.WriteString("    ssl_prefer_server_ciphers on;\n")
		sb.WriteString("    ssl_session_cache shared:SSL:10m;\n")
		sb.WriteString("    ssl_session_timeout 10m;\n\n")
	}

	// Proxy locations for each inbound
	sb.WriteString("    # Xray Inbound Locations\n")
	for _, i := range inbounds {
		if i.IsGRPC() {
			// gRPC location
			sb.WriteString(fmt.Sprintf("    # gRPC: %s\n", i.Tag))
			sb.WriteString(fmt.Sprintf("    location /%s {\n", i.ServiceName))
			sb.WriteString("        if ($content_type !~ \"application/grpc\") {\n")
			sb.WriteString("            return 404;\n")
			sb.WriteString("        }\n")
			sb.WriteString(fmt.Sprintf("        grpc_pass grpc://127.0.0.1:%d;\n", i.Port))
			sb.WriteString("        grpc_set_header Host $host;\n")
			sb.WriteString("        grpc_set_header X-Real-IP $remote_addr;\n")
			sb.WriteString("        grpc_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
			sb.WriteString("    }\n\n")
		} else if i.IsXHTTP() {
			// XHTTP location
			sb.WriteString(fmt.Sprintf("    # XHTTP: %s\n", i.Tag))
			sb.WriteString(fmt.Sprintf("    location %s {\n", i.Path))
			sb.WriteString("        proxy_redirect off;\n")
			sb.WriteString(fmt.Sprintf("        proxy_pass http://127.0.0.1:%d;\n", i.Port))
			sb.WriteString("        proxy_http_version 1.1;\n")
			sb.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
			sb.WriteString("        proxy_set_header Connection \"upgrade\";\n")
			sb.WriteString("        proxy_set_header Host $host;\n")
			sb.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
			sb.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
			sb.WriteString("    }\n\n")
		} else if i.Transport == models.TransportWS {
			// WebSocket location
			sb.WriteString(fmt.Sprintf("    # WebSocket: %s\n", i.Tag))
			sb.WriteString(fmt.Sprintf("    location %s {\n", i.Path))
			sb.WriteString("        proxy_redirect off;\n")
			sb.WriteString(fmt.Sprintf("        proxy_pass http://127.0.0.1:%d;\n", i.Port))
			sb.WriteString("        proxy_http_version 1.1;\n")
			sb.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
			sb.WriteString("        proxy_set_header Connection \"upgrade\";\n")
			sb.WriteString("        proxy_set_header Host $host;\n")
			sb.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
			sb.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
			sb.WriteString("    }\n\n")
		}
	}

	// Default location (fallback website)
	sb.WriteString("    # Default location (fallback)\n")
	sb.WriteString("    location / {\n")
	sb.WriteString("        root /var/www/html;\n")
	sb.WriteString("        index index.html index.htm;\n")
	sb.WriteString("        try_files $uri $uri/ =404;\n")
	sb.WriteString("    }\n\n")

	// Access and error logs
	sb.WriteString("    # Logging\n")
	sb.WriteString(fmt.Sprintf("    access_log /var/log/nginx/%s.access.log;\n", domain))
	sb.WriteString(fmt.Sprintf("    error_log /var/log/nginx/%s.error.log;\n", domain))
	sb.WriteString("}\n")

	return sb.String()
}

// GenerateStreamConfig is deprecated - no longer needed
// All traffic is handled by Nginx HTTP layer on port 443
func (g *ConfigGenerator) GenerateStreamConfig(inbounds []models.Inbound) error {
	// Remove old stream config if it exists
	filename := filepath.Join(g.streamDir, "xray-sni-routing.conf")
	if _, err := os.Stat(filename); err == nil {
		// File exists, remove it
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("failed to remove old stream config: %w", err)
		}
	}
	return nil
}
