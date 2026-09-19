package source

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitpatrol/internal/models"
)

var githubRateLimitUntil time.Time

type GitHubSource struct {
	token string
}

func (s *GitHubSource) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if s.token != "" {
		req.Header.Set("Authorization", "token "+s.token)
	}
	return http.DefaultClient.Do(req)
}

func (s *GitHubSource) parseURL(rawURL string) (string, string, error) {
	rawURL = strings.TrimSuffix(rawURL, "/")
	rawURL = strings.TrimSuffix(rawURL, ".git")

	var path string
	if strings.Contains(rawURL, ":") && strings.HasPrefix(rawURL, "git@") {
		parts := strings.Split(rawURL, ":")
		path = parts[len(parts)-1]
	} else {
		parts := strings.Split(rawURL, "/")
		if len(parts) >= 2 {
			path = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
	}

	if path != "" {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return path, parts[0], nil
		}
	}

	return "", "", fmt.Errorf("invalid GitHub URL: %s", rawURL)
}

func (s *GitHubSource) GetMetadata(url string) (models.Metadata, error) {
	if time.Now().Before(githubRateLimitUntil) {
		return models.Metadata{}, fmt.Errorf("GitHub API rate limit active, skipping metadata")
	}

	repoPath, username, err := s.parseURL(url)
	if err != nil {
		return models.Metadata{}, err
	}

	resp, err := s.get(fmt.Sprintf("https://api.github.com/repos/%s", repoPath))
	if err != nil {
		return models.Metadata{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		slog.Warn("GitHub Rate Limit hit. Pausing API calls for 15 minutes.")
		githubRateLimitUntil = time.Now().Add(15 * time.Minute)
		return models.Metadata{}, fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode != 200 {
		return models.Metadata{}, fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, repoPath)
	}

	var meta struct {
		StargazersCount int `json:"stargazers_count"`
		ForksCount      int `json:"forks_count"`
		OpenIssuesCount int `json:"open_issues_count"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &meta)

	return models.Metadata{
		Stars:      meta.StargazersCount,
		Forks:      meta.ForksCount,
		OpenIssues: meta.OpenIssuesCount,
		AvatarURL:  fmt.Sprintf("https://github.com/%s.png?size=100", username),
		Username:   username,
	}, nil
}

func (s *GitHubSource) GetWikiURL(url string) (string, bool) {
	wikiURL := strings.TrimSuffix(url, ".git") + ".wiki.git"
	return wikiURL, true
}

func (s *GitHubSource) SyncIssues(url string, destPath string) error {
	if time.Now().Before(githubRateLimitUntil) {
		return nil
	}

	repoPath, _, err := s.parseURL(url)
	if err != nil {
		return err
	}

	resp, err := s.get(fmt.Sprintf("https://api.github.com/repos/%s/issues?state=all&per_page=100", repoPath))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		githubRateLimitUntil = time.Now().Add(15 * time.Minute)
		return fmt.Errorf("GitHub API rate limit hit")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	os.MkdirAll(destPath, 0755)
	outFile, err := os.Create(filepath.Join(destPath, "issues.json"))
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	return err
}

func (s *GitHubSource) SyncReleases(url string, destPath string) error {
	if time.Now().Before(githubRateLimitUntil) {
		return nil
	}

	repoPath, _, err := s.parseURL(url)
	if err != nil {
		return err
	}

	resp, err := s.get(fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=100", repoPath))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		githubRateLimitUntil = time.Now().Add(15 * time.Minute)
		return fmt.Errorf("GitHub API rate limit hit")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	os.MkdirAll(destPath, 0755)
	outFile, err := os.Create(filepath.Join(destPath, "releases.json"))
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	return err
}
