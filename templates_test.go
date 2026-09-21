package xraypanel_test

import (
	"io/fs"
	"testing"

	xraypanel "xray-panel"
	"xray-panel/internal/web"
)

func TestLoadTemplates(t *testing.T) {
	webFS, err := fs.Sub(xraypanel.WebFiles, "web")
	if err != nil {
		t.Fatalf("failed to sub web filesystem: %v", err)
	}

	tmpl, err := web.LoadTemplates(webFS)
	if err != nil {
		t.Fatalf("failed to load templates: %v", err)
	}

	// Verify our templates are loaded
	expectedTemplates := []string{
		"wireguard",
		"wireguard-content",
		"components/wg-peers-table.html",
		"components/wg-peer-form.html",
		"components/wg-server-form.html",
		"components/wg-peer-config-modal.html",
		"logs",
		"logs-content",
		"components/logs-terminal.html",
	}

	for _, name := range expectedTemplates {
		if tmpl.Lookup(name) == nil {
			t.Errorf("expected template %q to be loaded, but it was not found", name)
		}
	}
}
