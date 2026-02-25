package cmd

import (
	"fmt"
	"io"
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
	Long: `Create an artifact attached to a session. Three modes:

  sessions artifact foo              Print a HEREDOC template to stdout (does not write a file)
  sessions artifact foo <<ART        Read artifact content from stdin and write file
  ...
  ART
  sessions artifact foo --import f   Body from file f (keeps original)
  sessions artifact foo --ingest f   Body from file f (deletes original after write)`,
	Args: cobra.ExactArgs(1),
	RunE: runArtifact,
}

var (
	artifactSession string
	artifactType    string
	artifactImport  string
	artifactIngest  string
)

func init() {
	artifactCmd.Flags().StringVar(&artifactSession, "session", "", "Session ID to attach to (default: most recent)")
	artifactCmd.Flags().StringVar(&artifactType, "type", "analysis", "Artifact type: decision, analysis, investigation, architecture, debug-log")
	artifactCmd.Flags().StringVar(&artifactImport, "import", "", "File path to import body from (keeps original)")
	artifactCmd.Flags().StringVar(&artifactIngest, "ingest", "", "File path to ingest body from (deletes original after write)")
	rootCmd.AddCommand(artifactCmd)
}

func runArtifact(cmd *cobra.Command, args []string) error {
	if artifactImport != "" && artifactIngest != "" {
		return fmt.Errorf("--import and --ingest are mutually exclusive")
	}
	if artifactImport != "" || artifactIngest != "" {
		return runArtifactFromFile(cmd, args[0])
	}
	if isStdinPiped() {
		return runArtifactFromStdin(cmd, args[0])
	}
	return runArtifactTemplate(args[0])
}

// runArtifactTemplate prints a HEREDOC template to stdout. Does NOT write a file.
func runArtifactTemplate(name string) error {
	sessionsDir, err := getSessionsDir()
	if err != nil {
		return err
	}
	sessionID, err := resolveSessionID(sessionsDir)
	if err != nil {
		return err
	}

	artifactName := ensureMD(name)
	title := titleFromName(artifactName)

	template := buildArtifactHeredocTemplate(artifactName, sessionID, artifactType, title)
	fmt.Print(template)
	return nil
}

// runArtifactFromStdin reads artifact content from stdin, parses it, and writes a file.
func runArtifactFromStdin(cmd *cobra.Command, name string) error {
	sessionsDir, err := getSessionsDir()
	if err != nil {
		return err
	}
	sessionID, err := resolveSessionID(sessionsDir)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return fmt.Errorf("empty input; provide artifact content via stdin")
	}

	a, err := parser.ParseArtifact(string(data))
	if err != nil {
		return fmt.Errorf("parsing stdin: %w", err)
	}

	// --type flag overrides stdin type if explicitly set
	if cmd.Flags().Changed("type") {
		a.Type = artifactType
	}

	artifactName := ensureMD(name)
	path, err := writeArtifact(sessionsDir, sessionID, artifactName, a)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// runArtifactFromFile reads body from a file, optional frontmatter from stdin.
func runArtifactFromFile(cmd *cobra.Command, name string) error {
	// Determine source file
	sourceFile := artifactImport
	if sourceFile == "" {
		sourceFile = artifactIngest
	}

	// Validate source file exists before doing anything
	bodyData, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("reading source file: %w", err)
	}

	sessionsDir, err := getSessionsDir()
	if err != nil {
		return err
	}
	sessionID, err := resolveSessionID(sessionsDir)
	if err != nil {
		return err
	}

	artifactName := ensureMD(name)
	var a *session.Artifact

	// If stdin is piped, parse frontmatter from it
	if isStdinPiped() {
		stdinData, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		if len(strings.TrimSpace(string(stdinData))) > 0 {
			a, err = parser.ParseArtifact(string(stdinData))
			if err != nil {
				return fmt.Errorf("parsing stdin frontmatter: %w", err)
			}
			// Override body with file content
			a.Body = strings.TrimSpace(string(bodyData))
		}
	}

	// If no stdin frontmatter, build defaults
	if a == nil {
		a = &session.Artifact{
			Title:   titleFromName(artifactName),
			Type:    artifactType,
			Summary: "",
			Status:  "draft",
			Body:    strings.TrimSpace(string(bodyData)),
		}
	}

	// --type flag overrides if explicitly set
	if cmd.Flags().Changed("type") {
		a.Type = artifactType
	}

	path, err := writeArtifact(sessionsDir, sessionID, artifactName, a)
	if err != nil {
		return err
	}

	// --ingest: delete original only after successful write
	if artifactIngest != "" {
		if err := os.Remove(sourceFile); err != nil {
			return fmt.Errorf("deleting source file after ingest: %w", err)
		}
	}

	fmt.Println(path)
	return nil
}

