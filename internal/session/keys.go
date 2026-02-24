package session

import (
	"path/filepath"
	"strings"
)

// ParseKey parses a key string into its components.
// Session key: "2026-02-24_1710"
// Artifact key: "2026-02-24_1710/sessions-cli-spec.md"
func ParseKey(key string) (sessionID, artifactFile string, isArtifact bool) {
	if idx := strings.IndexByte(key, '/'); idx >= 0 {
		return key[:idx], key[idx+1:], true
	}
	return key, "", false
}

// ResolveKeyToPath resolves a key to a filesystem path.
func ResolveKeyToPath(sessionsDir, key string) string {
	sessionID, artifactFile, isArtifact := ParseKey(key)
	if isArtifact {
		return filepath.Join(sessionsDir, sessionID, artifactFile)
	}
	return filepath.Join(sessionsDir, sessionID+".md")
}

// FormatSessionKey formats a session ID as a key.
func FormatSessionKey(sessionID string) string {
	return sessionID
}

// FormatArtifactKey formats a session ID and artifact filename as a key.
func FormatArtifactKey(sessionID, artifactFile string) string {
	return sessionID + "/" + artifactFile
}
