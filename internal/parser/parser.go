package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/glopal/sessions/internal/session"
	"gopkg.in/yaml.v3"
)

// ParseSessionFile parses a session markdown file with YAML frontmatter.
func ParseSessionFile(path string) (*session.Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}
	return ParseSession(string(data))
}

// ParseSession parses a session from raw content string.
func ParseSession(content string) (*session.Session, error) {
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}
	var s session.Session
	if err := yaml.Unmarshal([]byte(fm), &s); err != nil {
		return nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}
	s.Body = body
	return &s, nil
}

// ParseDeepDocFile parses a deep doc markdown file with YAML frontmatter.
func ParseDeepDocFile(path string) (*session.DeepDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading deep doc file: %w", err)
	}
	return ParseDeepDoc(string(data))
}

// ParseDeepDoc parses a deep doc from raw content string.
func ParseDeepDoc(content string) (*session.DeepDoc, error) {
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}
	var d session.DeepDoc
	if err := yaml.Unmarshal([]byte(fm), &d); err != nil {
		return nil, fmt.Errorf("parsing deep doc frontmatter: %w", err)
	}
	d.Body = body
	return &d, nil
}

// splitFrontmatter splits content into YAML frontmatter and markdown body.
func splitFrontmatter(content string) (string, string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("file does not start with frontmatter delimiter '---'")
	}

	// Find the closing ---
	rest := content[3:]
	// Skip the newline after opening ---
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}

	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", "", fmt.Errorf("no closing frontmatter delimiter '---' found")
	}

	fm := rest[:idx]
	body := rest[idx+4:] // skip past \n---

	// Skip newline after closing ---
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	} else if len(body) > 1 && body[0] == '\r' && body[1] == '\n' {
		body = body[2:]
	}

	return fm, strings.TrimSpace(body), nil
}

// SerializeSessionFrontmatter serializes session frontmatter to YAML.
func SerializeSessionFrontmatter(s *session.Session) (string, error) {
	data, err := yaml.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("marshaling session frontmatter: %w", err)
	}
	return string(data), nil
}

// WriteSessionFile writes a session to a file with frontmatter and body.
func WriteSessionFile(path string, s *session.Session) error {
	fm, err := SerializeSessionFrontmatter(s)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", fm, s.Body)
	return os.WriteFile(path, []byte(content), 0644)
}
