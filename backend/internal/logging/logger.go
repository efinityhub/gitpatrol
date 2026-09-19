package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"gitpatrol/internal/database"
)

type DBHandler struct {
	slog.Handler
	db          *database.DB
	wsBroadcast func(map[string]interface{})
	logChan     chan logEntry
}

type logEntry struct {
	Level      string
	Message    string
	Attributes string
	Time       string
	WsEntry    map[string]interface{}
}

func NewDBHandler(db *database.DB, wsBroadcast func(map[string]interface{})) *DBHandler {
	// Base handler is a JSON handler writing to stdout
	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handler := &DBHandler{
		Handler:     base,
		db:          db,
		wsBroadcast: wsBroadcast,
		logChan:     make(chan logEntry, 1000), // Buffer for logs
	}
	
	// Start background worker to write logs sequentially
	go handler.processLogs()
	
	return handler
}

func (h *DBHandler) processLogs() {
	for entry := range h.logChan {
		_, dbErr := h.db.Exec(`
			INSERT INTO system_logs (level, message, attributes, created_at)
			VALUES (?, ?, ?, ?)
		`, entry.Level, entry.Message, entry.Attributes, entry.Time)
		
		if dbErr != nil {
			fmt.Fprintf(os.Stderr, "failed to insert log to db: %v\n", dbErr)
		}

		if h.wsBroadcast != nil {
			h.wsBroadcast(entry.WsEntry)
		}
	}
}

func (h *DBHandler) Handle(ctx context.Context, r slog.Record) error {
	// Process standard handling (stdout)
	err := h.Handler.Handle(ctx, r)

	// Process attributes to JSON string
	attrs := make(map[string]interface{})
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	attrJSON, _ := json.Marshal(attrs)

	// Send to channel instead of raw goroutine
	wsEntry := map[string]interface{}{
		"level":      r.Level.String(),
		"message":    r.Message,
		"attributes": attrs,
		"time":       r.Time,
	}

	select {
	case h.logChan <- logEntry{
		Level:      r.Level.String(),
		Message:    r.Message,
		Attributes: string(attrJSON),
		Time:       r.Time.Format("2006-01-02 15:04:05.999999999-07:00"),
		WsEntry:    wsEntry,
	}:
	default:
		// If channel is full, drop log to avoid blocking main execution
		fmt.Fprintf(os.Stderr, "log channel full, dropping log: %s\n", r.Message)
	}

	return err
}

// Setup initializes the global slog logger with the custom DBHandler
func Setup(db *database.DB, wsBroadcast func(map[string]interface{})) {
	handler := NewDBHandler(db, wsBroadcast)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
