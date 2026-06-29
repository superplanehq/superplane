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
