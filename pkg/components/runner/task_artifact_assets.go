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

func PublishLinkFiles() []TaskFile {
	return []TaskFile{
		{Path: "publish_link.js", Content: publishLinkScript, Mode: "0644"},
		{Path: "bin/superplane", Content: publishLinkCommand, Mode: "0755"},
	}
}

func AppendTaskArtifactMCP(environment []TaskEnvironmentVariable, files []TaskFile) []TaskFile {
	if !HasArtifactUploadToken(environment) && !HasPublishLinks(environment) {
		return files
	}
	files = append(files, TaskArtifactMCPFile())
	if HasPublishLinks(environment) {
		files = append(files, PublishLinkFiles()...)
	}
	return files
}
