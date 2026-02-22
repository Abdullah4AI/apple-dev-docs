package docsembed

import _ "embed"

//go:embed WORKFLOWS.md
var WorkflowsGuide string

//go:embed API_NOTES.md
var APINotesGuide string
