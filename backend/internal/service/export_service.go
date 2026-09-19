package service

import (
	"fmt"
	"gitpatrol/internal/config"
	"gitpatrol/internal/database"
	"gitpatrol/internal/destination"
	"gitpatrol/internal/models"
)

type ExportService struct {
	db  *database.DB
	cfg *config.Config
}

func NewExportService(db *database.DB, cfg *config.Config) *ExportService {
	return &ExportService{
		db:  db,
		cfg: cfg,
	}
}

func (s *ExportService) Export(repoID int, destinationType string) (*models.ExportResponse, error) {
	// 1. Fetch Repo from DB
	var repo models.Repository
	err := s.db.QueryRow(`SELECT id, name, url, status FROM repositories WHERE id = ?`, repoID).Scan(
		&repo.ID, &repo.Name, &repo.URL, &repo.Status)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %v", err)
	}

	// 2. Select Destination based on type (or global config)
	if destinationType == "" {
		destinationType = s.cfg.ExportDestination
	}

	if destinationType == "" {
		return nil, fmt.Errorf("no export destination configured. go to user settings to select a vault.")
	}

	var dest destination.Destination

	switch destinationType {
	case "gitea":
		if s.cfg.GiteaURL == "" || s.cfg.GiteaToken == "" {
			return nil, fmt.Errorf("Gitea configuration is missing")
		}
		dest = &destination.GiteaDestination{
			BaseURL: s.cfg.GiteaURL,
			Token:   s.cfg.GiteaToken,
		}
	case "github":
		if s.cfg.GithubToken == "" {
			return nil, fmt.Errorf("GitHub token is not configured")
		}
		dest = &destination.GitHubDestination{
			Token: s.cfg.GithubToken,
		}
	case "gitlab":
		if s.cfg.GitlabToken == "" {
			return nil, fmt.Errorf("GitLab token is not configured")
		}
		dest = &destination.GitLabDestination{
			BaseURL: s.cfg.GitlabURL,
			Token:   s.cfg.GitlabToken,
		}
	default:
		return nil, fmt.Errorf("unsupported destination: %s", destinationType)
	}

	// 3. Create Remote Repo
	cloneURL, err := dest.CreateRepository(repo.Name, "Recovered via GitPatrol")
	if err != nil {
		return nil, fmt.Errorf("failed to create repository on %s: %v", destinationType, err)
	}

	// 4. Push Mirror
	localPath := RepoPath(s.cfg.DataDir, repo.ID, repo.Name)
	err = dest.PushMirror(localPath, cloneURL)
	if err != nil {
		return nil, fmt.Errorf("failed to push mirror: %v", err)
	}

	return &models.ExportResponse{
		Success:        true,
		Message:        fmt.Sprintf("Repository successfully exported to %s", destinationType),
		DestinationURL: cloneURL,
	}, nil
}
