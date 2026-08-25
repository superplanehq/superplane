package runner

import _ "embed"

//go:embed llm_usage.js
var LLMUsageScript string

func LLMUsageTaskFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "llm_usage.js", Content: LLMUsageScript, Mode: "0644"}
}
