package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glopal/sessions/internal/parser"
	"github.com/glopal/sessions/internal/root"
	"github.com/glopal/sessions/internal/session"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link [session1] [session2]",
	Short: "Manage related_sessions links",
	RunE:  runLink,
}

var linkAuto bool

func init() {
	linkCmd.Flags().BoolVar(&linkAuto, "auto", false, "Auto-link sessions that share files")
	rootCmd.AddCommand(linkCmd)
}

func runLink(cmd *cobra.Command, args []string) error {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return err
	}

	if err := ensureSessionsDir(sessionsDir); err != nil {
		return err
	}

	if linkAuto {
		return autoLink(sessionsDir)
	}

	if len(args) != 2 {
		return fmt.Errorf("provide exactly two session IDs, or use --auto")
	}

	return manualLink(sessionsDir, args[0], args[1])
}

func manualLink(sessionsDir, id1, id2 string) error {
	file1 := filepath.Join(sessionsDir, id1+".md")
	file2 := filepath.Join(sessionsDir, id2+".md")

	s1, err := parser.ParseSessionFile(file1)
	if err != nil {
		return fmt.Errorf("parsing session %s: %w", id1, err)
	}
	s2, err := parser.ParseSessionFile(file2)
	if err != nil {
		return fmt.Errorf("parsing session %s: %w", id2, err)
	}

	addRelated(s1, id2)
	addRelated(s2, id1)

	if err := parser.WriteSessionFile(file1, s1); err != nil {
		return fmt.Errorf("writing session %s: %w", id1, err)
	}
	if err := parser.WriteSessionFile(file2, s2); err != nil {
		return fmt.Errorf("writing session %s: %w", id2, err)
	}

	fmt.Printf("Linked %s <-> %s\n", id1, id2)
	return nil
}

func autoLink(sessionsDir string) error {
	sessions, err := loadAllSessions(sessionsDir)
	if err != nil {
		return err
	}

	// Build file -> session ID map
	fileMap := make(map[string][]string)
	for _, s := range sessions {
		for _, f := range s.FilesChanged {
			fileMap[f.Path] = append(fileMap[f.Path], s.SessionID)
		}
	}

	// Find sessions that share files
	links := make(map[string]map[string]bool)
	for _, ids := range fileMap {
		if len(ids) < 2 {
			continue
		}
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				if links[ids[i]] == nil {
					links[ids[i]] = make(map[string]bool)
				}
				if links[ids[j]] == nil {
					links[ids[j]] = make(map[string]bool)
				}
				links[ids[i]][ids[j]] = true
				links[ids[j]][ids[i]] = true
			}
		}
	}

	// Build session map for quick lookup
	sessionMap := make(map[string]*session.Session)
	for _, s := range sessions {
		sessionMap[s.SessionID] = s
	}

	// Apply links
	count := 0
	for id, related := range links {
		s := sessionMap[id]
		if s == nil {
			continue
		}
		changed := false
		for relID := range related {
			if addRelated(s, relID) {
				changed = true
			}
		}
		if changed {
			file := filepath.Join(sessionsDir, id+".md")
			if err := parser.WriteSessionFile(file, s); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to update %s: %v\n", id, err)
				continue
			}
			count++
		}
	}

	fmt.Printf("Auto-linked %d sessions\n", count)
	return nil
}

func addRelated(s *session.Session, id string) bool {
	for _, existing := range s.RelatedSessions {
		if existing == id {
			return false
		}
	}
	s.RelatedSessions = append(s.RelatedSessions, id)
	return true
}
