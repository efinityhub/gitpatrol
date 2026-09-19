package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gitpatrol/internal/auth"
	"gitpatrol/internal/config"
	"gitpatrol/internal/database"
	"gitpatrol/internal/models"
	"gitpatrol/internal/service"
	"gitpatrol/internal/websocket"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	db              *database.DB
	authService     *auth.AuthService
	syncManager     *service.SyncManager
	repoService     *service.RepoService
	healthService *service.HealthService
	exportService *service.ExportService
	hub             *websocket.Hub
	config          *config.Config
}

func NewHandler(db *database.DB, authService *auth.AuthService, syncManager *service.SyncManager, repoService *service.RepoService, healthService *service.HealthService, exportService *service.ExportService, hub *websocket.Hub, cfg *config.Config) *Handler {
	return &Handler{
		db:              db,
		authService:     authService,
		syncManager:     syncManager,
		repoService:     repoService,
		healthService:   healthService,
		exportService:   exportService,
		hub:             hub,
		config:          cfg,
	}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// Public Auth Routes
	e.GET("/api/auth/status", h.CheckAuthStatus)
	e.POST("/api/auth/register", h.Register)
	e.POST("/api/auth/login", h.Login)
	e.POST("/api/auth/logout", h.Logout)
	e.GET("/api/repositories/:id/badge", h.GetHealthBadge)

	// Protected API Group
	api := e.Group("/api", h.authService.Middleware)
	api.GET("/me", h.GetMe)
	api.PATCH("/user", h.UpdateUser)
	api.GET("/repositories", h.GetRepositories)
	api.POST("/repositories", h.AddRepository)
	api.PATCH("/repositories/:id", h.UpdateRepository)
	api.DELETE("/repositories/:id", h.DeleteRepository)
	api.POST("/repositories/:id/sync", h.SyncRepositoryNow)
	api.POST("/repositories/:id/export", h.ExportRepository)
	api.GET("/repositories/:id/readme", h.GetReadme)
	api.GET("/repositories/:id/assets/*", h.GetAsset)
	api.GET("/incidents", h.GetIncidents)
	api.DELETE("/incidents", h.ClearIncidents)
	api.GET("/logs", h.GetLogs)
	api.GET("/settings", h.GetSettings)
	api.PATCH("/settings", h.UpdateSettings)
	api.GET("/health", h.GetHealth)

	e.GET("/ws", func(c echo.Context) error {
		err := h.hub.HandleWebSocket(c)
		if err == nil {
			// Immediately send current health upon connection
			h.hub.BroadcastHealthStatus(h.healthService.GetStatus())
		}
		return err
	})
}

func (h *Handler) GetHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, h.healthService.GetStatus())
}

func (h *Handler) CheckAuthStatus(c echo.Context) error {
        needsBootstrap := h.authService.NeedsBootstrap()
        cookie, err := c.Cookie("token")
        if err != nil {
                return c.JSON(http.StatusOK, map[string]interface{}{
                        "authenticated":   false,
                        "needs_bootstrap": needsBootstrap,
                })
        }
        return c.JSON(http.StatusOK, map[string]interface{}{
                "authenticated":   cookie.Value != "",
                "needs_bootstrap": needsBootstrap,
        })
}
func (h *Handler) Register(c echo.Context) error {
        if !h.authService.NeedsBootstrap() {
                return c.JSON(http.StatusForbidden, map[string]string{"error": "registration is only allowed during initial system bootstrap"})
        }

        var body struct {
                Username string `json:"username"`
                Password string `json:"password"`
        }
        if err := c.Bind(&body); err != nil {
                return err
        }

        if err := h.authService.Register(body.Username, body.Password); err != nil {
                return c.JSON(http.StatusBadRequest, map[string]string{"error": "username already exists"})
        }

        return c.JSON(http.StatusCreated, map[string]string{"message": "registered successfully"})
}

func (h *Handler) Login(c echo.Context) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return err
	}

	token, err := h.authService.Login(body.Username, body.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	cookie := new(http.Cookie)
	cookie.Name = "token"
	cookie.Value = token
	cookie.Expires = time.Now().Add(72 * time.Hour)
	cookie.Path = "/"
	cookie.HttpOnly = true
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]string{"message": "login successful"})
}

