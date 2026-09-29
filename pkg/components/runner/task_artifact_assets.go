package runner

import _ "embed"

//go:embed task_artifact_mcp.js
var taskArtifactMCPScript string

func TaskArtifactMCPScript() string { return taskArtifactMCPScript }

func TaskArtifactMCPFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "task_artifact_mcp.js", Content: taskArtifactMCPScript, Mode: "0644"}
}

func AppendTaskArtifactMCP(environment []BrokerEnvironmentVariable, files []BrokerTaskFile) []BrokerTaskFile {
	if !HasArtifactUploadToken(environment) {
		return files
	}
	return append(files, TaskArtifactMCPFile())
}
