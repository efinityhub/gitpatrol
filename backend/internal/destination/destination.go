package destination

import (
	"gitpatrol/internal/models"
	"strings"
)

type Destination interface {
	CreateRepository(name string, description string) (string, error)
	PushMirror(localPath string, targetURL string) error
	SyncMetadata(repo *models.Repository, targetURL string) error
}

func redactToken(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, "***")
}
