package api

import (
	"time"

	"gorm.io/gorm"

	"xray-panel/internal/logger"
	"xray-panel/internal/models"
	"xray-panel/internal/xray"
)

// startTrafficSync starts a background goroutine that periodically syncs
// traffic statistics from Xray API to the database.
func (s *Server) startTrafficSync() {
	interval := 60 * time.Second
	apiClient := xray.NewAPIClientWithBinary(
		"127.0.0.1",
		s.config.Xray.APIPort,
		s.config.Xray.BinaryPath,
	)

	go func() {
		// Wait a bit for Xray to start
		time.Sleep(10 * time.Second)
		logger.Info("Traffic sync worker started (interval: %v)", interval)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.syncTraffic(apiClient)
		}
	}()
}

// syncTraffic pulls traffic statistics from Xray and updates users in the database.
func (s *Server) syncTraffic(client *xray.APIClient) {
	if !client.IsHealthy() {
		return // Xray not running, skip
	}

	var users []models.User
	if err := s.db.Where("enabled = ?", true).Find(&users).Error; err != nil {
		logger.Error("Traffic sync: failed to fetch users: %v", err)
		return
	}

	updated := 0
	for _, user := range users {
		// Query downlink traffic (reset after read)
		down, _ := client.GetStats(
			"user>>>"+user.Email+">>>traffic>>>downlink", true,
		)
		// Query uplink traffic (reset after read)
		up, _ := client.GetStats(
			"user>>>"+user.Email+">>>traffic>>>uplink", true,
		)

		total := down + up
		if total <= 0 {
			continue
		}

		// Atomic increment traffic in DB (avoids race condition)
		if err := s.db.Model(&models.User{}).
			Where("id = ?", user.ID).
			Update("traffic_used", gorm.Expr("traffic_used + ?", total)).Error; err != nil {
			logger.Error("Traffic sync: failed to update user %s: %v", user.Email, err)
			continue
		}
		updated++
	}

	if updated > 0 {
		logger.Info("Traffic sync: updated %d users", updated)
	}
}
