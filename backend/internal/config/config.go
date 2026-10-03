package config

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const defaultEnvPath = "./db/gitpatrol.env"

func EnvPath() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	return defaultEnvPath
}

type Config struct {
	JWTSecret          string
	PasswordPepper     string
	DBPath             string
	DataDir            string
	Port               string
	CorsAllowedOrigins []string
	WorkerCount        int
	GithubToken        string
	GitlabURL          string
	GitlabToken        string
	GiteaURL           string
	GiteaToken         string
	ExportDestination  string
	LogRetentionDays   int
	LogMaxRows         int
	SyncTimeoutMinutes int
	SecureCookie       bool
	envPath            string
}

func LoadConfig(envPath string) *Config {
	_ = godotenv.Load(envPath)

	modified := false

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = generateRandomString(16)
		os.Setenv("JWT_SECRET", jwtSecret)
		modified = true
	}

	pepper := os.Getenv("PASSWORD_PEPPER")
	if pepper == "" {
		pepper = generateRandomString(16)
		os.Setenv("PASSWORD_PEPPER", pepper)
		modified = true
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./db/gitpatrol.db"
		os.Setenv("DB_PATH", dbPath)
		modified = true
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
		os.Setenv("DATA_DIR", dataDir)
		modified = true
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		os.Setenv("PORT", port)
		modified = true
	}

	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:5173,http://localhost:3000"
		os.Setenv("CORS_ALLOWED_ORIGINS", corsOrigins)
		modified = true
	}

	workerCount, err := strconv.Atoi(os.Getenv("WORKERS"))
	if workerCount == 0 || err != nil {
		workerCount = 3
		os.Setenv("WORKERS", strconv.Itoa(workerCount))
		modified = true
	}

	github := os.Getenv("GITHUB_TOKEN")
	gitlabURL := os.Getenv("GITLAB_URL")
	if gitlabURL == "" {
		gitlabURL = "https://gitlab.com"
	}
	gitlabToken := os.Getenv("GITLAB_TOKEN")
	giteaURL := os.Getenv("GITEA_URL")
	giteaToken := os.Getenv("GITEA_TOKEN")
	exportDest := os.Getenv("EXPORT_DESTINATION")

	logRetentionDays, err := strconv.Atoi(os.Getenv("LOG_RETENTION_DAYS"))
	if logRetentionDays == 0 || err != nil {
		logRetentionDays = 7
		os.Setenv("LOG_RETENTION_DAYS", strconv.Itoa(logRetentionDays))
		modified = true
	}

	logMaxRows, err := strconv.Atoi(os.Getenv("LOG_MAX_ROWS"))
	if logMaxRows == 0 || err != nil {
		logMaxRows = 10000
		os.Setenv("LOG_MAX_ROWS", strconv.Itoa(logMaxRows))
		modified = true
	}

	syncTimeoutMinutes, err := strconv.Atoi(os.Getenv("SYNC_TIMEOUT_MINUTES"))
	if syncTimeoutMinutes == 0 || err != nil {
		syncTimeoutMinutes = 10
		os.Setenv("SYNC_TIMEOUT_MINUTES", strconv.Itoa(syncTimeoutMinutes))
		modified = true
	}

	secureCookie, err := strconv.ParseBool(os.Getenv("GP_SECURE_COOKIE"))
	if err != nil {
		secureCookie = false
		os.Setenv("GP_SECURE_COOKIE", "false")
		modified = true
	}

	cfg := &Config{
		JWTSecret:          jwtSecret,
		PasswordPepper:     pepper,
		DBPath:             dbPath,
		DataDir:            dataDir,
		Port:               port,
		CorsAllowedOrigins: strings.Split(corsOrigins, ","),
		WorkerCount:        workerCount,
		GithubToken:        github,
		GitlabURL:          gitlabURL,
		GitlabToken:        gitlabToken,
		GiteaURL:           giteaURL,
		GiteaToken:         giteaToken,
		ExportDestination:  exportDest,
		LogRetentionDays:   logRetentionDays,
		LogMaxRows:         logMaxRows,
		SyncTimeoutMinutes: syncTimeoutMinutes,
		SecureCookie:       secureCookie,
		envPath:            envPath,
	}

	if modified {
		_ = cfg.save()
	}

	return cfg
}

func (c *Config) UpdateTokens(github, gitlabURL, gitlabToken, giteaURL, giteaToken, exportDest string) error {
	if github != "" {
		c.GithubToken = github
		os.Setenv("GITHUB_TOKEN", github)
	}
	if gitlabURL != "" {
		c.GitlabURL = gitlabURL
		os.Setenv("GITLAB_URL", gitlabURL)
	}
	if gitlabToken != "" {
		c.GitlabToken = gitlabToken
		os.Setenv("GITLAB_TOKEN", gitlabToken)
	}
	if giteaURL != "" {
		c.GiteaURL = giteaURL
		os.Setenv("GITEA_URL", giteaURL)
	}
	if giteaToken != "" {
		c.GiteaToken = giteaToken
		os.Setenv("GITEA_TOKEN", giteaToken)
	}
	if exportDest != "" {
		c.ExportDestination = exportDest
		os.Setenv("EXPORT_DESTINATION", exportDest)
	}

	return c.save()
}

func (c *Config) save() error {
	env := map[string]string{
		"JWT_SECRET":           c.JWTSecret,
		"PASSWORD_PEPPER":      c.PasswordPepper,
		"DB_PATH":              c.DBPath,
		"DATA_DIR":             c.DataDir,
		"PORT":                 c.Port,
		"CORS_ALLOWED_ORIGINS": strings.Join(c.CorsAllowedOrigins, ","),
		"WORKERS":              strconv.Itoa(c.WorkerCount),
		"GITHUB_TOKEN":         c.GithubToken,
		"GITLAB_URL":           c.GitlabURL,
		"GITLAB_TOKEN":         c.GitlabToken,
		"GITEA_URL":            c.GiteaURL,
		"GITEA_TOKEN":          c.GiteaToken,
		"EXPORT_DESTINATION":   c.ExportDestination,
		"LOG_RETENTION_DAYS":   strconv.Itoa(c.LogRetentionDays),
		"LOG_MAX_ROWS":         strconv.Itoa(c.LogMaxRows),
		"SYNC_TIMEOUT_MINUTES": strconv.Itoa(c.SyncTimeoutMinutes),
		"GP_SECURE_COOKIE":     strconv.FormatBool(c.SecureCookie),
	}

	if err := os.MkdirAll(filepath.Dir(c.envPath), 0755); err != nil {
		slog.Warn("Could not create config directory", "error", err)
		return err
	}

	if err := godotenv.Write(env, c.envPath); err != nil {
		slog.Warn("Could not save .env file", "error", err)
		return err
	}
	if err := os.Chmod(c.envPath, 0600); err != nil {
		slog.Warn("Could not restrict .env file permissions", "error", err)
	}
	slog.Info("Secrets generated and saved", "path", c.envPath)
	return nil
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "efinity-default-fallback-secret-key"
	}
	return hex.EncodeToString(b)
}
