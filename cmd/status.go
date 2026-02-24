package cmd

import (
	"fmt"
	"os"

	"github.com/glopal/sessions/internal/root"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of deep docs, flagging stale decisions",
	RunE:  runStatus,
}

var (
	statusDocType string
	statusStale   bool
)

func init() {
	statusCmd.Flags().StringVar(&statusDocType, "doc-type", "", "Filter by doc type")
	statusCmd.Flags().BoolVar(&statusStale, "stale", false, "Show only superseded or deprecated docs")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	found := false
	for _, s := range sessions {
		for _, doc := range s.Docs {
			// Filter by doc type
			if statusDocType != "" && doc.Type != statusDocType {
				continue
			}

			dd, err := loadDeepDoc(sessionsDir, s.SessionID, doc.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not load %s/%s: %v\n", s.SessionID, doc.Path, err)
				continue
			}

			// Filter stale only
			if statusStale && dd.Status != "superseded" && dd.Status != "deprecated" {
				continue
			}

			found = true
			statusIcon := statusIndicator(dd.Status)
			fmt.Printf("%s %s  %s/%s  [%s]", statusIcon, dd.Status, s.SessionID, doc.Path, doc.Type)
			if dd.Title != "" {
				fmt.Printf("  %s", dd.Title)
			}
			if dd.Supersedes != "" {
				fmt.Printf("  (supersedes: %s)", dd.Supersedes)
			}
			fmt.Println()
		}
	}

	if !found {
		fmt.Println("No matching docs found.")
		os.Exit(2)
	}

	return nil
}

func statusIndicator(status string) string {
	switch status {
	case "accepted":
		return "+"
	case "draft":
		return "~"
	case "superseded":
		return "!"
	case "deprecated":
		return "x"
	default:
		return "?"
	}
}
