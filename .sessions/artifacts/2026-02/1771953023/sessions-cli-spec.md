---
title: Sessions CLI Specification
type: architecture
summary: Full specification for the sessions CLI tool including schemas, commands, and technical requirements
status: accepted
supersedes: ""
---

# Sessions CLI — Specification

## Overview

`sessions` is a Go CLI tool for managing a local, file-based session memory system. It stores concise session summaries as markdown files with YAML frontmatter, supports deep-dive documentation as subdirectories, and provides query tooling to retrieve context based on files changed, tags, document types, and free-text search.

The primary consumer of this tool is an AI coding agent that needs to build and retrieve context about a codebase's evolution over time.

## Directory Structure

```
.sessions/
  2026-02-24_0942.md                      # session summary
  2026-02-24_0942/                         # optional deep docs
    nullable-column-strategy.md
    loader-refactor-notes.md
  2026-02-24_1530.md
  2026-02-24_1530/
    eks-cost-allocation-breakdown.md
```

- Session files live at the root of `.sessions/`
- Deep docs live in a subdirectory matching the session ID
- The `.sessions/` directory is located at the project root (same level as `.git/`)

## Session File Schema

```yaml
---
timestamp: 2026-02-24T09:42:00-06:00       # ISO 8601 with timezone
session_id: 2026-02-24_0942                 # derived from timestamp, YYYY-MM-DD_HHMM
tags: [data-ingestion, bronzer, refactor]   # freeform labels
files_changed:
  - path: src/bronzer/loader.go             # relative to project root
    action: modified                        # added | modified | deleted | renamed
    summary: Refactored CSV parsing to handle nullable columns
  - path: src/bronzer/loader_test.go
    action: added
    summary: Added table-driven tests for nullable column edge cases
docs:                                       # optional, references to deep docs
  - path: nullable-column-strategy.md       # relative to session subdirectory
    type: decision                          # decision | analysis | investigation | architecture | debug-log
    summary: Why we treat all columns as nullable by default
related_sessions: [2026-02-23_1415]         # optional, manual or inferred links
---

## Summary

A concise 2-4 sentence summary of what happened in this session.

## Key Decisions

Bullet points of decisions made and brief rationale.

## Open Questions

Anything unresolved that future sessions should be aware of.
```

### Field Notes

- `session_id` is the unique identifier and filename stem. Format: `YYYY-MM-DD_HHMM`
- `files_changed[].action` is an enum: `added`, `modified`, `deleted`, `renamed`
- `docs[].type` is an enum: `decision`, `analysis`, `investigation`, `architecture`, `debug-log`
- `related_sessions` can be populated manually or by the `link` command
- All paths in `files_changed` are relative to the project root
- All paths in `docs` are relative to the session's subdirectory

## Deep Doc Schema

```yaml
---
title: Nullable Column Strategy
type: decision
status: accepted                            # draft | accepted | superseded | deprecated
supersedes: null                            # session_id/doc-filename if this replaces an earlier doc
---

Freeform markdown body.
```

### Status Lifecycle

- `draft` → `accepted` → optionally `superseded` or `deprecated`
- When a doc is superseded, the new doc's `supersedes` field references the old one
- Query tooling should flag non-accepted docs in output

## CLI Commands

### `sessions init`

Initialize a `.sessions/` directory in the current project.

```bash
sessions init
```

- Creates `.sessions/` at the project root
- Creates `.sessions/.gitkeep`
- Prints confirmation

### `sessions new`

Scaffold a new session file.

```bash
# Auto-generate session ID from current time
sessions new

# With explicit tags
sessions new --tags data-ingestion,bronzer

# Auto-populate files_changed from git diff
sessions new --git-diff
sessions new --git-diff --base main
```

- Creates `YYYY-MM-DD_HHMM.md` with frontmatter template
- `--git-diff` runs `git diff --name-status` (default: against HEAD, or against `--base` ref) and populates `files_changed`
- Opens the file path to stdout so the caller can pipe it to an editor or agent

### `sessions doc`

Create a deep doc attached to a session.

```bash
# Attach to most recent session
sessions doc "nullable-column-strategy" --type decision

# Attach to a specific session
sessions doc "nullable-column-strategy" --session 2026-02-24_0942 --type decision
```

- Creates the session subdirectory if it doesn't exist
- Scaffolds the deep doc with frontmatter
- Adds an entry to the parent session's `docs` frontmatter list

### `sessions query`

Search sessions by various criteria. Returns matching session summaries.

```bash
# By file path (exact or glob)
sessions query --file src/bronzer/loader.go
sessions query --file "src/bronzer/*.go"

# By tag
sessions query --tag data-ingestion

# By doc type
sessions query --doc-type decision

# By date range
sessions query --after 2026-02-01 --before 2026-02-28

# Combine filters (AND logic)
sessions query --file src/bronzer/loader.go --tag refactor

# Full-text search across session bodies
sessions query --search "nullable columns"

# Limit results
sessions query --tag bronzer --limit 5
```

**Default output:** A concise list showing session ID, summary line, and matched criteria.

