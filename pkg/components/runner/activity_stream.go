package runner

import _ "embed"

//go:embed activity_stream.js
var ActivityStreamScript string

func ActivityStreamTaskFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "activity_stream.js", Content: ActivityStreamScript, Mode: "0644"}
}
