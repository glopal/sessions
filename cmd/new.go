package cmd

import (
	"fmt"
	"io"
	"os"
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
	Short: "Create a new session file",
	Long: `Create a new session file. Three modes:

  sessions new              Print a HEREDOC template to stdout (does not write a file)
  sessions new --empty      Create an empty stub session file
  sessions new <<SESS       Read session content from stdin and write file
  ...
  SESS`,
	RunE: runNew,
}

var (
	newTags  string
	newEmpty bool
)

func init() {
	newCmd.Flags().StringVar(&newTags, "tags", "", "Comma-separated tags")
	newCmd.Flags().BoolVar(&newEmpty, "empty", false, "Create an empty stub session file")
	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	if newEmpty {
		return runNewEmpty()
	}
	if isStdinPiped() {
		return runNewFromStdin()
	}
	return runNewTemplate()
}

// isStdinPiped returns true if stdin is not a terminal (i.e., data is piped in).
func isStdinPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

// runNewEmpty creates a stub session file with empty summary and optional tags.
func runNewEmpty() error {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return err
	}
	if err := ensureSessionsDir(sessionsDir); err != nil {
		return err
	}

	now := time.Now()
	tags := parseTags(newTags)
	s := buildSystemSession(now, tags)
	s.Body = defaultBody()

	path, err := writeSession(sessionsDir, now, s)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// runNewTemplate prints a HEREDOC template to stdout with git status files.
func runNewTemplate() error {
	tags := parseTags(newTags)

	files, err := getGitStatusFiles()
	if err != nil {
		// Git not available or not a repo — continue with empty files
		files = nil
	}

	template := buildHeredocTemplate(tags, files)
	fmt.Print(template)
	return nil
}

// runNewFromStdin reads session content from stdin, parses it, and writes a file.
func runNewFromStdin() error {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return err
	}
	if err := ensureSessionsDir(sessionsDir); err != nil {
		return err
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return fmt.Errorf("empty input; provide session content via stdin")
	}

	stdinSession, err := parser.ParseSession(string(data))
	if err != nil {
		return fmt.Errorf("parsing stdin: %w", err)
	}

	now := time.Now()
	flagTags := parseTags(newTags)
	mergedTags := mergeTags(flagTags, stdinSession.Tags)

	// System-managed fields — always set by CLI
	s := buildSystemSession(now, mergedTags)

	// Claude-settable fields — copied from parsed stdin
	s.Summary = stdinSession.Summary
	s.FilesChanged = stdinSession.FilesChanged
	s.Body = stdinSession.Body

	path, err := writeSession(sessionsDir, now, s)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// parseTags splits a comma-separated tag string into a slice, trimming whitespace.
func parseTags(tagStr string) []string {
	if tagStr == "" {
		return nil
	}
	var tags []string
	for _, t := range strings.Split(tagStr, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// mergeTags returns the deduplicated union of two tag slices, preserving order
// (flagTags first, then stdinTags that aren't already present).
func mergeTags(flagTags, stdinTags []string) []string {
	if len(flagTags) == 0 && len(stdinTags) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var merged []string
	for _, t := range flagTags {
		if !seen[t] {
			seen[t] = true
			merged = append(merged, t)
		}
	}
	for _, t := range stdinTags {
		if !seen[t] {
			seen[t] = true
			merged = append(merged, t)
		}
	}
	return merged
}

// buildSystemSession creates a Session with system-managed fields populated.
func buildSystemSession(now time.Time, tags []string) *session.Session {
	return &session.Session{
		Timestamp:       now,
		SessionID:       fmt.Sprintf("%d", now.Unix()),
		Tags:            tags,
		Artifacts:       []session.ArtifactRef{},
		RelatedSessions: []string{},
	}
}

// writeSession writes a session file to the appropriate year-month subdirectory.
func writeSession(sessionsDir string, now time.Time, s *session.Session) (string, error) {
	yearMonth := now.UTC().Format("2006-01")
	sessionSubDir := filepath.Join(sessionsDir, "sessions", yearMonth)
	if err := os.MkdirAll(sessionSubDir, 0755); err != nil {
		return "", fmt.Errorf("creating session subdirectory: %w", err)
	}

	filePath := filepath.Join(sessionSubDir, s.SessionID+".md")
	if err := parser.WriteSessionFile(filePath, s); err != nil {
		return "", fmt.Errorf("writing session file: %w", err)
	}
	return filePath, nil
}

// defaultBody returns the scaffold body template for new sessions.
func defaultBody() string {
	return `## Key Decisions

- Decision 1 and rationale.

## Open Questions

- Anything unresolved that future sessions should be aware of.`
}

// getGitStatusFiles runs git status --porcelain and parses the output.
func getGitStatusFiles() ([]session.FileChange, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("running git status: %w", err)
	}
	return parseGitStatusOutput(string(out)), nil
}

// parseGitStatusOutput parses the output of git status --porcelain into FileChanges.
func parseGitStatusOutput(output string) []session.FileChange {
	var files []session.FileChange
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 4 {
			continue
		}
		// Porcelain format: XY PATH or XY PATH -> NEWPATH
		xy := line[:2]
		path := strings.TrimSpace(line[3:])
		if path == "" {
			continue
		}

		var action string
		switch {
		case xy == "??":
			action = "added"
		case strings.ContainsRune(xy, 'D'):
			action = "deleted"
		case strings.ContainsRune(xy, 'A'):
			action = "added"
		case strings.ContainsRune(xy, 'R'):
			action = "renamed"
			// Renames show as "old -> new"
			if idx := strings.Index(path, " -> "); idx >= 0 {
				path = path[idx+4:]
			}
		case strings.ContainsRune(xy, 'M'):
			action = "modified"
		default:
			action = "modified"
		}

		files = append(files, session.FileChange{
			Path:    path,
			Action:  action,
			Summary: "TODO",
		})
	}
	return files
}

// buildHeredocTemplate builds the HEREDOC template string for stdout.
func buildHeredocTemplate(tags []string, files []session.FileChange) string {
	var b strings.Builder

	b.WriteString("Run the following command with an updated HEREDOC.\n\n")
	b.WriteString("sessions new <<SESS\n")
	b.WriteString("---\n")
	b.WriteString("summary: \"\"\n")

	// Tags
	if len(tags) == 0 {
		b.WriteString("tags: []\n")
	} else {
		b.WriteString("tags:\n")
		for _, t := range tags {
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	}

	// Files changed
	if len(files) == 0 {
		b.WriteString("files_changed: []\n")
	} else {
		b.WriteString("files_changed:\n")
		for _, f := range files {
			fmt.Fprintf(&b, "  - path: %s\n", f.Path)
			fmt.Fprintf(&b, "    action: %s\n", f.Action)
			fmt.Fprintf(&b, "    summary: %s\n", f.Summary)
		}
	}

	b.WriteString("---\n\n")
	b.WriteString("## Key Decisions\n\n")
	b.WriteString("- Decision 1 and rationale.\n\n")
	b.WriteString("## Open Questions\n\n")
	b.WriteString("- Anything unresolved that future sessions should be aware of.\n")
	b.WriteString("SESS\n")

	return b.String()
}
