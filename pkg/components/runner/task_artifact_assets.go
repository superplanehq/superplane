package runner

import _ "embed"

//go:embed task_artifact_mcp.js
var taskArtifactMCPScript string

//go:embed publish_link.js
var publishLinkScript string

//go:embed publish_markdown.js
var publishMarkdownScript string

//go:embed publish_file.js
var publishFileScript string

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

func TaskArtifactMCPScript() string { return taskArtifactMCPScript }

func TaskArtifactMCPFile() TaskFile {
	return TaskFile{Path: "task_artifact_mcp.js", Content: taskArtifactMCPScript, Mode: "0644"}
}

func AppendTaskArtifactMCP(environment []TaskEnvironmentVariable, files []TaskFile) []TaskFile {
	if !HasArtifactUploadToken(environment) && !HasPublishLinks(environment) {
		return files
	}
	// The MCP script loads publish_link.js at startup, including visual
	// evidence runs that do not publish links.
	files = appendUniqueTaskFiles(files, TaskArtifactMCPFile(), PublishLinkScriptFile())
	if HasPublishLinks(environment) {
		files = appendUniqueTaskFiles(files, taskCLIFiles()...)
	}
	return files
}

// taskCLIFiles is the superplane shell command. task_cli.js loads
// publish_link.js, publish_markdown.js, and publish_file.js at startup,
// including calls that do not publish an artifact.
func taskCLIFiles() []TaskFile {
	return []TaskFile{
		TaskCLIScriptFile(), PublishLinkScriptFile(), PublishMarkdownScriptFile(), PublishFileScriptFile(), PublishLinkCommandFile(),
	}
}

func PublishLinkScriptFile() TaskFile {
	return TaskFile{Path: "publish_link.js", Content: publishLinkScript, Mode: "0644"}
}

func PublishMarkdownScriptFile() TaskFile {
	return TaskFile{Path: "publish_markdown.js", Content: publishMarkdownScript, Mode: "0644"}
}

func PublishFileScriptFile() TaskFile {
	return TaskFile{Path: "publish_file.js", Content: publishFileScript, Mode: "0644"}
}

func TaskCLIScriptFile() TaskFile {
	return TaskFile{Path: "task_cli.js", Content: taskCLIScript, Mode: "0644"}
}

func PublishLinkCommandFile() TaskFile {
	return TaskFile{Path: "bin/superplane", Content: taskCommand, Mode: "0755"}
}
