package native

import _ "embed"

//go:embed agent_prompt.md
var defaultAgentPrompt string

func DefaultAgentPrompt() string {
	return defaultAgentPrompt
}
