package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/glopal/sessions/internal/parser"
	"github.com/glopal/sessions/internal/session"
)

func ensureSessionsDir(sessionsDir string) error {
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return fmt.Errorf(".sessions/ directory not found; run 'sessions init' first")
	}
	return nil
}

// loadAllSessions reads and parses all session files from the sessions directory.
func loadAllSessions(sessionsDir string) ([]*session.Session, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("reading sessions directory: %w", err)
	}

	var sessions []*session.Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(sessionsDir, e.Name())
		s, err := parser.ParseSessionFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", e.Name(), err)
			continue
		}
		sessions = append(sessions, s)
	}

	// Sort by session ID descending (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].SessionID > sessions[j].SessionID
	})

	return sessions, nil
}

// loadDeepDoc loads and parses a deep doc from a session subdirectory.
func loadDeepDoc(sessionsDir, sessionID, docPath string) (*session.DeepDoc, error) {
	fullPath := filepath.Join(sessionsDir, sessionID, docPath)
	return parser.ParseDeepDocFile(fullPath)
}

// extractSummaryLine extracts the first sentence/paragraph from the body after "## Summary".
func extractSummaryLine(body string) string {
	lines := strings.Split(body, "\n")
	inSummary := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Summary" {
			inSummary = true
			continue
		}
		if inSummary {
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "## ") {
				break
			}
			// Return first non-empty line after ## Summary
			if len(trimmed) > 80 {
				return trimmed[:77] + "..."
			}
			return trimmed
		}
	}
	// Fallback: return first non-empty, non-heading line
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "---") {
			if len(trimmed) > 80 {
				return trimmed[:77] + "..."
			}
			return trimmed
		}
	}
	return "(no summary)"
}

// titleCase capitalizes the first letter of each word.
func titleCase(s string) string {
	prev := ' '
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(rune(prev)) {
			prev = r
			return unicode.ToTitle(r)
		}
		prev = r
		return r
	}, s)
}
