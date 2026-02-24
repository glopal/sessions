package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/glopal/sessions/internal/root"
	"github.com/glopal/sessions/internal/session"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Search sessions by various criteria",
	RunE:  runQuery,
}

var (
	queryFile    string
	queryTag     string
	queryDocType string
	queryAfter   string
	queryBefore  string
	querySearch  string
	queryLimit   int
	queryFormat  string
)

func init() {
	queryCmd.Flags().StringVar(&queryFile, "file", "", "Filter by file path (exact or glob)")
	queryCmd.Flags().StringVar(&queryTag, "tag", "", "Filter by tag")
	queryCmd.Flags().StringVar(&queryDocType, "doc-type", "", "Filter by doc type")
	queryCmd.Flags().StringVar(&queryAfter, "after", "", "Filter sessions after date (YYYY-MM-DD)")
	queryCmd.Flags().StringVar(&queryBefore, "before", "", "Filter sessions before date (YYYY-MM-DD)")
	queryCmd.Flags().StringVar(&querySearch, "search", "", "Full-text search across session bodies")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 0, "Limit number of results")
	queryCmd.Flags().StringVar(&queryFormat, "format", "text", "Output format: text or json")
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
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

	var results []*queryResult
	for _, s := range sessions {
		r := matchSession(s)
		if r != nil {
			results = append(results, r)
		}
	}

	if len(results) == 0 {
		fmt.Println("No matching sessions found.")
		os.Exit(2)
	}

	if queryLimit > 0 && len(results) > queryLimit {
		results = results[:queryLimit]
	}

	if queryFormat == "json" {
		return outputQueryJSON(results)
	}
	return outputQueryText(results)
}

type queryResult struct {
	Session      *session.Session
	MatchedFiles []string
	MatchedTags  []string
	MatchedDocs  []string
}

func matchSession(s *session.Session) *queryResult {
	r := &queryResult{Session: s}
	matched := true

	// File filter
	if queryFile != "" {
		matched = false
		g, err := glob.Compile(queryFile)
		if err != nil {
			// Fall back to exact match
			for _, f := range s.FilesChanged {
				if f.Path == queryFile {
					r.MatchedFiles = append(r.MatchedFiles, f.Path)
					matched = true
				}
			}
		} else {
			for _, f := range s.FilesChanged {
				if g.Match(f.Path) {
					r.MatchedFiles = append(r.MatchedFiles, f.Path)
					matched = true
				}
			}
		}
		if !matched {
			return nil
		}
	}

	// Tag filter
	if queryTag != "" {
		found := false
		for _, t := range s.Tags {
			if t == queryTag {
				r.MatchedTags = append(r.MatchedTags, t)
				found = true
			}
		}
		if !found {
			return nil
		}
	}

	// Doc type filter
	if queryDocType != "" {
		found := false
		for _, d := range s.Docs {
			if d.Type == queryDocType {
				r.MatchedDocs = append(r.MatchedDocs, d.Path)
				found = true
			}
		}
		if !found {
			return nil
		}
	}

	// Date filters
	if queryAfter != "" {
		afterDate, err := parseDateStr(queryAfter)
		if err == nil && s.Timestamp.Before(afterDate) {
			return nil
		}
	}

	if queryBefore != "" {
		beforeDate, err := parseDateStr(queryBefore)
		if err == nil {
			// Add a day to make "before" inclusive of the date
			endOfDay := beforeDate.AddDate(0, 0, 1)
			if s.Timestamp.After(endOfDay) || s.Timestamp.Equal(endOfDay) {
				return nil
			}
		}
	}

	// Full-text search
	if querySearch != "" {
		searchLower := strings.ToLower(querySearch)
		bodyLower := strings.ToLower(s.Body)
		if !strings.Contains(bodyLower, searchLower) {
			// Also search in file summaries and tags
			found := false
			for _, f := range s.FilesChanged {
				if strings.Contains(strings.ToLower(f.Summary), searchLower) {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}
	}

	return r
}

func parseDateStr(dateStr string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format %q (expected YYYY-MM-DD): %w", dateStr, err)
	}
	return t, nil
}

func outputQueryText(results []*queryResult) error {
	for _, r := range results {
		summary := extractSummaryLine(r.Session.Body)
		line := fmt.Sprintf("%s  %s", r.Session.SessionID, summary)

		var extras []string
		if len(r.MatchedFiles) > 0 {
			extras = append(extras, "files: "+strings.Join(r.MatchedFiles, ", "))
		}
		if len(r.MatchedTags) > 0 {
			extras = append(extras, "tags: "+strings.Join(r.MatchedTags, ", "))
		}
		if len(r.MatchedDocs) > 0 {
			extras = append(extras, "docs: "+strings.Join(r.MatchedDocs, ", "))
		}
		if len(extras) > 0 {
			line += "  [" + strings.Join(extras, "; ") + "]"
		}
		fmt.Println(line)
	}
	return nil
}

type queryJSONResult struct {
	SessionID string   `json:"session_id"`
	Summary   string   `json:"summary"`
	Tags      []string `json:"tags"`
	Files     []string `json:"matched_files,omitempty"`
	Docs      []string `json:"matched_docs,omitempty"`
}

func outputQueryJSON(results []*queryResult) error {
	var jsonResults []queryJSONResult
	for _, r := range results {
		jsonResults = append(jsonResults, queryJSONResult{
			SessionID: r.Session.SessionID,
			Summary:   extractSummaryLine(r.Session.Body),
			Tags:      r.Session.Tags,
			Files:     r.MatchedFiles,
			Docs:      r.MatchedDocs,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(jsonResults)
}
