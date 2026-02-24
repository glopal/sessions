package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Manage local, file-based session memory",
	Long:  `sessions is a CLI tool for managing a local, file-based session memory system. It stores concise session summaries as markdown files with YAML frontmatter, supports deep-dive documentation, and provides query tooling to retrieve context.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
