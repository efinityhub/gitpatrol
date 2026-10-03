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
	stop        chan struct{}
	stopOnce    sync.Once
	done        chan struct{}
}

func NewHealthService(db *database.DB, hub *websocket.Hub, syncManager *SyncManager, cfg *config.Config) *HealthService {
	return &HealthService{
		db:          db,
		hub:         hub,
		syncManager: syncManager,
		config:      cfg,
		status:      "healthy",
		checks:      make(map[string]interface{}),
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}
}

func (s *HealthService) Start() {
	slog.Info("Starting background health monitoring")
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			s.PerformCheck()
			s.CleanupLogs()
			select {
			case <-ticker.C:
			case <-s.stop:
				return
			}
		}
	}()
}

func (s *HealthService) Stop() {
	s.stopOnce.Do(func() { close(s.stop) })
	<-s.done
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

const uptimeWindowDays = 30

func (s *HealthService) recordCheck(status string) {
	_, err := s.db.Exec("INSERT INTO health_checks (checked_at, status) VALUES (?, ?)", time.Now().UTC().Format("2006-01-02 15:04:05"), status)
	if err != nil {
		slog.Error("Failed to record health check", "error", err)
	}

	_, err = s.db.Exec("DELETE FROM health_checks WHERE checked_at < datetime('now', ?)", fmt.Sprintf("-%d days", uptimeWindowDays))
	if err != nil {
		slog.Error("Failed to prune health checks", "error", err)
	}
}

func (s *HealthService) uptime() map[string]interface{} {
	var total, healthy int
	err := s.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'healthy' THEN 1 ELSE 0 END), 0) FROM health_checks").Scan(&total, &healthy)
	if err != nil || total == 0 {
		return map[string]interface{}{"percent": nil, "window_days": uptimeWindowDays}
	}
	return map[string]interface{}{
		"percent":     float64(healthy) / float64(total) * 100,
		"window_days": uptimeWindowDays,
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

	// 5. Uptime
	s.recordCheck(newStatus)
	checks["uptime"] = s.uptime()

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
