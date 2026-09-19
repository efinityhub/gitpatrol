package config

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret         string
	PasswordPepper    string
	DBPath            string
	WorkerCount       int
	GithubToken       string
	GitlabURL         string
	GitlabToken       string
	GiteaURL          string
	GiteaToken        string
	ExportDestination string
	LogRetentionDays  int
	LogMaxRows        int
	envPath           string
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

	if modified {
		_ = saveEnv(envPath, jwtSecret, pepper, dbPath, github, gitlabURL, gitlabToken, giteaURL, giteaToken, exportDest, workerCount, logRetentionDays, logMaxRows)
	}

	return &Config{
		JWTSecret:         jwtSecret,
		PasswordPepper:    pepper,
		DBPath:            dbPath,
		WorkerCount:       workerCount,
		GithubToken:       github,
		GitlabURL:         gitlabURL,
		GitlabToken:       gitlabToken,
		GiteaURL:          giteaURL,
		GiteaToken:        giteaToken,
		ExportDestination: exportDest,
		LogRetentionDays:  logRetentionDays,
		LogMaxRows:        logMaxRows,
		envPath:           envPath,
	}
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

	err := saveEnv(c.envPath, c.JWTSecret, c.PasswordPepper, c.DBPath, c.GithubToken, c.GitlabURL, c.GitlabToken, c.GiteaURL, c.GiteaToken, c.ExportDestination, c.WorkerCount, c.LogRetentionDays, c.LogMaxRows)

	return err
}

func saveEnv(path, jwt, pepper, dbPath, github, gitlabURL, gitlabToken, giteaURL, giteaToken, exportDest string, workerCount, logRetentionDays, logMaxRows int) error {
	env := map[string]string{
		"JWT_SECRET":         jwt,
		"PASSWORD_PEPPER":    pepper,
		"DB_PATH":            dbPath,
		"WORKERS":            strconv.Itoa(workerCount),
		"GITHUB_TOKEN":       github,
		"GITLAB_URL":         gitlabURL,
		"GITLAB_TOKEN":       gitlabToken,
		"GITEA_URL":          giteaURL,
		"GITEA_TOKEN":        giteaToken,
		"EXPORT_DESTINATION": exportDest,
		"LOG_RETENTION_DAYS": strconv.Itoa(logRetentionDays),
		"LOG_MAX_ROWS":       strconv.Itoa(logMaxRows),
	}

	err := godotenv.Write(env, path)
	if err != nil {
		slog.Warn("Could not save .env file", "error", err)
		return err
	} else {
		slog.Info("Secrets generated and saved", "path", path)
		return nil
	}
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "efinity-default-fallback-secret-key"
	}
	return hex.EncodeToString(b)
}
