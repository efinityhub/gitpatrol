package main

import (
	"database/sql"
	"embed"
	"log"
	"os"
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

func main() {
	cfg := config.LoadConfig("./db/gitpatrol.env")

	db, err := database.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	os.MkdirAll("./data", 0755)

	hub := websocket.NewHub()
	
	// Setup Structured Logging
	logging.Setup(db, hub.BroadcastSystemLog)

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
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))

	h := api.NewHandler(db, authService, syncManager, repoService, healthService, exportService, hub, cfg)
	h.RegisterRoutes(e)

	e.Static("/avatars", "./data/avatars")

	// UI Service (Embedded)
	service.UI = uiBuild
	uiService := service.NewUIService()
	uiService.RegisterRoutes(e)

	// Scheduler
	go func() {
		for {
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
						syncManager.Enqueue(id, url, name)
					}
				}
				rows.Close()
			}
			time.Sleep(1 * time.Minute)
		}
	}()

	e.Logger.Fatal(e.Start(":8080"))
}
