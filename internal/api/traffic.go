package api

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"xray-panel/internal/logger"
	"xray-panel/internal/models"
	"xray-panel/internal/system"
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
		logger.Debug("Traffic sync worker started (interval: %v)", interval)

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
			"user>>>"+user.StatsKey()+">>>traffic>>>downlink", true,
		)
		// Query uplink traffic (reset after read)
		up, _ := client.GetStats(
			"user>>>"+user.StatsKey()+">>>traffic>>>uplink", true,
		)

		total := down + up
		if total <= 0 {
			continue
		}

		// Atomic increment traffic in DB (avoids race condition)
		if err := s.db.Model(&models.User{}).
			Where("id = ?", user.ID).
			Update("traffic_used", gorm.Expr("traffic_used + ?", total)).Error; err != nil {
			logger.Error("Traffic sync: failed to update user %s: %v", user.StatsKey(), err)
			continue
		}
		updated++
	}

	if updated > 0 {
		logger.Debug("Traffic sync: updated %d users", updated)
	}
}

// startMonthlyTrafficReset starts a background goroutine that resets user traffic
// and snapshots the system network baseline at the start of each calendar month.
func (s *Server) startMonthlyTrafficReset() {
	go func() {
		// Check once at startup (handles the case where the panel was down during month rollover)
		s.runMonthlyResetIfNeeded()

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			s.runMonthlyResetIfNeeded()
		}
	}()
}

// runMonthlyResetIfNeeded checks whether a new calendar month has begun and, if so:
//  1. Resets traffic_used to 0 for all users (sets traffic_reset = now).
//  2. Snapshots the current OS network counters as the new monthly baseline.
func (s *Server) runMonthlyResetIfNeeded() {
	now := time.Now()
	currentMonth := now.Format("2006-01")

	// Read the stored month key
	var stored models.Setting
	if err := s.db.First(&stored, "key = ?", "monthly_net_baseline_month").Error; err != nil {
		// Setting not yet seeded — seed it now with current counters as baseline
		s.snapshotNetBaseline(now, currentMonth)
		return
	}

	if stored.Value == currentMonth {
		return // Already in the correct month, nothing to do
	}

	// ── New month detected ─────────────────────────────────────────────────────

	logger.Info("Monthly reset: new month %s (was %s), resetting traffic", currentMonth, stored.Value)

	// 1. Reset all user traffic
	s.resetAllUserTraffic(now)

	// 2. Snapshot new OS network baseline
	s.snapshotNetBaseline(now, currentMonth)
}

// resetAllUserTraffic zeroes traffic_used and updates traffic_reset for every user.
func (s *Server) resetAllUserTraffic(now time.Time) {
	result := s.db.Model(&models.User{}).
		Where("1 = 1"). // explicit condition so GORM issues an UPDATE … WHERE, not a no-op
		Updates(map[string]interface{}{
			"traffic_used":  int64(0),
			"traffic_reset": now,
		})
	if result.Error != nil {
		logger.Error("Monthly reset: failed to reset user traffic: %v", result.Error)
		return
	}
	logger.Info("Monthly reset: reset traffic for %d users", result.RowsAffected)
}

// snapshotNetBaseline reads the current OS network counters and persists them
// as the baseline for the given month (format "YYYY-MM").
func (s *Server) snapshotNetBaseline(now time.Time, month string) {
	sysInfo, err := system.GetSystemInfo()
	if err != nil {
		logger.Error("Monthly reset: failed to read system info for baseline: %v", err)
		return
	}

	upsert := func(key, value string) {
		setting := models.Setting{Key: key, Value: value}
		if err := s.db.Where("key = ?", key).
			Assign(models.Setting{Value: value}).
			FirstOrCreate(&setting).Error; err != nil {
			// FirstOrCreate succeeded (row exists) — do an explicit update
			s.db.Model(&models.Setting{}).
				Where("key = ?", key).
				Update("value", value)
		}
	}

	upsert("monthly_net_baseline_sent", fmt.Sprintf("%d", sysInfo.NetBytesSent))
	upsert("monthly_net_baseline_recv", fmt.Sprintf("%d", sysInfo.NetBytesRecv))
	upsert("monthly_net_baseline_month", month)

	logger.Info("Monthly reset: net baseline snapshot for %s — sent=%d recv=%d",
		month, sysInfo.NetBytesSent, sysInfo.NetBytesRecv)
}
