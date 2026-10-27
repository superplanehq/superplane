package runner

import _ "embed"

//go:embed task_artifact_mcp.js
var taskArtifactMCPScript string

//go:embed publish_link.js
var publishLinkScript string

const publishLinkCommand = `#!/bin/sh
set -eu
if [ -z "${SUPERPLANE_TASK_DIR:-}" ]; then
  echo "SUPERPLANE_TASK_DIR is required" >&2
  exit 1
fi
exec node "$SUPERPLANE_TASK_DIR/publish_link.js" "$@"
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
	files = append(files, TaskArtifactMCPFile(), PublishLinkScriptFile())
	if HasPublishLinks(environment) {
		files = append(files, PublishLinkCommandFile())
	}
	return files
}

func PublishLinkScriptFile() TaskFile {
	return TaskFile{Path: "publish_link.js", Content: publishLinkScript, Mode: "0644"}
}

func PublishLinkCommandFile() TaskFile {
	return TaskFile{Path: "bin/superplane", Content: publishLinkCommand, Mode: "0755"}
}
