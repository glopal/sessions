package cmd

import (
	"fmt"

	"github.com/glopal/sessions/internal/parser"
	"github.com/glopal/sessions/internal/root"
	"github.com/glopal/sessions/internal/session"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <key>",
	Short: "Edit session or artifact fields",
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
}

var editSummary string

func init() {
	editCmd.Flags().StringVar(&editSummary, "summary", "", "Set the summary (max 150 chars)")
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return err
	}

	if err := ensureSessionsDir(sessionsDir); err != nil {
		return err
	}

	key := args[0]

	if !cmd.Flags().Changed("summary") {
		return fmt.Errorf("no fields specified; use --summary to set a value")
	}

	if len(editSummary) > session.MaxSummaryLength {
		return fmt.Errorf("summary exceeds %d characters (%d given)", session.MaxSummaryLength, len(editSummary))
	}

	sessionID, artifactFile, isArtifact := session.ParseKey(key)
	path := session.ResolveKeyToPath(sessionsDir, key)

	if isArtifact {
		a, err := parser.ParseArtifactFile(path)
		if err != nil {
			return fmt.Errorf("loading artifact %s: %w", key, err)
		}
		a.Summary = editSummary
		if err := parser.WriteArtifactFile(path, a); err != nil {
			return fmt.Errorf("writing artifact %s: %w", key, err)
		}
	} else {
		_ = artifactFile // unused in session path
		s, err := parser.ParseSessionFile(path)
		if err != nil {
			return fmt.Errorf("loading session %s: %w", sessionID, err)
		}
		s.Summary = editSummary
		if err := parser.WriteSessionFile(path, s); err != nil {
			return fmt.Errorf("writing session %s: %w", sessionID, err)
		}
	}

	fmt.Printf("Updated %s\n", key)
	return nil
}
