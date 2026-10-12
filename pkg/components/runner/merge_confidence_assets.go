package runner

import _ "embed"

//go:embed merge_confidence_mcp.js
var mergeConfidenceMCPScript string

func MergeConfidenceMCPScript() string { return mergeConfidenceMCPScript }

func MergeConfidenceMCPFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "merge_confidence_mcp.js", Content: mergeConfidenceMCPScript, Mode: "0644"}
}

func AppendMergeConfidenceMCP(environment []BrokerEnvironmentVariable, files []BrokerTaskFile) []BrokerTaskFile {
	if !HasMergeConfidenceToken(environment) {
		return files
	}
	return appendUniqueTaskFiles(files, MergeConfidenceMCPFile())
}
