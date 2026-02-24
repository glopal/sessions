package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/glopal/sessions/internal/root"
	"github.com/glopal/sessions/internal/session"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context [files...]",
	Short: "Build a context bundle for a file or topic",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runContext,
}

var (
	contextDeep   bool
	contextFormat string
)

func init() {
	contextCmd.Flags().BoolVar(&contextDeep, "deep", false, "Include deep doc bodies in output")
	contextCmd.Flags().StringVar(&contextFormat, "format", "markdown", "Output format: markdown or json")
	rootCmd.AddCommand(contextCmd)
}

func runContext(cmd *cobra.Command, args []string) error {
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

	if contextFormat == "json" {
		return outputContextJSON(sessionsDir, sessions, args)
	}
	return outputContextMarkdown(sessionsDir, sessions, args)
}

func outputContextMarkdown(sessionsDir string, sessions []*session.Session, files []string) error {
	for _, file := range files {
		fmt.Printf("# Context: %s\n\n", file)

		found := false
		for _, s := range sessions {
			for _, fc := range s.FilesChanged {
				if fc.Path == file {
					found = true
					summary := extractSummaryLine(s.Body)
					fmt.Printf("## %s — %s\n", s.SessionID, summary)
					fmt.Printf("- **Action:** %s\n", fc.Action)
					fmt.Printf("- **Change:** %s\n", fc.Summary)
					if len(s.Tags) > 0 {
						fmt.Printf("- **Tags:** %s\n", strings.Join(s.Tags, ", "))
					}

					for _, doc := range s.Docs {
						status := ""
						dd, err := loadDeepDoc(sessionsDir, s.SessionID, doc.Path)
						if err == nil {
							status = dd.Status
						}
						statusStr := ""
						if status != "" {
							statusStr = fmt.Sprintf(" (%s)", status)
						}
						fmt.Printf("- **%s:** %s%s\n", capitalize(doc.Type), doc.Path, statusStr)

						if contextDeep && dd != nil && dd.Body != "" {
							fmt.Printf("\n### %s\n\n%s\n\n", dd.Title, dd.Body)
						}
					}
					fmt.Println()
					break
				}
			}
		}

		if !found {
			fmt.Printf("No sessions found for %s\n\n", file)
		}
	}
	return nil
}

type contextJSONOutput struct {
	File     string               `json:"file"`
	Sessions []contextJSONSession `json:"sessions"`
}

type contextJSONSession struct {
	SessionID   string           `json:"session_id"`
	Timestamp   string           `json:"timestamp"`
	Summary     string           `json:"summary"`
	FileAction  string           `json:"file_action"`
	FileSummary string           `json:"file_summary"`
	Tags        []string         `json:"tags"`
	Docs        []contextJSONDoc `json:"docs,omitempty"`
}

type contextJSONDoc struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

func outputContextJSON(sessionsDir string, sessions []*session.Session, files []string) error {
	var outputs []contextJSONOutput
	for _, file := range files {
		output := contextJSONOutput{
			File:     file,
			Sessions: []contextJSONSession{},
		}

		for _, s := range sessions {
			for _, fc := range s.FilesChanged {
				if fc.Path == file {
					cs := contextJSONSession{
						SessionID:   s.SessionID,
						Timestamp:   s.Timestamp.Format("2006-01-02T15:04:05-07:00"),
						Summary:     extractSummaryLine(s.Body),
						FileAction:  fc.Action,
						FileSummary: fc.Summary,
						Tags:        s.Tags,
					}

					for _, doc := range s.Docs {
						cd := contextJSONDoc{
							Path:    doc.Path,
							Type:    doc.Type,
							Summary: doc.Summary,
						}
						dd, err := loadDeepDoc(sessionsDir, s.SessionID, doc.Path)
						if err == nil {
							cd.Status = dd.Status
						}
						cs.Docs = append(cs.Docs, cd)
					}

					output.Sessions = append(output.Sessions, cs)
					break
				}
			}
		}

		outputs = append(outputs, output)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if len(outputs) == 1 {
		return enc.Encode(outputs[0])
	}
	return enc.Encode(outputs)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
