package config

import (
	"os"
	"path/filepath"
	"testing"
)

var envKeys = []string{
	"JWT_SECRET", "PASSWORD_PEPPER", "DB_PATH", "DATA_DIR", "PORT", "CORS_ALLOWED_ORIGINS",
	"WORKERS", "GITHUB_TOKEN", "GITLAB_URL", "GITLAB_TOKEN", "GITEA_URL", "GITEA_TOKEN",
	"EXPORT_DESTINATION", "LOG_RETENTION_DAYS", "LOG_MAX_ROWS", "SYNC_TIMEOUT_MINUTES", "GP_SECURE_COOKIE",
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range envKeys {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	clearEnv(t)
	cfg := LoadConfig(filepath.Join(t.TempDir(), "db", "gitpatrol.env"))

	if cfg.Port != "8080" || cfg.WorkerCount != 3 || cfg.SyncTimeoutMinutes != 10 {
		t.Errorf("unexpected defaults: port=%s workers=%d timeout=%d", cfg.Port, cfg.WorkerCount, cfg.SyncTimeoutMinutes)
	}
	if cfg.LogRetentionDays != 7 || cfg.LogMaxRows != 10000 {
		t.Errorf("unexpected log defaults: days=%d rows=%d", cfg.LogRetentionDays, cfg.LogMaxRows)
	}
	if cfg.DBPath != "./db/gitpatrol.db" || cfg.DataDir != "./data" {
		t.Errorf("unexpected paths: db=%s data=%s", cfg.DBPath, cfg.DataDir)
	}
	if cfg.SecureCookie {
		t.Error("secure cookie must default to off so plain-HTTP installs can log in")
	}
	if len(cfg.JWTSecret) < 16 || len(cfg.PasswordPepper) < 16 || cfg.JWTSecret == cfg.PasswordPepper {
		t.Errorf("secrets must be generated and distinct: %q %q", cfg.JWTSecret, cfg.PasswordPepper)
	}
	if cfg.GitlabURL != "https://gitlab.com" {
		t.Errorf("gitlab url = %s", cfg.GitlabURL)
	}
}

func TestLoadConfigKeepsSecretsAcrossRestarts(t *testing.T) {
	clearEnv(t)
	envPath := filepath.Join(t.TempDir(), "db", "gitpatrol.env")

	first := LoadConfig(envPath)
	if _, err := os.Stat(envPath); err != nil {
		t.Fatalf("generated config was not persisted: %v", err)
	}

	for _, k := range envKeys {
		os.Unsetenv(k)
	}
	second := LoadConfig(envPath)

	if first.JWTSecret != second.JWTSecret || first.PasswordPepper != second.PasswordPepper {
		t.Error("secrets changed between restarts, which would lock the operator out")
	}
}

func TestLoadConfigEnvironmentOverrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("WORKERS", "8")
	t.Setenv("DATA_DIR", "/srv/gitpatrol/data")
	t.Setenv("SYNC_TIMEOUT_MINUTES", "30")
	t.Setenv("GP_SECURE_COOKIE", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example,https://b.example")

	cfg := LoadConfig(filepath.Join(t.TempDir(), "gitpatrol.env"))

	if cfg.Port != "9090" || cfg.WorkerCount != 8 || cfg.DataDir != "/srv/gitpatrol/data" || cfg.SyncTimeoutMinutes != 30 {
		t.Errorf("overrides ignored: %+v", cfg)
	}
	if !cfg.SecureCookie {
		t.Error("GP_SECURE_COOKIE=true ignored")
	}
	if len(cfg.CorsAllowedOrigins) != 2 || cfg.CorsAllowedOrigins[1] != "https://b.example" {
		t.Errorf("cors origins = %v", cfg.CorsAllowedOrigins)
	}
}

func TestLoadConfigRepairsInvalidNumbers(t *testing.T) {
	clearEnv(t)
	t.Setenv("WORKERS", "many")
	t.Setenv("SYNC_TIMEOUT_MINUTES", "0")

	cfg := LoadConfig(filepath.Join(t.TempDir(), "gitpatrol.env"))
	if cfg.WorkerCount != 3 || cfg.SyncTimeoutMinutes != 10 {
		t.Errorf("invalid numbers must fall back to defaults: workers=%d timeout=%d", cfg.WorkerCount, cfg.SyncTimeoutMinutes)
	}
}

func TestEnvPath(t *testing.T) {
	t.Setenv("CONFIG_PATH", "")
	if got := EnvPath(); got != "./db/gitpatrol.env" {
		t.Errorf("default = %q", got)
	}
	t.Setenv("CONFIG_PATH", "/var/lib/gitpatrol/db/gitpatrol.env")
	if got := EnvPath(); got != "/var/lib/gitpatrol/db/gitpatrol.env" {
		t.Errorf("override = %q", got)
	}
}

func TestSavedConfigIsOwnerOnly(t *testing.T) {
	clearEnv(t)
	envPath := filepath.Join(t.TempDir(), "gitpatrol.env")
	LoadConfig(envPath)

	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("mode = %o, want 600 because the file holds secrets and tokens", perm)
	}
}
