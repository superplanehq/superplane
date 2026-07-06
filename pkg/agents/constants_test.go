package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentTokenScopes(t *testing.T) {
	canvasID := "canvas-123"

	assert.ElementsMatch(t, []string{
		"org:read",
		"integrations:read",
		"canvases:read:" + canvasID,
		"canvases:update_version:" + canvasID,
	}, AgentTokenScopes(canvasID))
}

func TestBuilderModeInstructionsIncludeNodeReplacementGuidance(t *testing.T) {
	instructions := modeInstructions(ModeBuilder)

	assert.Contains(t, instructions, "Do not change an existing node's implementation with update_node")
	assert.Contains(t, instructions, "component/trigger/widget/integration replacements must be delete_node plus add_node")
}
