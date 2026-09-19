package destination

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gitpatrol/internal/models"
	"net/http"
	"os/exec"
)

type GiteaDestination struct {
	BaseURL string
	Token   string
}

func (g *GiteaDestination) CreateRepository(name string, description string) (string, error) {
	apiURL := fmt.Sprintf("%s/api/v1/user/repos", g.BaseURL)
	
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
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return "", fmt.Errorf("Gitea API returned status %d", resp.StatusCode)
	}
	
	var result struct {
		CloneURL string `json:"clone_url"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	
	return result.CloneURL, nil
}

func (g *GiteaDestination) PushMirror(localPath string, targetURL string) error {
	// targetURL typically needs the token for authentication if it's not already in the URL
	// For this POC, we'll assume targetURL is the full URL with auth or we'll inject it.
	
	cmd := exec.Command("git", "push", "--mirror", targetURL)
	cmd.Dir = localPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push --mirror failed: %v, output: %s", err, redactToken(string(output), g.Token))
	}
	
	return nil
}

func (g *GiteaDestination) SyncMetadata(repo *models.Repository, targetURL string) error {
	// In a full implementation, we'd use the Gitea API to create issues, releases, etc.
	// For this POC, we'll just acknowledge that the logic goes here.
	return nil
}