```
2026-02-24_0942  Refactored Bronzer CSV loader for nullable columns  [files: src/bronzer/loader.go]
2026-02-23_1415  Initial Bronzer loader implementation                [files: src/bronzer/loader.go]
```

### `sessions context`

Build a context bundle for a file or topic. This is the primary interface for AI agents.

```bash
# Get all context about a file
sessions context src/bronzer/loader.go

# Include deep docs in output
sessions context src/bronzer/loader.go --deep

# Get context for multiple files
sessions context src/bronzer/loader.go src/bronzer/config.go

# Output as JSON for programmatic consumption
sessions context src/bronzer/loader.go --format json
```

**Default output (markdown):**

```markdown
# Context: src/bronzer/loader.go

## 2026-02-24_0942 — Refactored CSV parsing for nullable columns
- **Action:** modified
- **Change:** Refactored CSV parsing to handle nullable columns
- **Tags:** data-ingestion, bronzer, refactor
- **Decisions:** nullable-column-strategy (accepted)

## 2026-02-23_1415 — Initial Bronzer loader implementation
- **Action:** added
- **Change:** Created initial CSV loader with basic type inference
- **Tags:** bronzer, new-feature
```

When `--deep` is passed, inline the full deep doc bodies below their references.

**JSON output** should mirror this structure for programmatic use:

```json
{
  "file": "src/bronzer/loader.go",
  "sessions": [
    {
      "session_id": "2026-02-24_0942",
      "timestamp": "2026-02-24T09:42:00-06:00",
      "summary": "Refactored Bronzer CSV loader for nullable columns",
      "file_action": "modified",
      "file_summary": "Refactored CSV parsing to handle nullable columns",
      "tags": ["data-ingestion", "bronzer", "refactor"],
      "docs": [
        {
          "path": "nullable-column-strategy.md",
          "type": "decision",
          "status": "accepted",
          "summary": "Why we treat all columns as nullable by default"
        }
      ]
    }
  ]
}
```

### `sessions link`

Manage related_sessions links.

```bash
# Manually link two sessions
sessions link 2026-02-24_0942 2026-02-23_1415

# Auto-link sessions that share files
sessions link --auto
```

- `--auto` scans all sessions and adds `related_sessions` entries where sessions share `files_changed` paths
- Links are bidirectional — both session files get updated

### `sessions list`

List all sessions, optionally filtered.

```bash
# List all sessions (most recent first)
sessions list

# List with tag filter
sessions list --tag bronzer

# Show with file counts
sessions list --verbose
```

### `sessions status`

Show status of deep docs, flagging stale decisions.

```bash
# Show all decisions and their statuses
sessions status --doc-type decision

# Flag superseded or deprecated docs
sessions status --stale
```

## Technical Requirements

### Language & Dependencies

- Go 1.22+
- Use `gopkg.in/yaml.v3` for YAML frontmatter parsing
- Use `github.com/spf13/cobra` for CLI structure
- Use `github.com/gobwas/glob` for file path glob matching
- No database — everything is parsed from the filesystem at query time
- Keep it fast: lazy parse frontmatter only unless body is needed

### Frontmatter Parsing

Session files use `---` delimited YAML frontmatter followed by markdown body. The parser should:

1. Read the file
2. Extract content between the first two `---` lines as YAML
3. Parse the remainder as the markdown body
4. Return a structured `Session` type

```go
type Session struct {
    Timestamp       time.Time       `yaml:"timestamp"`
    SessionID       string          `yaml:"session_id"`
    Tags            []string        `yaml:"tags"`
    FilesChanged    []FileChange    `yaml:"files_changed"`
    Docs            []DocRef        `yaml:"docs"`
    RelatedSessions []string        `yaml:"related_sessions"`
    Body            string          `yaml:"-"`
}

type FileChange struct {
    Path    string `yaml:"path"`
    Action  string `yaml:"action"`
    Summary string `yaml:"summary"`
}

type DocRef struct {
    Path    string `yaml:"path"`
    Type    string `yaml:"type"`
    Summary string `yaml:"summary"`
}

type DeepDoc struct {
    Title      string `yaml:"title"`
    Type       string `yaml:"type"`
    Status     string `yaml:"status"`
    Supersedes string `yaml:"supersedes"`
    Body       string `yaml:"-"`
}
```

### Project Root Detection

Find the project root by walking up from `cwd` looking for `.git/` or `.sessions/`. Error if neither is found.

### Error Handling

- Missing `.sessions/` directory → suggest `sessions init`
- Malformed frontmatter → warn and skip file, don't crash
- No results → clean message, exit 0

### Output

- Default output is human-readable plain text / markdown to stdout
- `--format json` flag on `query` and `context` commands for machine consumption
- Use exit codes: 0 success, 1 error, 2 no results

## Future Considerations (Out of Scope for v1)

- **Embedding-based search**: Index session summaries with embeddings for semantic query
- **Git hooks**: Auto-create session stubs on commit
- **Session merging**: Combine multiple small sessions into a coherent narrative
- **MCP server**: Expose the query/context interface as an MCP tool for AI agents
- **Watch mode**: Monitor git activity and prompt for session creation
