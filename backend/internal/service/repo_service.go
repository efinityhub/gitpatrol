package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gitpatrol/internal/config"
	"gitpatrol/internal/database"
	"gitpatrol/internal/models"
	"gitpatrol/internal/source"
	"gitpatrol/internal/websocket"
)

type RepoService struct {
	db  *database.DB
	hub *websocket.Hub
	cfg *config.Config
}

func NewRepoService(db *database.DB, hub *websocket.Hub, cfg *config.Config) *RepoService {
	return &RepoService{
		db:  db,
		hub: hub,
		cfg: cfg,
	}
}

func (s *RepoService) SyncRepo(id int, url, name string) {
	s.UpdateStatus(id, "syncing", "")

	repoPath := filepath.Join("./data", name)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", url, repoPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			s.UpdateStatus(id, "error", formatCLIError(string(output), err))
			return
		}
	} else {
		cmd := exec.Command("git", "-C", repoPath, "fetch", "--all", "--tags", "--force")
		if output, err := cmd.CombinedOutput(); err != nil {
			s.UpdateStatus(id, "error", formatCLIError(string(output), err))
			return
		}
	}

	// Load existing metadata to avoid overwriting with zeros if fetch fails
	var existingStars, existingForks, existingIssues int
	s.db.QueryRow("SELECT stars, forks, open_issues FROM repositories WHERE id = ?", id).Scan(&existingStars, &existingForks, &existingIssues)

	src, err := source.GetSource(url, s.cfg.GithubToken, s.cfg.GitlabToken)
	var meta models.Metadata
	meta.Stars = existingStars
	meta.Forks = existingForks
	meta.OpenIssues = existingIssues

	if err == nil {
		newMeta, fetchErr := src.GetMetadata(url)
		if fetchErr == nil {
			meta = newMeta
			if meta.Username != "" {
				s.downloadAvatar(meta.AvatarURL, meta.Username)
			}
		} else {
			slog.Warn("Failed to fetch metadata", "url", url, "error", fetchErr)
		}

		if wikiURL, exists := src.GetWikiURL(url); exists {
			s.syncWiki(wikiURL, name)
		}

		metadataPath := filepath.Join("./data", name, "metadata")
		src.SyncIssues(url, metadataPath)
		src.SyncReleases(url, metadataPath)
	}

	history := s.getCommitHistory(repoPath)
	lastCommits := s.getLastCommits(repoPath)

	cmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%cI")
	output, _ := cmd.CombinedOutput()
	lastCommitTime, _ := time.Parse(time.RFC3339, strings.TrimSpace(string(output)))

	score := s.calculateHealthScore(meta, history, lastCommitTime)

	branchCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, _ := branchCmd.CombinedOutput()
	defaultBranch := strings.TrimSpace(string(branchOut))
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	s.db.Exec(`UPDATE repositories SET 
		status = 'synced', 
		last_sync = ?, 
		last_commit = ?, 
		stars = ?, 
		forks = ?, 
		open_issues = ?, 
		commit_history = ?, 
		health_score = ?,
		default_branch = ?,
		error_message = '' 
		WHERE id = ?`,
		time.Now(), lastCommits, meta.Stars, meta.Forks, meta.OpenIssues, history, score, defaultBranch, id)

	s.hub.BroadcastStatus(id, "synced", "")
}

func (s *RepoService) UpdateStatus(id int, status, errMsg string) {
	s.db.Exec("UPDATE repositories SET status = ?, error_message = ? WHERE id = ?", status, errMsg, id)

	if status == "error" {
		var repoName string
		s.db.QueryRow("SELECT name FROM repositories WHERE id = ?", id).Scan(&repoName)

		lowerMsg := strings.ToLower(errMsg)
		isPermanent := strings.Contains(lowerMsg, "not found") ||
			strings.Contains(lowerMsg, "access denied") ||
			strings.Contains(lowerMsg, "invalid repository") ||
			strings.Contains(lowerMsg, "already in use") ||
			strings.Contains(lowerMsg, "authentication failed")

		if isPermanent {
			s.db.Exec("UPDATE repositories SET auto_patrol = 0 WHERE id = ?", id)
			slog.Warn("Disabled auto_patrol due to permanent failure", "repo", repoName, "error", errMsg)
		}

		var lastMessage string
		err := s.db.QueryRow("SELECT message FROM incidents WHERE repo_id = ? AND resolved = 0 ORDER BY created_at DESC LIMIT 1", id).Scan(&lastMessage)
		if err != nil || lastMessage != errMsg {
			s.db.Exec("INSERT INTO incidents (repo_id, repo_name, message, created_at) VALUES (?, ?, ?, ?)", id, repoName, errMsg, time.Now())
		}
	}

	s.hub.BroadcastStatus(id, status, errMsg)
}