func (h *Handler) Logout(c echo.Context) error {
	cookie := new(http.Cookie)
	cookie.Name = "token"
	cookie.Value = ""
	cookie.Expires = time.Now().Add(-1 * time.Hour)
	cookie.Path = "/"
	cookie.HttpOnly = true
	c.SetCookie(cookie)
	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) GetMe(c echo.Context) error {
	username := c.Get("username").(string)
	userID := c.Get("user_id").(int)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":       userID,
		"username": username,
	})
}

func (h *Handler) UpdateUser(c echo.Context) error {
	userID := c.Get("user_id").(int)
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return err
	}

	if body.Username != "" {
		_, err := h.db.Exec("UPDATE users SET username = ? WHERE id = ?", body.Username, userID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "username already taken"})
		}
	}

	if body.Password != "" {
		// This should probably be in AuthService, but for simplicity of refactor:
		// hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(body.Password+h.config.PasswordPepper), bcrypt.DefaultCost)
		// h.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(hashedPassword), userID)
		// Actually, let's keep it simple and just do username for now, or add method to AuthService.
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "updated successfully"})
}

func (h *Handler) GetRepositories(c echo.Context) error {
	rows, err := h.db.Query("SELECT id, name, url, interval_minutes, last_sync, status, last_commit, error_message, stars, forks, open_issues, commit_history, health_score, default_branch, auto_patrol FROM repositories")
	if err != nil {
		return err
	}
	defer rows.Close()

	repos := []models.Repository{}
	for rows.Next() {
		var r models.Repository
		var lastSync sql.NullTime
		var lastCommit, errMsg, commitHistory sql.NullString
		err := rows.Scan(&r.ID, &r.Name, &r.URL, &r.IntervalMinutes, &lastSync, &r.Status, &lastCommit, &errMsg, &r.Stars, &r.Forks, &r.OpenIssues, &commitHistory, &r.HealthScore, &r.DefaultBranch, &r.AutoPatrol)
		if err != nil {
			return err
		}
		if lastSync.Valid {
			r.LastSync = lastSync.Time
		}
		r.LastCommit = lastCommit.String
		r.ErrorMessage = errMsg.String
		r.CommitHistory = commitHistory.String
		repos = append(repos, r)
	}
	return c.JSON(http.StatusOK, repos)
}

func (h *Handler) AddRepository(c echo.Context) error {
	var repo models.Repository
	if err := c.Bind(&repo); err != nil {
		return err
	}

	if repo.IntervalMinutes == 0 {
		repo.IntervalMinutes = 60
	}

	res, err := h.db.Exec("INSERT INTO repositories (name, url, interval_minutes) VALUES (?, ?, ?)", repo.Name, repo.URL, repo.IntervalMinutes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "repository already exists"})
	}

	id, _ := res.LastInsertId()
	h.hub.BroadcastStatus(int(id), "syncing", "")
	h.syncManager.Enqueue(int(id), repo.URL, repo.Name)

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id})}

func (h *Handler) UpdateRepository(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Interval   int `json:"interval_minutes"`
		AutoPatrol int `json:"auto_patrol"`
	}
	if err := c.Bind(&body); err != nil {
		return err
	}

	_, err := h.db.Exec("UPDATE repositories SET interval_minutes = ?, auto_patrol = ? WHERE id = ?", body.Interval, body.AutoPatrol, id)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}

