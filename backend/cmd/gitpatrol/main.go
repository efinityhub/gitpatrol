package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gitpatrol/internal/api"
	"gitpatrol/internal/auth"
	"gitpatrol/internal/config"
	"gitpatrol/internal/database"
	"gitpatrol/internal/logging"
	"gitpatrol/internal/service"
	"gitpatrol/internal/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed build/*
var uiBuild embed.FS

const shutdownTimeout = 30 * time.Second

func main() {
	cfg := config.LoadConfig(config.EnvPath())

	db, err := database.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}

	os.MkdirAll(cfg.DataDir, 0755)

	hub := websocket.NewHub()

	// Setup Structured Logging
	flushLogs := logging.Setup(db, hub.BroadcastSystemLog)

	repoService := service.NewRepoService(db, hub, cfg)
	syncManager := service.NewSyncManager(cfg.WorkerCount, db, hub, repoService)
	authService := auth.NewAuthService(cfg, db)
	healthService := service.NewHealthService(db, hub, syncManager, cfg)
	exportService := service.NewExportService(db, cfg)

	healthService.Start()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.CorsAllowedOrigins,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))

	h := api.NewHandler(db, authService, syncManager, repoService, healthService, exportService, hub, cfg)
	h.RegisterRoutes(e)

	e.Static("/avatars", filepath.Join(cfg.DataDir, "avatars"))

	// UI Service (Embedded)
	service.UI = uiBuild
	uiService := service.NewUIService()
	uiService.RegisterRoutes(e)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// Scheduler
	schedulerDone := make(chan struct{})
	go func() {
		defer close(schedulerDone)
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		type dueRepo struct {
			id        int
			name, url string
		}

		for {
			var due []dueRepo
			rows, _ := db.Query("SELECT id, name, url, interval_minutes, last_sync FROM repositories WHERE auto_patrol = 1 AND status != 'syncing'")
			if rows != nil {
				for rows.Next() {
					var id int
					var name, url string
					var interval int
					var lastSync sql.NullTime

					if err := rows.Scan(&id, &name, &url, &interval, &lastSync); err != nil {
						continue
					}

					shouldSync := true
					if lastSync.Valid {
						if time.Since(lastSync.Time) < time.Duration(interval)*time.Minute {
							shouldSync = false
						}
					}

					if shouldSync {
						due = append(due, dueRepo{id: id, name: name, url: url})
					}
				}
				rows.Close()
			}

			for _, r := range due {
				syncManager.Enqueue(r.id, r.url, r.name)
			}

			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	serverErr := make(chan error, 1)
	go func() { serverErr <- e.Start(":" + cfg.Port) }()

	exitCode := 0
	select {
	case err := <-serverErr:
		slog.Error("HTTP server stopped unexpectedly", "error", err)
		exitCode = 1
	case <-ctx.Done():
		slog.Info("Shutdown signal received, stopping GitPatrol")
	}
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Warn("HTTP server did not shut down cleanly, closing remaining connections", "error", err)
		e.Close()
	}
	healthService.Stop()
	syncManager.Stop(shutdownCtx)
	<-schedulerDone

	slog.Info("GitPatrol stopped")
	flushLogs()
	db.Close()

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
