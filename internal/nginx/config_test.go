package nginx

import (
	"bytes"
	"strings"
	"testing"
)

func TestPanelConfigTmpl(t *testing.T) {
	data := panelTmplData{
		Domain:   "panel.example.com",
		CertPath: "/etc/ssl/panel.crt",
		KeyPath:  "/etc/ssl/panel.key",
		Port:     "8082",
	}

	var buf bytes.Buffer
	err := panelConfigTmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("failed to execute panelConfigTmpl: %v", err)
	}

	output := buf.String()

	// Verify modern HTTP/2 syntax
	if strings.Contains(output, "ssl http2") {
		t.Errorf("expected no deprecated 'ssl http2' in output, but found it")
	}

	if !strings.Contains(output, "listen 443 ssl;") {
		t.Errorf("expected 'listen 443 ssl;' in output")
	}

	if !strings.Contains(output, "listen [::]:443 ssl;") {
		t.Errorf("expected 'listen [::]:443 ssl;' in output")
	}

	if !strings.Contains(output, "http2 on;") {
		t.Errorf("expected 'http2 on;' in output")
	}

	if !strings.Contains(output, "server_name panel.example.com;") {
		t.Errorf("expected server_name in output")
	}

	if !strings.Contains(output, "proxy_pass http://127.0.0.1:8082;") {
		t.Errorf("expected proxy_pass in output")
	}
}

func TestInboundsConfigTmpl(t *testing.T) {
	data := inboundsTmplData{
		Domain:   "node.example.com",
		HasCert:  true,
		CertPath: "/etc/ssl/node.crt",
		KeyPath:  "/etc/ssl/node.key",
		Inbounds: []inboundLocationData{
			{
				Tag:         "vless-grpc",
				IsGRPC:      true,
				ServiceName: "mygrpc",
				Upstream:    "grpc://127.0.0.1:10001",
			},
			{
				Tag:      "vless-ws",
				IsWS:     true,
				Path:     "/ws-path",
				Upstream: "http://127.0.0.1:10002",
			},
		},
	}

	var buf bytes.Buffer
	err := inboundsConfigTmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("failed to execute inboundsConfigTmpl: %v", err)
	}

	output := buf.String()

	// Verify modern HTTP/2 syntax
	if strings.Contains(output, "ssl http2") {
		t.Errorf("expected no deprecated 'ssl http2' in output, but found it")
	}

	if !strings.Contains(output, "listen 443 ssl;") {
		t.Errorf("expected 'listen 443 ssl;' in output")
	}

	if !strings.Contains(output, "listen [::]:443 ssl;") {
		t.Errorf("expected 'listen [::]:443 ssl;' in output")
	}

	if !strings.Contains(output, "http2 on;") {
		t.Errorf("expected 'http2 on;' in output")
	}

	if !strings.Contains(output, "location /mygrpc") {
		t.Errorf("expected grpc location")
	}

	if !strings.Contains(output, "location /ws-path") {
		t.Errorf("expected ws location")
	}
}
