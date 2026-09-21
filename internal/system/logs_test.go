package system_test

import (
	"strings"
	"testing"

	"xray-panel/internal/system"
)

func TestGetServiceLogs(t *testing.T) {
	// Test requesting logs with different line counts
	logs, err := system.GetServiceLogs(system.LogSourcePanel, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) == 0 {
		t.Errorf("expected non-empty output or friendly message")
	}

	// Test boundary condition: negative lines fallback
	logsNeg, err := system.GetServiceLogs(system.LogSourceXray, -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logsNeg) == 0 {
		t.Errorf("expected non-empty output")
	}

	// Test wireguard logs
	wgLogs, err := system.GetServiceLogs(system.LogSourceWireguard, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(strings.TrimSpace(wgLogs)) == 0 {
		t.Errorf("expected non-empty output")
	}
}
