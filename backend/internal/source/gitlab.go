package source

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitpatrol/internal/models"
)

var gitlabRateLimitUntil time.Time

type GitLabSource struct {
	token string
}

func (s *GitLabSource) get(apiURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	if s.token != "" {
		req.Header.Set("PRIVATE-TOKEN", s.token)
	}
	return http.DefaultClient.Do(req)
}

func (s *GitLabSource) parseURL(rawURL string) (string, error) {
	rawURL = strings.TrimSuffix(rawURL, "/")
	rawURL = strings.TrimSuffix(rawURL, ".git")

	if strings.Contains(rawURL, ":") && strings.HasPrefix(rawURL, "git@") {
		parts := strings.Split(rawURL, ":")
		return parts[len(parts)-1], nil
	}

	if strings.HasPrefix(rawURL, "http") {
		u, err := url.Parse(rawURL)
		if err == nil {
			return strings.TrimPrefix(u.Path, "/"), nil
		}
	}

	parts := strings.Split(rawURL, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1], nil
	}

	return "", fmt.Errorf("invalid GitLab URL: %s", rawURL)
}

func (s *GitLabSource) encodedPath(repoURL string) string {
	path, err := s.parseURL(repoURL)
	if err != nil {
		return ""
	}
	return url.PathEscape(path)
}

func (s *GitLabSource) GetMetadata(repoURL string) (models.Metadata, error) {
	if time.Now().Before(gitlabRateLimitUntil) {
		return models.Metadata{}, fmt.Errorf("GitLab API rate limit active, skipping metadata")
	}

	encPath := s.encodedPath(repoURL)
	if encPath == "" {
		return models.Metadata{}, fmt.Errorf("invalid URL")
	}

	resp, err := s.get(fmt.Sprintf("https://gitlab.com/api/v4/projects/%s", encPath))
	if err != nil {
		return models.Metadata{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		gitlabRateLimitUntil = time.Now().Add(15 * time.Minute)
		slog.Warn("GitLab Rate Limit hit. Pausing API calls for 15 minutes.")
		return models.Metadata{}, fmt.Errorf("GitLab API rate limit hit")
	}

	if resp.StatusCode != 200 {
		return models.Metadata{}, fmt.Errorf("GitLab API returned %d for %s", resp.StatusCode, repoURL)
	}

	var meta struct {
		StarCount       int    `json:"star_count"`
		ForksCount      int    `json:"forks_count"`
		OpenIssuesCount int    `json:"open_issues_count"`
		AvatarURL       string `json:"avatar_url"`
		Namespace       struct {
			Path string `json:"path"`
		} `json:"namespace"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &meta)

	return models.Metadata{
		Stars:      meta.StarCount,
		Forks:      meta.ForksCount,
		OpenIssues: meta.OpenIssuesCount,
		AvatarURL:  meta.AvatarURL,
		Username:   meta.Namespace.Path,
	}, nil
}

func (s *GitLabSource) GetWikiURL(repoURL string) (string, bool) {
	wikiURL := strings.TrimSuffix(repoURL, ".git") + ".wiki.git"
	return wikiURL, true
}

func (s *GitLabSource) SyncIssues(repoURL string, destPath string) error {
	if time.Now().Before(gitlabRateLimitUntil) {
		return nil
	}

	encPath := s.encodedPath(repoURL)
	if encPath == "" {
		return fmt.Errorf("invalid URL")
	}

	resp, err := s.get(fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/issues?state=all&per_page=100", encPath))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		gitlabRateLimitUntil = time.Now().Add(15 * time.Minute)
		return fmt.Errorf("GitLab API rate limit hit")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitLab API returned %d", resp.StatusCode)
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

func (s *GitLabSource) SyncReleases(repoURL string, destPath string) error {
	if time.Now().Before(gitlabRateLimitUntil) {
		return nil
	}

	encPath := s.encodedPath(repoURL)
	if encPath == "" {
		return fmt.Errorf("invalid URL")
	}

	resp, err := s.get(fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/releases?per_page=100", encPath))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		gitlabRateLimitUntil = time.Now().Add(15 * time.Minute)
		return fmt.Errorf("GitLab API rate limit hit")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitLab API returned %d", resp.StatusCode)
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
