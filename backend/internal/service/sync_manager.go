package service

import (
	"context"
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
	wg          sync.WaitGroup
	stop        chan struct{}
	stopOnce    sync.Once
	killCtx     context.Context
	kill        context.CancelFunc
}

func NewSyncManager(workerCount int, db *database.DB, hub *websocket.Hub, repoService *RepoService) *SyncManager {
	killCtx, kill := context.WithCancel(context.Background())
	m := &SyncManager{
		tasks:       make(chan SyncTask, 100),
		activeTasks: make(map[int]bool),
		workerCount: workerCount,
		db:          db,
		hub:         hub,
		repoService: repoService,
		stop:        make(chan struct{}),
		killCtx:     killCtx,
		kill:        kill,
	}

	for i := 0; i < workerCount; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
	slog.Info("Started SyncManager", "workers", workerCount)
	return m
}

func (m *SyncManager) Enqueue(id int, url, name string) {
	select {
	case <-m.stop:
		return
	default:
	}

	m.mu.Lock()
	if m.activeTasks[id] {
		m.mu.Unlock()
		slog.Info("Task already in queue or active, skipping duplicate", "repo", name)
		return
	}
	m.activeTasks[id] = true
	m.mu.Unlock()

	select {
	case m.tasks <- SyncTask{ID: id, URL: url, Name: name}:
		slog.Info("Enqueued sync", "repo", name)
	case <-m.stop:
		m.mu.Lock()
		delete(m.activeTasks, id)
		m.mu.Unlock()
	}
}

func (m *SyncManager) worker(id int) {
	defer m.wg.Done()
	for {
		select {
		case <-m.stop:
			return
		case task := <-m.tasks:
			slog.Info("Starting sync", "worker", id, "repo", task.Name)
			m.repoService.SyncRepo(m.killCtx, task.ID, task.URL, task.Name)
			slog.Info("Finished sync", "worker", id, "repo", task.Name)

			m.mu.Lock()
			delete(m.activeTasks, task.ID)
			m.mu.Unlock()
		}
	}
}

func (m *SyncManager) Stop(ctx context.Context) {
	m.stopOnce.Do(func() { close(m.stop) })

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		slog.Warn("Sync workers did not finish within the shutdown window, cancelling in-flight syncs")
		m.kill()
		<-done
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
