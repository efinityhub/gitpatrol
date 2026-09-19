package service

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"gitpatrol/internal/config"
	"gitpatrol/internal/database"
	"gitpatrol/internal/websocket"
)

type HealthService struct {
	db          *database.DB
	hub         *websocket.Hub
	syncManager *SyncManager
	config      *config.Config
	status      string
	checks      map[string]interface{}
	mu          sync.RWMutex
}

func NewHealthService(db *database.DB, hub *websocket.Hub, syncManager *SyncManager, cfg *config.Config) *HealthService {
	return &HealthService{
		db:          db,
		hub:         hub,
		syncManager: syncManager,
		config:      cfg,
		status:      "healthy",
		checks:      make(map[string]interface{}),
	}
}

func (s *HealthService) Start() {
	slog.Info("Starting background health monitoring")
	go func() {
		for {
			s.PerformCheck()
			s.CleanupLogs()
			time.Sleep(1 * time.Minute)
		}
	}()
}

func (s *HealthService) CleanupLogs() {
	// Delete logs older than X days
	queryDays := fmt.Sprintf(`DELETE FROM system_logs WHERE created_at < datetime('now', '-%d days')`, s.config.LogRetentionDays)
	_, err := s.db.Exec(queryDays)
	if err != nil {
		slog.Error("Failed to cleanup old logs", "error", err)
	}
	
	// Keep only the most recent Y entries
	queryRows := fmt.Sprintf(`
		DELETE FROM system_logs 
		WHERE id NOT IN (
			SELECT id FROM system_logs ORDER BY id DESC LIMIT %d
		)
	`, s.config.LogMaxRows)
	_, err = s.db.Exec(queryRows)
	if err != nil {
		slog.Error("Failed to trim log limits", "error", err)
	}
}

func (s *HealthService) PerformCheck() {
	newStatus := "healthy"
	checks := make(map[string]interface{})

	// 1. Internet Check
	internet := true
	start := time.Now()
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
	if err != nil {
		internet = false
		newStatus = "degraded"
	} else {
		conn.Close()
	}
	checks["internet"] = map[string]interface{}{
		"connected": internet,
		"latency":   time.Since(start).String(),
	}

	// 2. Disk Space Check
	var stat syscall.Statfs_t
	wd, _ := os.Getwd()
	err = syscall.Statfs(wd, &stat)
	if err == nil {
		free := stat.Bavail * uint64(stat.Bsize)
		total := stat.Blocks * uint64(stat.Bsize)
		used := total - free
		percent := float64(used) / float64(total) * 100

		checks["disk"] = map[string]interface{}{
			"free_bytes":   free,
			"total_bytes":  total,
			"used_percent": fmt.Sprintf("%.1f%%", percent),
		}

		if percent > 90 {
			newStatus = "critical"
		}
	}

	// 3. Database Check
	dbStatus := true
	if err := s.db.Ping(); err != nil {
		dbStatus = false
		newStatus = "critical"
	}
	checks["database"] = dbStatus

	// 4. Worker Pool
	checks["workers"] = s.syncManager.GetStats()

	s.mu.Lock()
	oldStatus := s.status
	s.status = newStatus
	s.checks = checks
	s.mu.Unlock()

	// Handle status changes or periodic heartbeat
	if newStatus != oldStatus {
		slog.Info("System status changed", "old", oldStatus, "new", newStatus)
		
		// Log incident if not healthy
		if newStatus != "healthy" {
			message := fmt.Sprintf("System health degraded to %s. Internet: %v, Disk: %s", 
				newStatus, internet, checks["disk"].(map[string]interface{})["used_percent"])
			
			s.db.Exec("INSERT INTO incidents (repo_id, repo_name, message, created_at) VALUES (?, ?, ?, ?)", 
				nil, "SYSTEM", message, time.Now())
		} else {
			s.db.Exec("INSERT INTO incidents (repo_id, repo_name, message, created_at, resolved) VALUES (?, ?, ?, ?, ?)", 
				nil, "SYSTEM", "System recovered to healthy state.", time.Now(), 1)
		}
	}

	// Always broadcast current health as a heartbeat to keep UI timers fresh
	s.hub.BroadcastHealthStatus(map[string]interface{}{
		"status":    newStatus,
		"checks":    checks,
		"timestamp": time.Now(),
	})
}

func (s *HealthService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]interface{}{
		"status":    s.status,
		"checks":    s.checks,
		"timestamp": time.Now(),
	}
}
