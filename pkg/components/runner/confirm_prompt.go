package runner

import _ "embed"

//go:embed confirm_prompt.js
var confirmPromptScript string

func ConfirmPromptTaskFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "confirm_prompt.js", Content: confirmPromptScript, Mode: "0755"}
}
