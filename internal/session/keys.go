package session

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ParseKey parses a key string into its components.
// Session key: "1740422423"
// Artifact key: "1740422423/sessions-cli-spec.md"
func ParseKey(key string) (sessionID, artifactFile string, isArtifact bool) {
	if idx := strings.IndexByte(key, '/'); idx >= 0 {
		return key[:idx], key[idx+1:], true
	}
	return key, "", false
}

// EpochToYearMonth converts an epoch string to a "YYYY-MM" directory name using UTC.
func EpochToYearMonth(epochStr string) (string, error) {
	epoch, err := strconv.ParseInt(epochStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid epoch %q: %w", epochStr, err)
	}
	t := time.Unix(epoch, 0).UTC()
	return t.Format("2006-01"), nil
}

// ResolveKeyToPath resolves a key to a filesystem path.
// Session: .sessions/sessions/YYYY-MM/EPOCH.md
// Artifact: .sessions/artifacts/YYYY-MM/EPOCH/artifact.md
func ResolveKeyToPath(sessionsDir, key string) string {
	sessionID, artifactFile, isArtifact := ParseKey(key)
	yearMonth, err := EpochToYearMonth(sessionID)
	if err != nil {
		// Fallback for non-epoch IDs (e.g. legacy format)
		if isArtifact {
			return filepath.Join(sessionsDir, "artifacts", sessionID, artifactFile)
		}
		return filepath.Join(sessionsDir, "sessions", sessionID+".md")
	}
	if isArtifact {
		return filepath.Join(sessionsDir, "artifacts", yearMonth, sessionID, artifactFile)
	}
	return filepath.Join(sessionsDir, "sessions", yearMonth, sessionID+".md")
}

// ResolveSessionPath resolves a session ID to its file path.
func ResolveSessionPath(sessionsDir, sessionID string) string {
	return ResolveKeyToPath(sessionsDir, FormatSessionKey(sessionID))
}

// ResolveArtifactDir resolves a session ID to its artifact directory path.
func ResolveArtifactDir(sessionsDir, sessionID string) string {
	yearMonth, err := EpochToYearMonth(sessionID)
	if err != nil {
		return filepath.Join(sessionsDir, "artifacts", sessionID)
	}
	return filepath.Join(sessionsDir, "artifacts", yearMonth, sessionID)
}

// FormatSessionKey formats a session ID as a key.
func FormatSessionKey(sessionID string) string {
	return sessionID
}

// FormatArtifactKey formats a session ID and artifact filename as a key.
func FormatArtifactKey(sessionID, artifactFile string) string {
	return sessionID + "/" + artifactFile
}
