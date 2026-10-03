package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitpatrol/internal/config"
	"gitpatrol/internal/models"
)

func daysAgo(n int) time.Time {
	return time.Now().Add(-time.Duration(n) * 24 * time.Hour)
}

func TestCalculateHealthScore(t *testing.T) {
	s := &RepoService{}
	allActive := "[1,1,1,1,1,1,1,1,1,1,1,1,1,1]"
	fiveActive := "[1,0,2,0,3,0,0,4,0,0,5,0,0,0]"

	tests := []struct {
		name    string
		meta    models.Metadata
		history string
		last    time.Time
		want    int
	}{
		{"no activity, fresh commit", models.Metadata{}, "[]", daysAgo(0), 50},
		{"invalid history is ignored", models.Metadata{}, "not json", daysAgo(0), 50},
		{"every day active", models.Metadata{}, allActive, daysAgo(0), 92},
		{"counts active days, not commits", models.Metadata{}, fiveActive, daysAgo(0), 65},
		{"exactly 30 days is not stale", models.Metadata{}, "[]", daysAgo(30), 50},
		{"31 days is stale", models.Metadata{}, "[]", daysAgo(31), 40},
		{"over 90 days", models.Metadata{}, "[]", daysAgo(100), 20},
		{"over 365 days", models.Metadata{}, fiveActive, daysAgo(400), 5},
		{"floors at zero", models.Metadata{}, "[]", daysAgo(400), 0},
		{"1000 stars is not a bonus", models.Metadata{Stars: 1000}, "[]", daysAgo(0), 50},
		{"over 1000 stars", models.Metadata{Stars: 1001}, "[]", daysAgo(0), 55},
		{"over 10000 stars", models.Metadata{Stars: 10001}, "[]", daysAgo(0), 60},
		{"caps at 100", models.Metadata{Stars: 50000}, allActive, daysAgo(0), 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.calculateHealthScore(tt.meta, tt.history, tt.last); got != tt.want {
				t.Errorf("score = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFormatCLIError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		err    error
		want   string
	}{
		{"auth failed", "fatal: Authentication failed for 'https://github.com/a/b'", nil, "Authentication failed. For private repositories, check that GITHUB_TOKEN or GITLAB_TOKEN is set and has access."},
		{"prompts disabled", "fatal: could not read Username: terminal prompts disabled", nil, "Authentication failed. For private repositories, check that GITHUB_TOKEN or GITLAB_TOKEN is set and has access."},
		{"not found", "remote: Repository not found.", nil, "The remote repository was not found. Please check the URL."},
		{"unreadable remote", "fatal: Could not read from remote repository.", nil, "The remote repository was not found. Please check the URL."},
		{"no network", "fatal: unable to access: Could not resolve host: github.com", nil, "Failed to connect to the host. Check your internet connection or the provider status."},
		{"folder in use", "fatal: destination path 'x' already exists and is not an empty directory.", nil, "The local storage for this repository is already in use by another folder."},
		{"exit 128", "", errors.New("exit status 128"), "Access denied or invalid repository. Verify the URL and permissions."},
		{"other error", "", errors.New("boom"), "Operation failed: boom"},
		{"nothing to go on", "", nil, "An unexpected error occurred during synchronization."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCLIError(tt.output, tt.err); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatSyncErrorTimeout(t *testing.T) {
	s := &RepoService{cfg: &config.Config{SyncTimeoutMinutes: 7}}
	got := s.formatSyncError(nil, errSyncTimeout)
	want := "Sync timed out after 7 minutes. The remote may be slow, rate-limited, or unreachable."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestScrubTokens(t *testing.T) {
	cfg := &config.Config{GithubToken: "ghp_secret", GitlabToken: "glpat_secret"}
	got := scrubTokens("clone https://ghp_secret@github.com failed, glpat_secret rejected", cfg)
	want := "clone https://***@github.com failed, *** rejected"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got := scrubTokens("nothing to hide", &config.Config{}); got != "nothing to hide" {
		t.Errorf("empty tokens must not alter the text, got %q", got)
	}
}

func TestRepoPath(t *testing.T) {
	root := filepath.Join("data")
	tests := []struct {
		name string
		id   int
		repo string
		want string
	}{
		{"plain", 3, "gitpatrol", filepath.Join(root, "repos", "3_gitpatrol")},
		{"traversal is flattened", 4, "../../etc/passwd", filepath.Join(root, "repos", "4_passwd")},
		{"nested path", 5, "a/b", filepath.Join(root, "repos", "5_b")},
		{"empty name", 6, "", filepath.Join(root, "repos", "6_repo")},
		{"dot dot", 7, "..", filepath.Join(root, "repos", "7_repo")},
		{"root", 8, "/", filepath.Join(root, "repos", "8_repo")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RepoPath(root, tt.id, tt.repo); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.bin"), make([]byte, 100), 0644)
	os.MkdirAll(filepath.Join(dir, "sub", "deeper"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "b.bin"), make([]byte, 250), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "deeper", "c.bin"), make([]byte, 50), 0644)

	outside := filepath.Join(t.TempDir(), "outside.bin")
	os.WriteFile(outside, make([]byte, 9999), 0644)
	os.Symlink(outside, filepath.Join(dir, "link"))

	if got := dirSize(dir); got != 400 {
		t.Errorf("size = %d, want 400", got)
	}
	if got := dirSize(filepath.Join(dir, "missing")); got != 0 {
		t.Errorf("missing dir size = %d, want 0", got)
	}
}
