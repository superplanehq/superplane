package runner

import _ "embed"

//go:embed llm_usage.js
var LLMUsageScript string

func LLMUsageTaskFile() TaskFile {
	return TaskFile{Path: "llm_usage.js", Content: LLMUsageScript, Mode: "0644"}
}
