package runner

import _ "embed"

//go:embed publish_markdown.js
var publishMarkdownScript string

//go:embed task_cli.js
var taskCLIScript string

const taskCommand = `#!/bin/sh
set -eu
if [ -z "${SUPERPLANE_TASK_DIR:-}" ]; then
  echo "SUPERPLANE_TASK_DIR is required" >&2
  exit 1
fi
exec node "$SUPERPLANE_TASK_DIR/task_cli.js" "$@"
`

func AppendTaskArtifactMCP(environment []TaskEnvironmentVariable, files []TaskFile) []TaskFile {
	if !HasPublishLinks(environment) {
		return files
	}
	return appendUniqueTaskFiles(files, PublishMarkdownScriptFile())
}

func PublishMarkdownScriptFile() TaskFile {
	return TaskFile{Path: "publish_markdown.js", Content: publishMarkdownScript, Mode: "0644"}
}

func TaskCLIScriptFile() TaskFile {
	return TaskFile{Path: "task_cli.js", Content: taskCLIScript, Mode: "0644"}
}

func PublishLinkCommandFile() TaskFile {
	return TaskFile{Path: "bin/superplane", Content: taskCommand, Mode: "0755"}
}

// TaskCommandFiles is the superplane shell command. It calls the same tool
// handlers as the global runner MCP server.
func TaskCommandFiles() []TaskFile {
	return []TaskFile{TaskCLIScriptFile(), PublishLinkCommandFile()}
}
