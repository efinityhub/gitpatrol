package models

import "time"

type Repository struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	IntervalMinutes int       `json:"interval_minutes"`
	LastSync        time.Time `json:"last_sync"`
	Status          string    `json:"status"`
	LastCommit      string    `json:"last_commit"`
	ErrorMessage    string    `json:"error_message"`
	Stars           int       `json:"stars"`
	Forks           int       `json:"forks"`
	OpenIssues      int       `json:"open_issues"`
	CommitHistory   string    `json:"commit_history"`
	HealthScore     int       `json:"health_score"`
	DefaultBranch   string    `json:"default_branch"`
	AutoPatrol      int       `json:"auto_patrol"`
	SizeBytes       int64     `json:"size_bytes"`
}

type Metadata struct {
	Stars      int
	Forks      int
	OpenIssues int
	AvatarURL  string
	Username   string
}
