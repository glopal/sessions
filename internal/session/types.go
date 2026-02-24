package session

import "time"

type Session struct {
	Timestamp       time.Time    `yaml:"timestamp"`
	SessionID       string       `yaml:"session_id"`
	Tags            []string     `yaml:"tags"`
	FilesChanged    []FileChange `yaml:"files_changed"`
	Docs            []DocRef     `yaml:"docs"`
	RelatedSessions []string     `yaml:"related_sessions"`
	Body            string       `yaml:"-"`
}

type FileChange struct {
	Path    string `yaml:"path"`
	Action  string `yaml:"action"`
	Summary string `yaml:"summary"`
}

type DocRef struct {
	Path    string `yaml:"path"`
	Type    string `yaml:"type"`
	Summary string `yaml:"summary"`
}

type DeepDoc struct {
	Title      string `yaml:"title"`
	Type       string `yaml:"type"`
	Status     string `yaml:"status"`
	Supersedes string `yaml:"supersedes"`
	Body       string `yaml:"-"`
}
