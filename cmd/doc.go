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

var docCmd = &cobra.Command{
	Use:   "doc [name]",
	Short: "Create a deep doc attached to a session",
	Args:  cobra.ExactArgs(1),
	RunE:  runDoc,
}

var (
	docSession string
	docType    string
)

func init() {
	docCmd.Flags().StringVar(&docSession, "session", "", "Session ID to attach to (default: most recent)")
	docCmd.Flags().StringVar(&docType, "type", "analysis", "Doc type: decision, analysis, investigation, architecture, debug-log")
	rootCmd.AddCommand(docCmd)
}

func runDoc(cmd *cobra.Command, args []string) error {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return err
	}

	if err := ensureSessionsDir(sessionsDir); err != nil {
		return err
	}

	docName := args[0]
	if !strings.HasSuffix(docName, ".md") {
		docName += ".md"
	}

	// Determine session ID
	sessionID := docSession
	if sessionID == "" {
		sessionID, err = findMostRecentSession(sessionsDir)
		if err != nil {
			return err
		}
	}

	// Validate session file exists
	sessionFile := filepath.Join(sessionsDir, sessionID+".md")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		return fmt.Errorf("session file not found: %s", sessionFile)
	}

	// Create session subdirectory
	subDir := filepath.Join(sessionsDir, sessionID)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return fmt.Errorf("creating session subdirectory: %w", err)
	}

	// Scaffold deep doc
	docPath := filepath.Join(subDir, docName)
	title := strings.TrimSuffix(filepath.Base(docName), ".md")
	title = strings.ReplaceAll(title, "-", " ")
	title = titleCase(title)

	docContent := fmt.Sprintf(`---
title: %s
type: %s
status: draft
supersedes: null
---

`, title, docType)

	if err := os.WriteFile(docPath, []byte(docContent), 0644); err != nil {
		return fmt.Errorf("writing deep doc: %w", err)
	}

	// Update parent session's docs list
	if err := addDocToSession(sessionFile, docName, docType); err != nil {
		return fmt.Errorf("updating session file: %w", err)
	}

	fmt.Println(docPath)
	return nil
}

func findMostRecentSession(sessionsDir string) (string, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return "", fmt.Errorf("reading sessions directory: %w", err)
	}

	var sessionIDs []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		sessionIDs = append(sessionIDs, strings.TrimSuffix(e.Name(), ".md"))
	}

	if len(sessionIDs) == 0 {
		return "", fmt.Errorf("no sessions found; create one with 'sessions new'")
	}

	sort.Strings(sessionIDs)
	return sessionIDs[len(sessionIDs)-1], nil
}

func addDocToSession(sessionFile, docName, docType string) error {
	s, err := parser.ParseSessionFile(sessionFile)
	if err != nil {
		return err
	}

	s.Docs = append(s.Docs, session.DocRef{
		Path:    docName,
		Type:    docType,
		Summary: "",
	})

	return parser.WriteSessionFile(sessionFile, s)
}
