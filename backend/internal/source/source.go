package source

import (
	"encoding/base64"
	"fmt"
	"strings"

	"gitpatrol/internal/models"
)

type Source interface {
	GetMetadata(url string) (models.Metadata, error)
	GetWikiURL(url string) (string, bool)
	SyncIssues(url string, destPath string) error
	SyncReleases(url string, destPath string) error
	GitAuthArgs() []string
}

func GetSource(url, githubToken, gitlabToken string) (Source, error) {
	if strings.Contains(url, "github.com") {
		return &GitHubSource{token: githubToken}, nil
	}
	if strings.Contains(url, "gitlab.com") {
		return &GitLabSource{token: gitlabToken}, nil
	}
	return nil, fmt.Errorf("provider not supported for %s", url)
}

func gitAuthHeaderArgs(host, username, token string) []string {
	if token == "" {
		return nil
	}
	header := "AUTHORIZATION: basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+token))
	return []string{"-c", "http.https://" + host + "/.extraheader=" + header}
}
