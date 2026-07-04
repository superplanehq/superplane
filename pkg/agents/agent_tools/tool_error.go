package agenttools

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/agents"
)

func customToolError(toolUseID, message string) agents.CustomToolResult {
	content, _ := json.Marshal(map[string]string{"error": message})
	return agents.CustomToolResult{
		CustomToolUseID: toolUseID,
		Content:         string(content),
		IsError:         true,
	}
}
