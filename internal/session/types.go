package session

import "time"

const MaxSummaryLength = 150

type Session struct {
	Timestamp       time.Time     `yaml:"timestamp"`
	SessionID       string        `yaml:"session_id"`
	Summary         string        `yaml:"summary"`
	Tags            []string      `yaml:"tags"`
	FilesChanged    []FileChange  `yaml:"files_changed"`
	Artifacts       []ArtifactRef `yaml:"artifacts"`
	RelatedSessions []string      `yaml:"related_sessions"`
	Body            string        `yaml:"-"`
}

type FileChange struct {
	Path    string `yaml:"path"`
	Action  string `yaml:"action"`
	Summary string `yaml:"summary"`
}

type ArtifactRef struct {
	Path    string `yaml:"path"`
	Type    string `yaml:"type"`
	Summary string `yaml:"summary"`
}

type Artifact struct {
	Title      string `yaml:"title"`
	Type       string `yaml:"type"`
	Summary    string `yaml:"summary"`
	Status     string `yaml:"status"`
	Supersedes string `yaml:"supersedes"`
	Body       string `yaml:"-"`
}
