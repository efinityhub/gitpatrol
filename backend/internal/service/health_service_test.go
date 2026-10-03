package service

import (
	"path/filepath"
	"testing"
	"time"

	"gitpatrol/internal/database"
)

func newTestHealthService(t *testing.T) (*HealthService, *database.DB) {
	t.Helper()
	db, err := database.NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &HealthService{db: db}, db
}

func TestUptimeWithoutChecks(t *testing.T) {
	s, _ := newTestHealthService(t)
	got := s.uptime()
	if got["percent"] != nil {
		t.Errorf("percent = %v, want nil before the first check", got["percent"])
	}
	if got["window_days"] != uptimeWindowDays {
		t.Errorf("window_days = %v, want %d", got["window_days"], uptimeWindowDays)
	}
}

func TestUptimePercent(t *testing.T) {
	s, _ := newTestHealthService(t)
	for _, status := range []string{"healthy", "healthy", "healthy", "degraded"} {
		s.recordCheck(status)
	}
	got, ok := s.uptime()["percent"].(float64)
	if !ok {
		t.Fatalf("percent has type %T, want float64", s.uptime()["percent"])
	}
	if got != 75 {
		t.Errorf("percent = %v, want 75", got)
	}
}

func TestRecordCheckPrunesOldRows(t *testing.T) {
	s, db := newTestHealthService(t)
	old := time.Now().UTC().AddDate(0, 0, -(uptimeWindowDays + 1)).Format("2006-01-02 15:04:05")
	if _, err := db.Exec("INSERT INTO health_checks (checked_at, status) VALUES (?, 'degraded')", old); err != nil {
		t.Fatal(err)
	}

	s.recordCheck("healthy")

	var total int
	db.QueryRow("SELECT COUNT(*) FROM health_checks").Scan(&total)
	if total != 1 {
		t.Errorf("rows after prune = %d, want 1", total)
	}
	if got := s.uptime()["percent"]; got != float64(100) {
		t.Errorf("percent = %v, want 100 once the old failure is pruned", got)
	}
}