// buildArtifactHeredocTemplate builds the HEREDOC template string for stdout.
func buildArtifactHeredocTemplate(name, sessionID, artifactType, title string) string {
	var b strings.Builder

	b.WriteString("Run the following command with an updated HEREDOC.\n\n")
	fmt.Fprintf(&b, "sessions artifact %s --session %s <<ART\n", strings.TrimSuffix(name, ".md"), sessionID)
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %s\n", title)
	fmt.Fprintf(&b, "type: %s\n", artifactType)
	b.WriteString("summary: \"\"\n")
	b.WriteString("status: draft\n")
	b.WriteString("supersedes: \"\"\n")
	b.WriteString("---\n\n")
	b.WriteString("Content goes here.\n")
	b.WriteString("ART\n")

	return b.String()
}

// titleFromName derives a title from an artifact filename.
// "sessions-new-spec.md" → "Sessions New Spec"
func titleFromName(name string) string {
	base := strings.TrimSuffix(filepath.Base(name), ".md")
	spaced := strings.ReplaceAll(base, "-", " ")
	return titleCase(spaced)
}

// writeArtifact writes an artifact file and updates the parent session's artifacts list.
func writeArtifact(sessionsDir, sessionID, name string, a *session.Artifact) (string, error) {
	// Validate session file exists
	sessionFile := session.ResolveSessionPath(sessionsDir, sessionID)
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		return "", fmt.Errorf("session file not found: %s", sessionFile)
	}

	// Create artifact subdirectory
	artifactDir := session.ResolveArtifactDir(sessionsDir, sessionID)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return "", fmt.Errorf("creating artifact subdirectory: %w", err)
	}

	// Write artifact file
	artifactPath := filepath.Join(artifactDir, name)
	if err := parser.WriteArtifactFile(artifactPath, a); err != nil {
		return "", fmt.Errorf("writing artifact: %w", err)
	}

	// Update parent session's artifacts list
	if err := addArtifactToSession(sessionFile, name, a); err != nil {
		return "", fmt.Errorf("updating session file: %w", err)
	}

	return artifactPath, nil
}

// resolveSessionID resolves the --session flag or finds the most recent session.
func resolveSessionID(sessionsDir string) (string, error) {
	if artifactSession != "" {
		return artifactSession, nil
	}
	return findMostRecentSession(sessionsDir)
}

// getSessionsDir returns the sessions directory, ensuring it exists.
func getSessionsDir() (string, error) {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return "", err
	}
	if err := ensureSessionsDir(sessionsDir); err != nil {
		return "", err
	}
	return sessionsDir, nil
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

func addArtifactToSession(sessionFile, artifactName string, a *session.Artifact) error {
	s, err := parser.ParseSessionFile(sessionFile)
	if err != nil {
		return err
	}

	s.Artifacts = append(s.Artifacts, session.ArtifactRef{
		Path:    artifactName,
		Type:    a.Type,
		Summary: a.Summary,
	})

	return parser.WriteSessionFile(sessionFile, s)
}

// ensureMD appends .md to a name if not already present.
func ensureMD(name string) string {
	if !strings.HasSuffix(name, ".md") {
		return name + ".md"
	}
	return name
}