func (s *RepoService) syncWiki(url string, name string) {
	wikiPath := filepath.Join("./data", name, "wiki")
	if _, err := os.Stat(wikiPath); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", url, wikiPath)
		cmd.Run()
	} else {
		exec.Command("git", "-C", wikiPath, "fetch", "--all", "--tags", "--force").Run()
	}
}

func (s *RepoService) getCommitHistory(repoPath string) string {
	history := make([]int, 14)
	now := time.Now()
	for i := 0; i < 14; i++ {
		day := now.AddDate(0, 0, -i)
		start := day.Format("2006-01-02 00:00:00")
		end := day.Format("2006-01-02 23:59:59")

		cmd := exec.Command("git", "-C", repoPath, "rev-list", "--count", "--all", "--since=\""+start+"\"", "--until=\""+end+"\"")
		output, _ := cmd.CombinedOutput()
		var count int
		fmt.Sscanf(string(output), "%d", &count)
		history[13-i] = count
	}
	res, _ := json.Marshal(history)
	return string(res)
}

func (s *RepoService) getLastCommits(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "log", "--all", "-10", "--format=%H|%an|%cr|%s|%d")
	output, _ := cmd.CombinedOutput()
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	var results []string
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		refs := strings.TrimSpace(parts[4])
		if refs == "" || refs == "()" {
			hash := parts[0]
			branchCmd := exec.Command("git", "-C", repoPath, "branch", "-a", "--contains", hash)
			branchOut, _ := branchCmd.CombinedOutput()
			bLines := strings.Split(strings.TrimSpace(string(branchOut)), "\n")

			if len(bLines) > 0 {
				for _, b := range bLines {
					b = strings.TrimSpace(strings.TrimPrefix(b, "*"))
					if !strings.Contains(b, "HEAD") && b != "" {
						branch := strings.TrimPrefix(b, "remotes/origin/")
						parts[4] = "(" + branch + ")"
						break
					}
				}
			}
		}
		results = append(results, strings.Join(parts, "|"))
	}
	return strings.Join(results, "\n")
}

func (s *RepoService) calculateHealthScore(meta models.Metadata, historyStr string, lastCommitDate time.Time) int {
	score := 50
	var history []int
	json.Unmarshal([]byte(historyStr), &history)
	activeDays := 0
	for _, count := range history {
		if count > 0 {
			activeDays++
		}
	}
	score += activeDays * 3
	daysSinceLast := int(time.Since(lastCommitDate).Hours() / 24)
	if daysSinceLast > 30 {
		score -= 10
	}
	if daysSinceLast > 90 {
		score -= 20
	}
	if daysSinceLast > 365 {
		score -= 30
	}
	if meta.Stars > 1000 {
		score += 5
	}
	if meta.Stars > 10000 {
		score += 5
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func (s *RepoService) downloadAvatar(url string, username string) {
	os.MkdirAll("./data/avatars", 0755)
	avatarPath := filepath.Join("./data/avatars", username+".png")
	if _, err := os.Stat(avatarPath); err == nil {
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(avatarPath)
	if err != nil {
		return
	}
	defer out.Close()

	io.Copy(out, resp.Body)
}

func formatCLIError(output string, err error) string {
	out := strings.ToLower(output)
	if strings.Contains(out, "authentication failed") || strings.Contains(out, "terminal prompts disabled") {
		return "Authentication failed. Private repositories are not supported yet."
	}
	if strings.Contains(out, "not found") || strings.Contains(out, "could not read from remote") {
		return "The remote repository was not found. Please check the URL."
	}
	if strings.Contains(out, "connection refused") || strings.Contains(out, "could not resolve host") {
		return "Failed to connect to the host. Check your internet connection or the provider status."
	}
	if strings.Contains(out, "already exists and is not an empty directory") {
		return "The local storage for this repository is already in use by another folder."
	}

	if err != nil {
		if strings.Contains(err.Error(), "exit status 128") {
			return "Access denied or invalid repository. Verify the URL and permissions."
		}
		return "Operation failed: " + err.Error()
	}
	return "An unexpected error occurred during synchronization."
}