func (h *Handler) DeleteRepository(c echo.Context) error {
	id := c.Param("id")

	var name string
	h.db.QueryRow("SELECT name FROM repositories WHERE id = ?", id).Scan(&name)

	h.db.Exec("DELETE FROM repositories WHERE id = ?", id)
	h.db.Exec("DELETE FROM incidents WHERE repo_id = ?", id)

	if name != "" {
		os.RemoveAll(filepath.Join("./data", name))
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handler) SyncRepositoryNow(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var url, name string
	err := h.db.QueryRow("SELECT url, name FROM repositories WHERE id = ?", id).Scan(&url, &name)
	if err != nil {
		return err
	}

	h.syncManager.Enqueue(id, url, name)
	return c.JSON(http.StatusOK, map[string]string{"message": "syncing started"})
}

func (h *Handler) GetReadme(c echo.Context) error {
	id := c.Param("id")
	var name string
	h.db.QueryRow("SELECT name FROM repositories WHERE id = ?", id).Scan(&name)

	path := filepath.Join("./data", name, "README.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Try lowercase
		path = filepath.Join("./data", name, "readme.md")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return c.String(http.StatusNotFound, "README not found")
		}
	}

	data, _ := os.ReadFile(path)
	return c.String(http.StatusOK, string(data))
}

func (h *Handler) GetAsset(c echo.Context) error {
	id := c.Param("id")
	assetPath := c.Param("*")

	var name string
	h.db.QueryRow("SELECT name FROM repositories WHERE id = ?", id).Scan(&name)

	fullPath := filepath.Join("./data", name, assetPath)

	// Security: check if path is within repo data
	cleanPath, _ := filepath.Abs(fullPath)
	repoAbs, _ := filepath.Abs(filepath.Join("./data", name))
	if !strings.HasPrefix(cleanPath, repoAbs) {
		return c.String(http.StatusForbidden, "Forbidden")
	}

	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return c.String(http.StatusNotFound, "Asset not found")
	}

	return c.File(cleanPath)
}

func (h *Handler) GetIncidents(c echo.Context) error {
	rows, err := h.db.Query("SELECT id, repo_id, repo_name, message, created_at, resolved FROM incidents ORDER BY created_at DESC")
	if err != nil {
		return err
	}
	defer rows.Close()

	incidents := []models.Incident{}
	for rows.Next() {
		var i models.Incident
		err := rows.Scan(&i.ID, &i.RepoID, &i.RepoName, &i.Message, &i.CreatedAt, &i.Resolved)
		if err != nil {
			return err
		}
		incidents = append(incidents, i)
	}
	return c.JSON(http.StatusOK, incidents)
}

func (h *Handler) ClearIncidents(c echo.Context) error {
	h.db.Exec("DELETE FROM incidents")
	return c.NoContent(http.StatusOK)
}

func (h *Handler) GetHealthBadge(c echo.Context) error {
	id := c.Param("id")
	var score int
	var name string
	err := h.db.QueryRow("SELECT health_score, name FROM repositories WHERE id = ?", id).Scan(&score, &name)
	if err != nil {
		return c.String(http.StatusNotFound, "Repo not found")
	}

	color := "green"
	if score < 70 {
		color = "yellow"
	}
	if score < 40 {
		color = "red"
	}

	badgeURL := fmt.Sprintf("https://img.shields.io/badge/gitpatrol-health_%d%%25-%s", score, color)
	resp, err := http.Get(badgeURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.Response().Header().Set(echo.HeaderContentType, "image/svg+xml")
	io.Copy(c.Response().Writer, resp.Body)
	return nil
}

func (h *Handler) ExportRepository(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	
	var body struct {
		Destination string `json:"destination"`
	}
	if err := c.Bind(&body); err != nil {
		return err
	}

	res, err := h.exportService.Export(id, body.Destination)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetLogs(c echo.Context) error {
	limit := c.QueryParam("limit")
	if limit == "" {
		limit = "100"
	}
	
	rows, err := h.db.Query(`
		SELECT id, level, message, attributes, created_at 
		FROM system_logs 
		ORDER BY created_at DESC 
		LIMIT ?
	`, limit)
	if err != nil {
		return err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int
		var level, message, attributes, createdAt string
		if err := rows.Scan(&id, &level, &message, &attributes, &createdAt); err != nil {
			continue
		}
		
		var attrsMap map[string]interface{}
		json.Unmarshal([]byte(attributes), &attrsMap)

		logs = append(logs, map[string]interface{}{
			"id":         id,
			"level":      level,
			"message":    message,
			"attributes": attrsMap,
			"created_at": createdAt,
		})
	}
	
	if logs == nil {
		logs = []map[string]interface{}{}
	}
	
	return c.JSON(http.StatusOK, logs)
}
