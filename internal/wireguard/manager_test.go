package wireguard_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"xray-panel/internal/models"
	"xray-panel/internal/wireguard"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite in memory: %v", err)
	}

	err = db.AutoMigrate(
		&models.Inbound{},
		&models.WGServerConfig{},
		&models.WGPeer{},
	)
	if err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	return db
}

func TestWireGuardKeyGeneration(t *testing.T) {
	privKey, pubKey, err := models.GenerateWGKeyPair()
	if err != nil {
		t.Fatalf("GenerateWGKeyPair failed: %v", err)
	}
	if len(privKey) == 0 || len(pubKey) == 0 {
		t.Fatalf("empty keys generated: priv=%q, pub=%q", privKey, pubKey)
	}

	derivedPub, err := models.DeriveWGPublicKey(privKey)
	if err != nil {
		t.Fatalf("DeriveWGPublicKey failed: %v", err)
	}
	if derivedPub != pubKey {
		t.Errorf("derived public key %q does not match generated public key %q", derivedPub, pubKey)
	}
}

func TestIPAllocation(t *testing.T) {
	db := setupTestDB(t)

	serverAddr := "10.0.0.1/24"
	ip1 := models.GetNextAvailableIP(db, serverAddr)
	if ip1 != "10.0.0.2/32" {
		t.Errorf("expected 10.0.0.2/32, got %s", ip1)
	}

	// Insert peer 1
	db.Create(&models.WGPeer{
		Name:       "Peer 1",
		PublicKey:  "pub1",
		AllowedIPs: "10.0.0.2/32",
		Enabled:    true,
	})

	ip2 := models.GetNextAvailableIP(db, serverAddr)
	if ip2 != "10.0.0.3/32" {
		t.Errorf("expected 10.0.0.3/32, got %s", ip2)
	}

	// Insert peer 2
	db.Create(&models.WGPeer{
		Name:       "Peer 2",
		PublicKey:  "pub2",
		AllowedIPs: "10.0.0.3/32",
		Enabled:    true,
	})

	ip3 := models.GetNextAvailableIP(db, serverAddr)
	if ip3 != "10.0.0.4/32" {
		t.Errorf("expected 10.0.0.4/32, got %s", ip3)
	}
}

func TestPortConflictDetection(t *testing.T) {
	db := setupTestDB(t)

	// No conflict initially
	conflict, _ := wireguard.CheckPortConflict(db, 51820)
	if conflict {
		t.Errorf("expected no conflict, got true")
	}

	// Create enabled inbound with port 51820
	db.Create(&models.Inbound{
		ID:        "inbound-1",
		Tag:       "vless-in",
		Protocol:  models.ProtocolVLESS,
		Transport: models.TransportWS,
		Port:      51820,
		Enabled:   true,
	})

	conflict, tag := wireguard.CheckPortConflict(db, 51820)
	if !conflict || tag != "vless-in" {
		t.Errorf("expected conflict with vless-in, got conflict=%v tag=%s", conflict, tag)
	}

	// If disabled, should not conflict
	db.Model(&models.Inbound{}).Where("id = ?", "inbound-1").Update("enabled", false)
	conflict, _ = wireguard.CheckPortConflict(db, 51820)
	if conflict {
		t.Errorf("expected no conflict when inbound is disabled, got true")
	}
}

func TestConfigGeneration(t *testing.T) {
	db := setupTestDB(t)

	serverCfg := &models.WGServerConfig{
		InterfaceName: "wg0",
		PrivateKey:    "server_priv_key",
		PublicKey:     "server_pub_key",
		ListenPort:    51820,
		Address:       "10.0.0.1/24",
		MTU:           1420,
		PostUp:        models.DefaultPostUp("%i"),
		PostDown:      models.DefaultPostDown("%i"),
		Enabled:       true,
	}

	peers := []models.WGPeer{
		{
			ID:                  1,
			Name:                "Client-Tokyo",
			PublicKey:           "tokyo_pub_key",
			PrivateKey:          "tokyo_priv_key",
			AllowedIPs:          "10.0.0.2/32",
			PersistentKeepalive: 25,
			Enabled:             true,
		},
		{
			ID:         2,
			Name:       "Client-Disabled",
			PublicKey:  "disabled_pub_key",
			AllowedIPs: "10.0.0.3/32",
			Enabled:    false, // should be excluded from wg0.conf
		},
	}

	mgr := wireguard.NewManager(db)
	conf := mgr.GenerateWG0Conf(serverCfg, peers)

	if !strings.Contains(conf, "ListenPort = 51820") {
		t.Errorf("expected ListenPort = 51820 in conf, got:\n%s", conf)
	}
	if !strings.Contains(conf, "Address = 10.0.0.1/24") {
		t.Errorf("expected Address in conf")
	}
	if !strings.Contains(conf, "PublicKey = tokyo_pub_key") {
		t.Errorf("expected tokyo_pub_key in conf")
	}
	if strings.Contains(conf, "disabled_pub_key") {
		t.Errorf("disabled peer should not be in wg0.conf")
	}
	// EnableNAT is false: PostUp and PostDown must NOT be written
	if strings.Contains(conf, "PostUp") || strings.Contains(conf, "PostDown") {
		t.Errorf("expected no PostUp/PostDown when EnableNAT is false, got:\n%s", conf)
	}

	// EnableNAT is true: PostUp and PostDown must be written
	serverCfg.EnableNAT = true
	confWithNAT := mgr.GenerateWG0Conf(serverCfg, peers)
	if !strings.Contains(confWithNAT, "PostUp") || !strings.Contains(confWithNAT, "PostDown") {
		t.Errorf("expected PostUp/PostDown when EnableNAT is true, got:\n%s", confWithNAT)
	}

	// Test client configs
	clientWG := mgr.GenerateClientWGConfig(serverCfg, &peers[0], "1.2.3.4")
	if !strings.Contains(clientWG, "Endpoint = 1.2.3.4:51820") {
		t.Errorf("expected Endpoint = 1.2.3.4:51820 in client config, got:\n%s", clientWG)
	}
	if !strings.Contains(clientWG, "PrivateKey = tokyo_priv_key") {
		t.Errorf("expected client private key in client config")
	}

	clientXrayJSON, err := mgr.GenerateClientXrayJSON(serverCfg, &peers[0], "1.2.3.4")
	if err != nil {
		t.Fatalf("GenerateClientXrayJSON failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(clientXrayJSON), &parsed); err != nil {
		t.Fatalf("invalid json generated: %v", err)
	}
	if parsed["protocol"] != "wireguard" {
		t.Errorf("expected protocol wireguard, got %v", parsed["protocol"])
	}
}
