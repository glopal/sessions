# sessions
`sessions` is a Go CLI tool for managing a local, file-based session memory system. It stores concise session summaries as markdown files with YAML frontmatter, supports deep-dive documentation as subdirectories, and provides query tooling to retrieve context based on files changed, tags, document types, and free-text search.

The primary consumer of this tool is an AI coding agent that needs to build and retrieve context about a codebase's evolution over time.