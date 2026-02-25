package cmd

import (
	"strings"
	"testing"

	"github.com/glopal/sessions/internal/parser"
)

func TestTitleFromName(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{"kebab-case with .md", "sessions-new-spec.md", "Sessions New Spec"},
		{"kebab-case without .md", "sessions-new-spec", "Sessions New Spec"},
		{"single word", "readme", "Readme"},
		{"single word with .md", "readme.md", "Readme"},
		{"already has .md in name", "my-doc.md", "My Doc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titleFromName(tt.input)
			if got != tt.want {
				t.Errorf("titleFromName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildArtifactHeredocTemplate(t *testing.T) {
	t.Run("default structure", func(t *testing.T) {
		out := buildArtifactHeredocTemplate("foo.md", "1771969857", "analysis", "Foo")

		if !strings.Contains(out, "sessions artifact foo --session 1771969857 <<ART") {
			t.Error("expected HEREDOC command with name (without .md) and session ID")
		}
		if !strings.Contains(out, "title: Foo") {
			t.Error("expected title in output")
		}
		if !strings.Contains(out, "type: analysis") {
			t.Error("expected type in output")
		}
		if !strings.Contains(out, "summary: \"\"") {
			t.Error("expected empty summary in output")
		}
		if !strings.Contains(out, "status: draft") {
			t.Error("expected status draft in output")
		}
		if !strings.Contains(out, "supersedes: \"\"") {
			t.Error("expected empty supersedes in output")
		}
		if !strings.HasSuffix(out, "ART\n") {
			t.Error("expected output to end with 'ART\\n'")
		}
		if !strings.Contains(out, "Content goes here.") {
			t.Error("expected placeholder body content")
		}
	})

	t.Run("custom type", func(t *testing.T) {
		out := buildArtifactHeredocTemplate("my-design.md", "123456", "decision", "My Design")
		if !strings.Contains(out, "type: decision") {
			t.Error("expected custom type in output")
		}
		if !strings.Contains(out, "sessions artifact my-design --session 123456 <<ART") {
			t.Error("expected correct command with custom values")
		}
	})
}

func TestArtifactRoundTrip(t *testing.T) {
	// Build a template, extract the HEREDOC content, parse with real parser, verify fields
	out := buildArtifactHeredocTemplate("sessions-new-spec.md", "1771969857", "analysis", "Sessions New Spec")

	// Extract content between "<<ART\n" and "\nART\n"
	startMarker := "<<ART\n"
	startIdx := strings.Index(out, startMarker)
	if startIdx < 0 {
		t.Fatal("could not find start marker in template")
	}
	content := out[startIdx+len(startMarker):]

	endMarker := "\nART\n"
	endIdx := strings.Index(content, endMarker)
	if endIdx < 0 {
		t.Fatal("could not find end marker in template")
	}
	content = content[:endIdx]

	// Parse with the real parser
	a, err := parser.ParseArtifact(content)
	if err != nil {
		t.Fatalf("ParseArtifact failed: %v", err)
	}

	if a.Title != "Sessions New Spec" {
		t.Errorf("Title = %q, want %q", a.Title, "Sessions New Spec")
	}
	if a.Type != "analysis" {
		t.Errorf("Type = %q, want %q", a.Type, "analysis")
	}
	if a.Summary != "" {
		t.Errorf("Summary = %q, want empty", a.Summary)
	}
	if a.Status != "draft" {
		t.Errorf("Status = %q, want %q", a.Status, "draft")
	}
	if a.Body != "Content goes here." {
		t.Errorf("Body = %q, want %q", a.Body, "Content goes here.")
	}
}

func TestEnsureMD(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo", "foo.md"},
		{"foo.md", "foo.md"},
		{"bar-baz", "bar-baz.md"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ensureMD(tt.input)
			if got != tt.want {
				t.Errorf("ensureMD(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
