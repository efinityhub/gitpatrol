package service

import (
	"log/slog"
	"sync"

	"gitpatrol/internal/database"
	"gitpatrol/internal/websocket"
)

type SyncTask struct {
	ID   int
	URL  string
	Name string
}

type SyncManager struct {
	tasks       chan SyncTask
	activeTasks map[int]bool
	mu          sync.Mutex
	workerCount int
	db          *database.DB
	hub         *websocket.Hub
	repoService *RepoService
}

func NewSyncManager(workerCount int, db *database.DB, hub *websocket.Hub, repoService *RepoService) *SyncManager {
	m := &SyncManager{
		tasks:       make(chan SyncTask, 100),
		activeTasks: make(map[int]bool),
		workerCount: workerCount,
		db:          db,
		hub:         hub,
		repoService: repoService,
	}

	for i := 0; i < workerCount; i++ {
		go m.worker(i)
	}
	slog.Info("Started SyncManager", "workers", workerCount)
	return m
}

func (m *SyncManager) Enqueue(id int, url, name string) {
	m.mu.Lock()
	if m.activeTasks[id] {
		m.mu.Unlock()
		slog.Info("Task already in queue or active, skipping duplicate", "repo", name)
		return
	}
	m.activeTasks[id] = true
	m.mu.Unlock()

	m.tasks <- SyncTask{ID: id, URL: url, Name: name}
	slog.Info("Enqueued sync", "repo", name)
}

func (m *SyncManager) worker(id int) {
	for task := range m.tasks {
		slog.Info("Starting sync", "worker", id, "repo", task.Name)
		m.repoService.SyncRepo(task.ID, task.URL, task.Name)
		slog.Info("Finished sync", "worker", id, "repo", task.Name)

		m.mu.Lock()
		delete(m.activeTasks, task.ID)
		m.mu.Unlock()
	}
}

func (m *SyncManager) GetStats() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]interface{}{
		"active_tasks":  len(m.activeTasks),
		"queued_tasks":  len(m.tasks),
		"total_workers": m.workerCount,
	}
}
