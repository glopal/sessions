package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/glopal/sessions/internal/root"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions, optionally filtered",
	RunE:  runList,
}

var (
	listTag     string
	listVerbose bool
)

func init() {
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	listCmd.Flags().BoolVar(&listVerbose, "verbose", false, "Show file counts")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return err
	}

	if err := ensureSessionsDir(sessionsDir); err != nil {
		return err
	}

	sessions, err := loadAllSessions(sessionsDir)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		os.Exit(2)
	}

	for _, s := range sessions {
		// Tag filter
		if listTag != "" {
			found := false
			for _, t := range s.Tags {
				if t == listTag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		summary := s.Summary
		if summary == "" {
			summary = "(no summary)"
		}
		line := fmt.Sprintf("%s  %s", s.SessionID, summary)

		if listVerbose {
			line += fmt.Sprintf("  [files: %d, artifacts: %d]", len(s.FilesChanged), len(s.Artifacts))
		}

		if len(s.Tags) > 0 {
			line += fmt.Sprintf("  [%s]", strings.Join(s.Tags, ", "))
		}

		fmt.Println(line)
	}
	return nil
}
