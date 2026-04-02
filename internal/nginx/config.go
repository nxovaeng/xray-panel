package nginx

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"xray-panel/internal/models"

	"gorm.io/gorm"
)

const (
	ManagedHeader = "# Managed by Xray Panel"
)

// ConfigGenerator handles generating Nginx configurations
type ConfigGenerator struct {
	configDir string
	streamDir string
	reloadCmd string
	db        *gorm.DB
	socketDir string
}

// NewGenerator creates a new Nginx config generator
func NewGenerator(configDir, streamDir string) *ConfigGenerator {
	return &ConfigGenerator{
		configDir: configDir,
		streamDir: streamDir,
		reloadCmd: "systemctl reload nginx",
		socketDir: "/dev/shm",
	}
}

// SetDB sets the database connection for tracking configs
func (g *ConfigGenerator) SetDB(db *gorm.DB) {
	g.db = db
}

// SetReloadCmd sets custom reload command
func (g *ConfigGenerator) SetReloadCmd(cmd string) {
	g.reloadCmd = cmd
}

// SetSocketDir sets the Unix Domain Socket directory
func (g *ConfigGenerator) SetSocketDir(dir string) {
	if dir != "" {
		g.socketDir = dir
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

// recordConfig records a generated Nginx config in database
func (g *ConfigGenerator) recordConfig(inboundID, domain, configPath, configType string) error {
	if g.db == nil {
		return nil // Skip if no database connection
	}

	// Check if config already exists
	var existing models.NginxConfig
	result := g.db.Where("inbound_id = ? AND config_path = ?", inboundID, configPath).First(&existing)

	if result.Error == nil {
		// Update existing record
		existing.Domain = domain
		existing.ConfigType = configType
		return g.db.Save(&existing).Error
	}

	// Create new record
	config := &models.NginxConfig{
		InboundID:  inboundID,
		Domain:     domain,
		ConfigPath: configPath,
		ConfigType: configType,
		IsManaged:  true,
	}
	return g.db.Create(config).Error
}

// CleanupInboundConfigs removes Nginx configs for a specific inbound
func (g *ConfigGenerator) CleanupInboundConfigs(inboundID string) error {
	if g.db == nil {
		return nil
	}

	// Find all configs for this inbound
	var configs []models.NginxConfig
	if err := g.db.Where("inbound_id = ?", inboundID).Find(&configs).Error; err != nil {
		return err
	}

	// Delete config files
	for _, config := range configs {
		if config.IsManaged {
			// Check if file has managed header before deleting
			if g.isManagedFile(config.ConfigPath) {
				os.Remove(config.ConfigPath)
			}
		}
	}

	// Delete database records
	return g.db.Where("inbound_id = ?", inboundID).Delete(&models.NginxConfig{}).Error
}

// isManagedFile checks if a file is managed by the panel
func (g *ConfigGenerator) isManagedFile(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		firstLine := scanner.Text()
		return strings.Contains(firstLine, ManagedHeader)
	}
	return false
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

	data := panelTmplData{
		Domain:   domain,
		CertPath: certPath,
		KeyPath:  keyPath,
		Port:     port,
	}

	var buf bytes.Buffer
	if err := panelConfigTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute panel template: %w", err)
	}

	filename := filepath.Join(g.configDir, fmt.Sprintf("%s.conf", domain))
	return g.writeConfig(filename, buf.String())
}

// GenerateHTTPConfig generates Nginx HTTP server blocks for inbounds
func (g *ConfigGenerator) GenerateHTTPConfig(inbounds []models.Inbound) error {
	// Group inbounds by actual domain (considering wildcard subdomains)
	domainInbounds := make(map[string][]models.Inbound)
	for _, i := range inbounds {
		if i.Domain != nil {
			// Use ActualDomain if set (for wildcard certs), otherwise use Domain.Domain
			domain := i.Domain.Domain
			if i.ActualDomain != "" {
				domain = i.ActualDomain
			}
			domainInbounds[domain] = append(domainInbounds[domain], i)
		}
	}

	// Generate config for each domain
	for domain, inbounds := range domainInbounds {
		conf := g.buildServerBlock(domain, inbounds)
		filename := filepath.Join(g.configDir, fmt.Sprintf("%s.conf", domain))

		if err := g.writeConfig(filename, conf); err != nil {
			return err
		}

		// Record each inbound's config
		for _, inbound := range inbounds {
			if err := g.recordConfig(inbound.ID, domain, filename, "http"); err != nil {
				// Log error but don't fail
				fmt.Printf("Warning: failed to record config for inbound %s: %v\n", inbound.ID, err)
			}
		}
	}

	return nil
}

func (g *ConfigGenerator) buildServerBlock(domain string, inbounds []models.Inbound) string {
	data := inboundsTmplData{
		Domain: domain,
	}

	if len(inbounds) > 0 && inbounds[0].Domain != nil {
		d := inbounds[0].Domain
		data.HasCert = true
		data.CertPath = d.CertPath
		data.KeyPath = d.KeyPath
	}

	var locations []inboundLocationData
	for _, i := range inbounds {
		// Determine upstream address
		var grpcUpstream, httpUpstream string
		if i.UseUDS {
			sockPath := i.SocketPath(g.socketDir)
			grpcUpstream = fmt.Sprintf("grpc://unix:%s", sockPath)
			httpUpstream = fmt.Sprintf("http://unix:%s", sockPath)
		} else {
			grpcUpstream = fmt.Sprintf("grpc://127.0.0.1:%d", i.Port)
			httpUpstream = fmt.Sprintf("http://127.0.0.1:%d", i.Port)
		}

		loc := inboundLocationData{
			Tag:          i.Tag,
			ActualDomain: i.ActualDomain,
			IsGRPC:       i.IsGRPC(),
			IsXHTTP:      i.IsXHTTP(),
			IsWS:         i.Transport == models.TransportWS,
			ServiceName:  i.ServiceName,
			Path:         i.Path,
		}

		if loc.IsGRPC {
			loc.Upstream = grpcUpstream
		} else if loc.IsXHTTP || loc.IsWS {
			loc.Upstream = httpUpstream
		}
		locations = append(locations, loc)
	}
	data.Inbounds = locations

	var buf bytes.Buffer
	if err := inboundsConfigTmpl.Execute(&buf, data); err != nil {
		// As this shouldn't fail locally, returning empty string implies it failed
		return ""
	}

	return buf.String()
}
