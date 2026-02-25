package cmd

import (
	"strings"
	"testing"

	"github.com/glopal/sessions/internal/parser"
	"github.com/glopal/sessions/internal/session"
)

func TestParseTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty string", "", nil},
		{"single tag", "refactor", []string{"refactor"}},
		{"multiple tags", "refactor,cli,go", []string{"refactor", "cli", "go"}},
		{"whitespace trimming", " refactor , cli , go ", []string{"refactor", "cli", "go"}},
		{"trailing comma", "refactor,cli,", []string{"refactor", "cli"}},
		{"only commas", ",,", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTags(tt.input)
			if !tagsEqual(got, tt.want) {
				t.Errorf("parseTags(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMergeTags(t *testing.T) {
	tests := []struct {
		name      string
		flagTags  []string
		stdinTags []string
		want      []string
	}{
		{"both nil", nil, nil, nil},
		{"flag only", []string{"a", "b"}, nil, []string{"a", "b"}},
		{"stdin only", nil, []string{"x", "y"}, []string{"x", "y"}},
		{"no overlap", []string{"a"}, []string{"b"}, []string{"a", "b"}},
		{"overlap deduped", []string{"a", "b"}, []string{"b", "c"}, []string{"a", "b", "c"}},
		{"flag order preserved", []string{"z", "a"}, []string{"m"}, []string{"z", "a", "m"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeTags(tt.flagTags, tt.stdinTags)
			if !tagsEqual(got, tt.want) {
				t.Errorf("mergeTags(%v, %v) = %v, want %v", tt.flagTags, tt.stdinTags, got, tt.want)
			}
		})
	}
}

func TestParseGitStatusOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []session.FileChange
	}{
		{"empty output", "", nil},
		{
			"modified file",
			" M cmd/new.go\n",
			[]session.FileChange{{Path: "cmd/new.go", Action: "modified", Summary: "TODO"}},
		},
		{
			"added file (staged)",
			"A  cmd/new.go\n",
			[]session.FileChange{{Path: "cmd/new.go", Action: "added", Summary: "TODO"}},
		},
		{
			"deleted file",
			" D old.go\n",
			[]session.FileChange{{Path: "old.go", Action: "deleted", Summary: "TODO"}},
		},
		{
			"untracked file",
			"?? newfile.go\n",
			[]session.FileChange{{Path: "newfile.go", Action: "added", Summary: "TODO"}},
		},
		{
			"renamed file",
			"R  old.go -> new.go\n",
			[]session.FileChange{{Path: "new.go", Action: "renamed", Summary: "TODO"}},
		},
		{
			"mixed status",
			" M cmd/new.go\n?? sessions-new-spec.md\nA  internal/foo.go\n D removed.go\n",
			[]session.FileChange{
				{Path: "cmd/new.go", Action: "modified", Summary: "TODO"},
				{Path: "sessions-new-spec.md", Action: "added", Summary: "TODO"},
				{Path: "internal/foo.go", Action: "added", Summary: "TODO"},
				{Path: "removed.go", Action: "deleted", Summary: "TODO"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitStatusOutput(tt.output)
			if len(got) != len(tt.want) {
				t.Fatalf("parseGitStatusOutput() returned %d files, want %d\ngot: %+v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i].Path != tt.want[i].Path {
					t.Errorf("file[%d].Path = %q, want %q", i, got[i].Path, tt.want[i].Path)
				}
				if got[i].Action != tt.want[i].Action {
					t.Errorf("file[%d].Action = %q, want %q", i, got[i].Action, tt.want[i].Action)
				}
				if got[i].Summary != tt.want[i].Summary {
					t.Errorf("file[%d].Summary = %q, want %q", i, got[i].Summary, tt.want[i].Summary)
				}
			}
		})
	}
}

func TestBuildHeredocTemplate(t *testing.T) {
	t.Run("empty tags and files", func(t *testing.T) {
		out := buildHeredocTemplate(nil, nil)
		if !strings.Contains(out, "tags: []") {
			t.Error("expected 'tags: []' in output")
		}
		if !strings.Contains(out, "files_changed: []") {
			t.Error("expected 'files_changed: []' in output")
		}
		if !strings.HasSuffix(out, "SESS\n") {
			t.Error("expected output to end with 'SESS\\n'")
		}
		if !strings.Contains(out, "sessions new <<SESS") {
			t.Error("expected HEREDOC command in output")
		}
	})

	t.Run("populated tags and files", func(t *testing.T) {
		tags := []string{"refactor", "cli"}
		files := []session.FileChange{
			{Path: "cmd/new.go", Action: "modified", Summary: "TODO"},
		}
		out := buildHeredocTemplate(tags, files)
		if !strings.Contains(out, "  - refactor") {
			t.Error("expected tag 'refactor' in output")
		}
		if !strings.Contains(out, "  - cli") {
			t.Error("expected tag 'cli' in output")
		}
		if !strings.Contains(out, "  - path: cmd/new.go") {
			t.Error("expected file path in output")
		}
		if !strings.Contains(out, "    action: modified") {
			t.Error("expected file action in output")
		}
	})
}

func TestRoundTrip(t *testing.T) {
	// Build a template, extract the HEREDOC content, parse it, verify fields
	tags := []string{"test-tag"}
	files := []session.FileChange{
		{Path: "cmd/new.go", Action: "modified", Summary: "TODO"},
		{Path: "readme.md", Action: "added", Summary: "TODO"},
	}
	template := buildHeredocTemplate(tags, files)

	// Extract content between "sessions new <<SESS\n" and "\nSESS\n"
	startMarker := "sessions new <<SESS\n"
	startIdx := strings.Index(template, startMarker)
	if startIdx < 0 {
		t.Fatal("could not find start marker in template")
	}
	content := template[startIdx+len(startMarker):]

	endMarker := "\nSESS\n"
	endIdx := strings.Index(content, endMarker)
	if endIdx < 0 {
		t.Fatal("could not find end marker in template")
	}
	content = content[:endIdx]

	// Parse with the real parser
	s, err := parser.ParseSession(content)
	if err != nil {
		t.Fatalf("ParseSession failed: %v", err)
	}

	if s.Summary != "" {
		t.Errorf("Summary = %q, want empty", s.Summary)
	}
	if len(s.Tags) != 1 || s.Tags[0] != "test-tag" {
		t.Errorf("Tags = %v, want [test-tag]", s.Tags)
	}
	if len(s.FilesChanged) != 2 {
		t.Fatalf("FilesChanged has %d entries, want 2", len(s.FilesChanged))
	}
	if s.FilesChanged[0].Path != "cmd/new.go" {
		t.Errorf("FilesChanged[0].Path = %q, want cmd/new.go", s.FilesChanged[0].Path)
	}
	if s.FilesChanged[0].Action != "modified" {
		t.Errorf("FilesChanged[0].Action = %q, want modified", s.FilesChanged[0].Action)
	}
	if s.FilesChanged[1].Path != "readme.md" {
		t.Errorf("FilesChanged[1].Path = %q, want readme.md", s.FilesChanged[1].Path)
	}
	if !strings.Contains(s.Body, "## Key Decisions") {
		t.Error("Body should contain '## Key Decisions'")
	}
}

// tagsEqual compares two string slices, treating nil and empty as equal.
func tagsEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
