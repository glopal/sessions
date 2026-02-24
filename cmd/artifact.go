package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/glopal/sessions/internal/parser"
	"github.com/glopal/sessions/internal/root"
	"github.com/glopal/sessions/internal/session"
	"github.com/spf13/cobra"
)

var artifactCmd = &cobra.Command{
	Use:   "artifact [name]",
	Short: "Create an artifact attached to a session",
	Args:  cobra.ExactArgs(1),
	RunE:  runArtifact,
}

var (
	artifactSession string
	artifactType    string
)

func init() {
	artifactCmd.Flags().StringVar(&artifactSession, "session", "", "Session ID to attach to (default: most recent)")
	artifactCmd.Flags().StringVar(&artifactType, "type", "analysis", "Artifact type: decision, analysis, investigation, architecture, debug-log")
	rootCmd.AddCommand(artifactCmd)
}

func runArtifact(cmd *cobra.Command, args []string) error {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return err
	}

	if err := ensureSessionsDir(sessionsDir); err != nil {
		return err
	}

	artifactName := args[0]
	if !strings.HasSuffix(artifactName, ".md") {
		artifactName += ".md"
	}

	// Determine session ID
	sessionID := artifactSession
	if sessionID == "" {
		sessionID, err = findMostRecentSession(sessionsDir)
		if err != nil {
			return err
		}
	}

	// Validate session file exists
	sessionFile := session.ResolveSessionPath(sessionsDir, sessionID)
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		return fmt.Errorf("session file not found: %s", sessionFile)
	}

	// Create artifact subdirectory
	artifactDir := session.ResolveArtifactDir(sessionsDir, sessionID)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return fmt.Errorf("creating artifact subdirectory: %w", err)
	}

	// Scaffold artifact
	artifactPath := filepath.Join(artifactDir, artifactName)
	title := strings.TrimSuffix(filepath.Base(artifactName), ".md")
	title = strings.ReplaceAll(title, "-", " ")
	title = titleCase(title)

	artifactContent := fmt.Sprintf(`---
title: %s
type: %s
summary: ""
status: draft
supersedes: null
---

`, title, artifactType)

	if err := os.WriteFile(artifactPath, []byte(artifactContent), 0644); err != nil {
		return fmt.Errorf("writing artifact: %w", err)
	}

	// Update parent session's artifacts list
	if err := addArtifactToSession(sessionFile, artifactName, artifactType); err != nil {
		return fmt.Errorf("updating session file: %w", err)
	}

	fmt.Println(artifactPath)
	return nil
}

func findMostRecentSession(sessionsDir string) (string, error) {
	sessionsSubDir := filepath.Join(sessionsDir, "sessions")
	ymDirs, err := os.ReadDir(sessionsSubDir)
	if err != nil {
		return "", fmt.Errorf("reading sessions directory: %w", err)
	}

	var sessionIDs []string
	for _, ym := range ymDirs {
		if !ym.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(sessionsSubDir, ym.Name()))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			sessionIDs = append(sessionIDs, strings.TrimSuffix(e.Name(), ".md"))
		}
	}

	if len(sessionIDs) == 0 {
		return "", fmt.Errorf("no sessions found; create one with 'sessions new'")
	}

	sort.Strings(sessionIDs)
	return sessionIDs[len(sessionIDs)-1], nil
}

func addArtifactToSession(sessionFile, artifactName, artifactType string) error {
	s, err := parser.ParseSessionFile(sessionFile)
	if err != nil {
		return err
	}

	s.Artifacts = append(s.Artifacts, session.ArtifactRef{
		Path:    artifactName,
		Type:    artifactType,
		Summary: "",
	})

	return parser.WriteSessionFile(sessionFile, s)
}
