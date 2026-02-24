package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glopal/sessions/internal/root"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a .sessions/ directory in the current project",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := root.FindProjectRoot()
		if err != nil {
			// Fall back to cwd if no project root found
			projectRoot, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}
		}

		sessionsDir := filepath.Join(projectRoot, ".sessions")

		if info, err := os.Stat(sessionsDir); err == nil && info.IsDir() {
			fmt.Println(".sessions/ directory already exists")
			return nil
		}

		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			return fmt.Errorf("creating .sessions/ directory: %w", err)
		}

		gitkeep := filepath.Join(sessionsDir, ".gitkeep")
		if err := os.WriteFile(gitkeep, []byte{}, 0644); err != nil {
			return fmt.Errorf("creating .gitkeep: %w", err)
		}

		fmt.Printf("Initialized .sessions/ directory at %s\n", sessionsDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
