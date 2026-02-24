package cmd

import (
	"fmt"
	"os"

	"github.com/glopal/sessions/internal/root"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of artifacts, flagging stale decisions",
	RunE:  runStatus,
}

var (
	statusArtifactType string
	statusStale        bool
)

func init() {
	statusCmd.Flags().StringVar(&statusArtifactType, "artifact-type", "", "Filter by artifact type")
	statusCmd.Flags().BoolVar(&statusStale, "stale", false, "Show only superseded or deprecated artifacts")
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
		for _, art := range s.Artifacts {
			// Filter by artifact type
			if statusArtifactType != "" && art.Type != statusArtifactType {
				continue
			}

			a, err := loadArtifact(sessionsDir, s.SessionID, art.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not load %s/%s: %v\n", s.SessionID, art.Path, err)
				continue
			}

			// Filter stale only
			if statusStale && a.Status != "superseded" && a.Status != "deprecated" {
				continue
			}

			found = true
			statusIcon := statusIndicator(a.Status)
			fmt.Printf("%s %s  %s/%s  [%s]", statusIcon, a.Status, s.SessionID, art.Path, art.Type)
			if a.Title != "" {
				fmt.Printf("  %s", a.Title)
			}
			if a.Supersedes != "" {
				fmt.Printf("  (supersedes: %s)", a.Supersedes)
			}
			fmt.Println()
		}
	}

	if !found {
		fmt.Println("No matching artifacts found.")
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
