package nginx

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"xray-panel/internal/models"
)

const (
	ManagedHeader = "# Managed by Xray Panel"
)

// ConfigGenerator handles generating Nginx configurations
type ConfigGenerator struct {
	configDir string
	reloadCmd string
}

// NewGenerator creates a new Nginx config generator
func NewGenerator(configDir, reloadCmd string) *ConfigGenerator {
	return &ConfigGenerator{
		configDir: configDir,
		reloadCmd: reloadCmd,
	}
}

// Reload reloads Nginx configuration
func (g *ConfigGenerator) Reload() error {
	parts := strings.Fields(g.reloadCmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty reload command")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	return cmd.Run()
}

// writeConfig safely writes Nginx configuration
func (g *ConfigGenerator) writeConfig(filename string, content string) error {
	// Check if file exists
	if _, err := os.Stat(filename); err == nil {
		// File exists, check for managed header
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		isManaged := false
		if scanner.Scan() {
			firstLine := scanner.Text()
			if strings.Contains(firstLine, ManagedHeader) {
				isManaged = true
			}
		}

		if !isManaged {
			return fmt.Errorf("file %s exists and is not managed by Xray Panel", filename)
		}
	}

	// Add header
	fullContent := fmt.Sprintf("%s\n%s", ManagedHeader, content)
	return os.WriteFile(filename, []byte(fullContent), 0644)
}

// GeneratePanelConfig generates Nginx config for the panel itself
func (g *ConfigGenerator) GeneratePanelConfig(domain, certPath, keyPath, listenAddr string) error {
	// Extract port from listenAddr (e.g., ":8082" -> "8082")
	parts := strings.Split(listenAddr, ":")
	port := parts[len(parts)-1]

	var sb strings.Builder

	// HTTP to HTTPS redirect
	sb.WriteString("server {\n")
	sb.WriteString("    listen 80;\n")
	sb.WriteString("    listen [::]:80;\n")
	sb.WriteString(fmt.Sprintf("    server_name %s;\n", domain))
	sb.WriteString("    return 301 https://$host$request_uri;\n")
	sb.WriteString("}\n\n")

	// HTTPS server block
	sb.WriteString("server {\n")
	sb.WriteString("    listen 443 ssl http2;\n")
	sb.WriteString("    listen [::]:443 ssl http2;\n")
	sb.WriteString(fmt.Sprintf("    server_name %s;\n\n", domain))

	// SSL Configuration
	sb.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", certPath))
	sb.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", keyPath))
	sb.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
	sb.WriteString("    ssl_ciphers HIGH:!aNULL:!MD5;\n")
	sb.WriteString("    ssl_prefer_server_ciphers on;\n")
	sb.WriteString("    ssl_session_cache shared:SSL:10m;\n")
	sb.WriteString("    ssl_session_timeout 10m;\n\n")

	// Security Headers
	sb.WriteString("    add_header Strict-Transport-Security \"max-age=31536000; includeSubDomains\" always;\n")
	sb.WriteString("    add_header X-Frame-Options \"SAMEORIGIN\" always;\n")
	sb.WriteString("    add_header X-Content-Type-Options \"nosniff\" always;\n")
	sb.WriteString("    add_header X-XSS-Protection \"1; mode=block\" always;\n\n")

	// Proxy location
	sb.WriteString("    location / {\n")
	sb.WriteString(fmt.Sprintf("        proxy_pass http://127.0.0.1:%s;\n", port))
	sb.WriteString("        proxy_set_header Host $host;\n")
	sb.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	sb.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	sb.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
	sb.WriteString("        \n")
	sb.WriteString("        # WebSocket support\n")
	sb.WriteString("        proxy_http_version 1.1;\n")
	sb.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
	sb.WriteString("        proxy_set_header Connection \"upgrade\";\n")
	sb.WriteString("    }\n")
	sb.WriteString("}\n")

	filename := filepath.Join(g.configDir, fmt.Sprintf("%s.conf", domain))
	return g.writeConfig(filename, sb.String())
}

// GenerateHTTPConfig generates Nginx HTTP server blocks for inbounds
func (g *ConfigGenerator) GenerateHTTPConfig(inbounds []models.Inbound) error {
	// Group inbounds by domain
	domainInbounds := make(map[string][]models.Inbound)
	for _, i := range inbounds {
		if i.Domain != nil {
			domainInbounds[i.Domain.Domain] = append(domainInbounds[i.Domain.Domain], i)
		}
	}

	// Generate config for each domain
	for domain, inbounds := range domainInbounds {
		conf := g.buildServerBlock(domain, inbounds)
		filename := filepath.Join(g.configDir, fmt.Sprintf("%s.conf", domain))
		if err := g.writeConfig(filename, conf); err != nil {
			return err
		}
	}

	return nil
}

func (g *ConfigGenerator) buildServerBlock(domain string, inbounds []models.Inbound) string {
	var sb strings.Builder
	
	// HTTP to HTTPS redirect
	sb.WriteString("server {\n")
	sb.WriteString("    listen 80;\n")
	sb.WriteString("    listen [::]:80;\n")
	sb.WriteString(fmt.Sprintf("    server_name %s;\n", domain))
	sb.WriteString("    return 301 https://$host$request_uri;\n")
	sb.WriteString("}\n\n")

	// HTTPS server block
	sb.WriteString("server {\n")
	sb.WriteString("    listen 443 ssl http2;\n")
	sb.WriteString("    listen [::]:443 ssl http2;\n")
	sb.WriteString(fmt.Sprintf("    server_name %s;\n\n", domain))

	// SSL Configuration
	if len(inbounds) > 0 && inbounds[0].Domain != nil {
		d := inbounds[0].Domain
		sb.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", d.CertPath))
		sb.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", d.KeyPath))
		sb.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
		sb.WriteString("    ssl_ciphers HIGH:!aNULL:!MD5;\n")
		sb.WriteString("    ssl_prefer_server_ciphers on;\n")
		sb.WriteString("    ssl_session_cache shared:SSL:10m;\n")
		sb.WriteString("    ssl_session_timeout 10m;\n\n")
	}

	// Proxy locations for each inbound
	for _, i := range inbounds {
		// ... (keep existing location generation logic)
		if i.IsGRPC() {
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

	// Default location
	sb.WriteString("    location / {\n")
	sb.WriteString("        root /var/www/html;\n")
	sb.WriteString("        index index.html;\n")
	sb.WriteString("        try_files $uri $uri/ =404;\n")
	sb.WriteString("    }\n")
	sb.WriteString("}\n")

	return sb.String()
}
