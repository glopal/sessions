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

// loadAllSessions reads and parses all session files from .sessions/sessions/ recursively.
func loadAllSessions(sessionsDir string) ([]*session.Session, error) {
	sessionsSubDir := filepath.Join(sessionsDir, "sessions")
	ymDirs, err := os.ReadDir(sessionsSubDir)
	if err != nil {
		return nil, fmt.Errorf("reading sessions directory: %w", err)
	}

	var sessions []*session.Session
	for _, ym := range ymDirs {
		if !ym.IsDir() {
			continue
		}
		ymPath := filepath.Join(sessionsSubDir, ym.Name())
		entries, err := os.ReadDir(ymPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", ym.Name(), err)
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			path := filepath.Join(ymPath, e.Name())
			s, err := parser.ParseSessionFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", e.Name(), err)
				continue
			}
			sessions = append(sessions, s)
		}
	}

	// Sort by session ID descending (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].SessionID > sessions[j].SessionID
	})

	return sessions, nil
}

// loadArtifact loads and parses an artifact from the artifacts subdirectory.
func loadArtifact(sessionsDir, sessionID, artifactPath string) (*session.Artifact, error) {
	fullPath := filepath.Join(session.ResolveArtifactDir(sessionsDir, sessionID), artifactPath)
	return parser.ParseArtifactFile(fullPath)
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
