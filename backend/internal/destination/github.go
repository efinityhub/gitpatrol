package destination

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gitpatrol/internal/models"
	"net/http"
	"os/exec"
)

type GitHubDestination struct {
	Token string
}

func (g *GitHubDestination) CreateRepository(name string, description string) (string, error) {
	apiURL := "https://api.github.com/user/repos"
	
	body := map[string]interface{}{
		"name":        name,
		"description": description,
		"private":     true,
	}
	
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Authorization", "token "+g.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var result struct {
		CloneURL string `json:"clone_url"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	
	return result.CloneURL, nil
}

func (g *GitHubDestination) PushMirror(localPath string, targetURL string) error {
	// Inject token into URL for headless push
	// targetURL is like https://github.com/user/repo.git
	// we want https://token@github.com/user/repo.git
	authenticatedURL := fmt.Sprintf("https://%s@%s", g.Token, targetURL[8:])
	
	cmd := exec.Command("git", "push", "--mirror", authenticatedURL)
	cmd.Dir = localPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push --mirror to GitHub failed: %v, output: %s", err, redactToken(string(output), g.Token))
	}
	
	return nil
}

func (g *GitHubDestination) SyncMetadata(repo *models.Repository, targetURL string) error {
	return nil
}
