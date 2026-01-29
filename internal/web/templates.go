package web

import (
	"fmt"
	"html/template"
	"io/fs"
)

// LoadTemplates loads all HTML templates
func LoadTemplates(templateFS fs.FS) (*template.Template, error) {
	// Define helper functions
	funcMap := template.FuncMap{
		"formatBytes": func(bytes int64) string {
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
		"calculatePercentage": func(used, total int64) int {
			if total == 0 {
				return 0
			}
			p := int(float64(used) / float64(total) * 100)
			if p > 100 {
				return 100
			}
			return p
		},
		"bytesToGB": func(bytes int64) int64 {
			const gb = 1024 * 1024 * 1024
			return bytes / gb
		},
		"formatDate": func(t interface{}) string {
			// Handle time.Time type
			if tt, ok := t.(interface{ Format(string) string }); ok {
				return tt.Format("2006-01-02")
			}
			return ""
		},
	}

	tmpl := template.New("").Funcs(funcMap)

	// Load nav
	if err := loadTemplate(tmpl, templateFS, "templates/nav.html"); err != nil {
		return nil, err
	}

	// Load login
	if err := loadTemplate(tmpl, templateFS, "templates/login.html"); err != nil {
		return nil, err
	}

	// Load pages (each page is now a complete template with nav)
	pages := []string{
		"templates/pages/dashboard.html",
		"templates/pages/users.html",
		"templates/pages/inbounds.html",
		"templates/pages/outbounds.html",
		"templates/pages/routing.html",
		"templates/pages/domains.html",
		"templates/pages/settings.html",
	}
	for _, page := range pages {
		if err := loadTemplate(tmpl, templateFS, page); err != nil {
			return nil, err
		}
	}

	// Load components
	components := []string{
		"templates/components/users-table.html",
		"templates/components/user-form.html",
		"templates/components/inbounds-table.html",
		"templates/components/inbound-form.html",
		"templates/components/domains-table.html",
		"templates/components/domain-form.html",
		"templates/components/dashboard-stats.html",
		"templates/components/outbounds-table.html",
		"templates/components/outbound-form.html",
		"templates/components/routing-table.html",
		"templates/components/routing-form.html",
	}
	for _, comp := range components {
		if err := loadTemplate(tmpl, templateFS, comp); err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}

func loadTemplate(tmpl *template.Template, templateFS fs.FS, filePath string) error {
	content, err := fs.ReadFile(templateFS, filePath)
	if err != nil {
		return err
	}

	// 直接解析，让 {{define}} 自动创建模板
	// 不使用 tmpl.New(name)，避免创建冗余的空模板
	_, err = tmpl.Parse(string(content))
	return err
}
