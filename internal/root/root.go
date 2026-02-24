package root

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectRoot walks up from the current directory looking for .git/ or .sessions/.
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return findProjectRootFrom(dir)
}

func findProjectRootFrom(dir string) (string, error) {
	for {
		if isDir(filepath.Join(dir, ".git")) || isDir(filepath.Join(dir, ".sessions")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (no .git/ or .sessions/ found)")
		}
		dir = parent
	}
}

// SessionsDir returns the path to the .sessions/ directory.
func SessionsDir() (string, error) {
	root, err := FindProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".sessions"), nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
