package database

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	if err := os.MkdirAll("db", 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode and set busy timeout for better concurrency
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA busy_timeout=5000;")
	db.SetMaxOpenConns(1)

	if err := createTables(db); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE,
			password_hash TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS system_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT,
			message TEXT,
			attributes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS incidents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER,
			repo_name TEXT,
			message TEXT,
			created_at TIMESTAMP,
			resolved INTEGER DEFAULT 0,
			FOREIGN KEY(repo_id) REFERENCES repositories(id)
		);`,
		`CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			url TEXT UNIQUE,
			interval_minutes INTEGER DEFAULT 60,
			last_sync TIMESTAMP,
			status TEXT DEFAULT 'pending',
			last_commit TEXT,
			error_message TEXT,
			stars INTEGER DEFAULT 0,
			forks INTEGER DEFAULT 0,
			open_issues INTEGER DEFAULT 0,
			commit_history TEXT,
			health_score INTEGER DEFAULT 0,
			default_branch TEXT DEFAULT 'main',
			auto_patrol INTEGER DEFAULT 1
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	// Migrations (idempotent ADD COLUMN if not exists)
	// SQLite doesn't support IF NOT EXISTS for ADD COLUMN easily, but exec and ignore error is common for simple cases
	// or check table info. For simplicity in this refactor, we keep them as is.
	cols := []string{
		"ALTER TABLE repositories ADD COLUMN stars INTEGER DEFAULT 0",
		"ALTER TABLE repositories ADD COLUMN forks INTEGER DEFAULT 0",
		"ALTER TABLE repositories ADD COLUMN open_issues INTEGER DEFAULT 0",
		"ALTER TABLE repositories ADD COLUMN commit_history TEXT",
		"ALTER TABLE repositories ADD COLUMN health_score INTEGER DEFAULT 0",
		"ALTER TABLE repositories ADD COLUMN default_branch TEXT DEFAULT 'main'",
		"ALTER TABLE repositories ADD COLUMN auto_patrol INTEGER DEFAULT 1",
	}
	for _, c := range cols {
		_, _ = db.Exec(c)
	}

	return nil
}
