package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/glopal/sessions/internal/parser"
	"github.com/glopal/sessions/internal/root"
	"github.com/glopal/sessions/internal/session"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Scaffold a new session file",
	RunE:  runNew,
}

var (
	newTags  string
	gitDiff  bool
	diffBase string
)

func init() {
	newCmd.Flags().StringVar(&newTags, "tags", "", "Comma-separated tags")
	newCmd.Flags().BoolVar(&gitDiff, "git-diff", false, "Auto-populate files_changed from git diff")
	newCmd.Flags().StringVar(&diffBase, "base", "HEAD", "Base ref for git diff (used with --git-diff)")
	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return err
	}

	if err := ensureSessionsDir(sessionsDir); err != nil {
		return err
	}

	now := time.Now()
	sessionID := now.Format("2006-01-02_1504")

	var tags []string
	if newTags != "" {
		for _, t := range strings.Split(newTags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	var files []session.FileChange
	if gitDiff {
		files, err = getGitDiffFiles(diffBase)
		if err != nil {
			return fmt.Errorf("getting git diff: %w", err)
		}
	}

	s := &session.Session{
		Timestamp:    now,
		SessionID:    sessionID,
		Tags:         tags,
		FilesChanged: files,
	}

	s.Body = `## Key Decisions

- Decision 1 and rationale.

## Open Questions

- Anything unresolved that future sessions should be aware of.`

	filePath := filepath.Join(sessionsDir, sessionID+".md")
	if err := parser.WriteSessionFile(filePath, s); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	fmt.Println(filePath)
	return nil
}

func getGitDiffFiles(base string) ([]session.FileChange, error) {
	out, err := exec.Command("git", "diff", "--name-status", base).Output()
	if err != nil {
		return nil, fmt.Errorf("running git diff: %w", err)
	}

	var files []session.FileChange
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		path := parts[1]

		var action string
		switch {
		case strings.HasPrefix(status, "A"):
			action = "added"
		case strings.HasPrefix(status, "M"):
			action = "modified"
		case strings.HasPrefix(status, "D"):
			action = "deleted"
		case strings.HasPrefix(status, "R"):
			action = "renamed"
			if len(parts) >= 3 {
				path = parts[2] // use the new name for renames
			}
		default:
			action = "modified"
		}

		files = append(files, session.FileChange{
			Path:    path,
			Action:  action,
			Summary: "",
		})
	}
	return files, nil
}
