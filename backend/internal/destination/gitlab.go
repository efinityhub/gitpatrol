package destination

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gitpatrol/internal/models"
	"net/http"
	"os/exec"
	"net/url"
)

type GitLabDestination struct {
	BaseURL string
	Token   string
}

func (g *GitLabDestination) CreateRepository(name string, description string) (string, error) {
	apiURL := fmt.Sprintf("%s/api/v4/projects", g.BaseURL)
	
	body := map[string]interface{}{
		"name":        name,
		"description": description,
		"visibility":  "private",
	}
	
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}
	
	var result struct {
		HttpURLToRepo string `json:"http_url_to_repo"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	
	return result.HttpURLToRepo, nil
}

func (g *GitLabDestination) PushMirror(localPath string, targetURL string) error {
	// targetURL is like https://gitlab.com/user/repo.git
	parsedURL, _ := url.Parse(targetURL)
	authenticatedURL := fmt.Sprintf("https://oauth2:%s@%s", g.Token, parsedURL.Host+parsedURL.Path)

	cmd := exec.Command("git", "push", "--mirror", authenticatedURL)
	cmd.Dir = localPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push --mirror to GitLab failed: %v, output: %s", err, redactToken(string(output), g.Token))
	}
	
	return nil
}

func (g *GitLabDestination) SyncMetadata(repo *models.Repository, targetURL string) error {
	return nil
}
