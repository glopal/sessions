package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/glopal/sessions/internal/parser"
	"github.com/glopal/sessions/internal/root"
	"github.com/glopal/sessions/internal/session"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [keys...]",
	Short: "Validate sessions and artifacts",
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

type validationIssue struct {
	Key    string
	Reason string
}

func runValidate(cmd *cobra.Command, args []string) error {
	sessionsDir, err := root.SessionsDir()
	if err != nil {
		return err
	}

	if err := ensureSessionsDir(sessionsDir); err != nil {
		return err
	}

	var issues []validationIssue

	if len(args) == 0 {
		// Validate all sessions and their artifacts
		sessions, err := loadAllSessions(sessionsDir)
		if err != nil {
			return err
		}
		for _, s := range sessions {
			issues = append(issues, validateSessionSummary(s.SessionID, s.Summary)...)
			for _, art := range s.Artifacts {
				key := session.FormatArtifactKey(s.SessionID, art.Path)
				a, err := loadArtifact(sessionsDir, s.SessionID, art.Path)
				if err != nil {
					issues = append(issues, validationIssue{Key: key, Reason: "unreadable"})
					continue
				}
				issues = append(issues, validateArtifactSummary(key, a.Summary)...)
			}
		}
	} else {
		// Validate only specified keys
		for _, key := range args {
			sessionID, _, isArtifact := session.ParseKey(key)
			path := session.ResolveKeyToPath(sessionsDir, key)
			if isArtifact {
				a, err := parser.ParseArtifactFile(path)
				if err != nil {
					issues = append(issues, validationIssue{Key: key, Reason: "unreadable"})
					continue
				}
				issues = append(issues, validateArtifactSummary(key, a.Summary)...)
			} else {
				s, err := parser.ParseSessionFile(path)
				if err != nil {
					issues = append(issues, validationIssue{Key: key, Reason: "unreadable"})
					continue
				}
				issues = append(issues, validateSessionSummary(sessionID, s.Summary)...)
			}
		}
	}

	if len(issues) == 0 {
		fmt.Println("All valid.")
		return nil
	}

	fmt.Println("PROBLEM")
	fmt.Println("Invalid Summary")
	fmt.Println()
	fmt.Println("AFFECTED KEYS")
	for _, issue := range issues {
		fmt.Printf("- %s (%s)\n", issue.Key, issue.Reason)
	}
	fmt.Println()
	fmt.Println("FIX")
	fmt.Printf("sessions edit <KEY> --summary \"<SUMMARY_LTE_%d_CHARS>\"\n", session.MaxSummaryLength)

	os.Exit(1)
	return nil
}

func validateSessionSummary(sessionID, summary string) []validationIssue {
	return validateSummary(session.FormatSessionKey(sessionID), summary)
}

func validateArtifactSummary(key, summary string) []validationIssue {
	return validateSummary(key, summary)
}

func validateSummary(key, summary string) []validationIssue {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return []validationIssue{{Key: key, Reason: "missing"}}
	}
	if len(summary) > session.MaxSummaryLength {
		return []validationIssue{{Key: key, Reason: "too long"}}
	}
	return nil
}
