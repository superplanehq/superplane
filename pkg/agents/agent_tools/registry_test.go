package agenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
)

const anthropicToolDescriptionLimit = 1024

func TestRegistryDefinitions_ReturnsRegisteredToolsInStableOrder(t *testing.T) {
	registry := NewRegistry(Dependencies{})

	definitions := registry.Definitions()

	require.Len(t, definitions, 2)
	assert.Equal(t, AppAgentToolName, definitions[0].Name())
	assert.Equal(t, ComponentSchemaAgentToolName, definitions[1].Name())
	assert.NotEmpty(t, definitions[0].Description())
	assert.NotEmpty(t, definitions[0].InputSchema())
}

func TestRegistryDefinitions_FitProviderDescriptionLimit(t *testing.T) {
	registry := NewRegistry(Dependencies{})

	for _, definition := range registry.Definitions() {
		description := definition.Description()
		require.NotEmpty(t, description)
		assert.LessOrEqual(
			t,
			len(description),
			anthropicToolDescriptionLimit,
			fmt.Sprintf("%s description exceeds provider limit", definition.Name()),
		)
	}
}

func TestSchemaRevision_IsStableForRegisteredDefinitions(t *testing.T) {
	first := SchemaRevision()
	second := SchemaRevision()

	assert.Equal(t, first, second)
	assert.Contains(t, first, "agent-tools-v1.2.0:")
}

func TestRegistryExecuteCustomTool_DispatchesByToolName(t *testing.T) {
	registry := NewRegistry(Dependencies{})

	result := registry.ExecuteCustomTool(context.Background(), agents.AgentSessionContext{}, agents.CustomToolUse{
		ID:    "toolu_schema",
		Name:  ComponentSchemaAgentToolName,
		Input: `{}`,
	})

	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "component schema registry is not configured")
}

func TestRegistryExecuteCustomTool_RejectsUnknownTool(t *testing.T) {
	registry := NewRegistry(Dependencies{})

	result := registry.ExecuteCustomTool(context.Background(), agents.AgentSessionContext{}, agents.CustomToolUse{
		ID:   "toolu_missing",
		Name: "missing_tool",
	})

	assert.True(t, result.IsError)
	assert.Equal(t, "toolu_missing", result.CustomToolUseID)
	assertJSONErrorContains(t, result.Content, `unsupported custom tool "missing_tool"`)
}

func assertJSONErrorContains(t *testing.T, content, expected string) {
	t.Helper()

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	assert.Contains(t, payload["error"], expected)
}
