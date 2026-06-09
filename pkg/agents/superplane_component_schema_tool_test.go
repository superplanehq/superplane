package agents

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/registryimports"
)

var _ = registryimports.Loaded

func TestSuperPlaneComponentSchemaTool_ReturnsExactSlackSchema(t *testing.T) {
	tool := newComponentSchemaTool(t)

	result := executeComponentSchemaTool(t, tool, superPlaneComponentSchemaInput{
		ComponentKeys: []string{"slack.waitForButtonClick"},
	})

	require.Empty(t, result.Missing)
	require.Len(t, result.Components, 1)

	component := result.Components[0]
	assert.Equal(t, "slack.waitForButtonClick", component.Key)
	assert.Equal(t, "action", component.Kind)
	assert.Equal(t, "slack", component.RequiresIntegration)
	assert.ElementsMatch(t, []string{"channel", "message", "buttons", "timeout"}, componentFieldNames(component.Configuration))
	assert.ElementsMatch(t, []string{"received", "timeout"}, outputChannelNames(component.OutputChannels))
	assert.Contains(t, result.Notes, "Use output_channels.name exactly in edge channel values; labels are display-only.")
	assert.NotContains(t, result.Notes, "Use output_channels.name exactly in edge sourceName values; labels are display-only.")
}

func TestSuperPlaneComponentSchemaTool_ReturnsCoreComponentSchema(t *testing.T) {
	tool := newComponentSchemaTool(t)

	result := executeComponentSchemaTool(t, tool, superPlaneComponentSchemaInput{
		ComponentKeys: []string{"wait"},
	})

	require.Empty(t, result.Missing)
	require.Len(t, result.Components, 1)

	component := result.Components[0]
	assert.Equal(t, "wait", component.Key)
	assert.Equal(t, "action", component.Kind)
	assert.Empty(t, component.RequiresIntegration)
	assert.Contains(t, componentFieldNames(component.Configuration), "mode")
	assert.Contains(t, componentFieldNames(component.Configuration), "waitFor")
	assert.Contains(t, componentFieldNames(component.Configuration), "unit")
	assert.Contains(t, outputChannelNames(component.OutputChannels), "default")
}

func TestSuperPlaneComponentSchemaTool_ReturnsVendorComponents(t *testing.T) {
	tool := newComponentSchemaTool(t)

	result := executeComponentSchemaTool(t, tool, superPlaneComponentSchemaInput{
		Vendors:         []string{"slack"},
		IncludeExamples: true,
		Limit:           5,
	})

	require.NotEmpty(t, result.Components)
	for _, component := range result.Components {
		assert.Equal(t, "slack", component.RequiresIntegration)
		assert.Contains(t, component.Key, "slack.")
		assert.Empty(t, component.ExampleOutput)
		assert.Empty(t, component.ExampleData)
	}
}

func TestSuperPlaneComponentSchemaTool_ReportsMissingKeys(t *testing.T) {
	tool := newComponentSchemaTool(t)

	result := executeComponentSchemaTool(t, tool, superPlaneComponentSchemaInput{
		ComponentKeys: []string{"missing.component"},
	})

	assert.Empty(t, result.Components)
	assert.Equal(t, []string{"missing.component"}, result.Missing)
}

func newComponentSchemaTool(t *testing.T) *SuperPlaneComponentSchemaTool {
	t.Helper()

	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)
	return NewSuperPlaneComponentSchemaTool(reg)
}

func executeComponentSchemaTool(t *testing.T, tool *SuperPlaneComponentSchemaTool, input superPlaneComponentSchemaInput) superPlaneComponentSchemaResult {
	t.Helper()

	data, err := json.Marshal(input)
	require.NoError(t, err)

	toolResult := tool.ExecuteCustomTool(context.Background(), AgentSessionContext{}, CustomToolUse{
		ID:    "toolu_schema",
		Name:  SuperPlaneComponentSchemaToolName,
		Input: string(data),
	})
	require.False(t, toolResult.IsError)

	var result superPlaneComponentSchemaResult
	require.NoError(t, json.Unmarshal([]byte(toolResult.Content), &result))
	return result
}

func componentFieldNames(fields []superPlaneComponentField) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}

func outputChannelNames(channels []superPlaneOutputChannel) []string {
	names := make([]string, 0, len(channels))
	for _, channel := range channels {
		names = append(names, channel.Name)
	}
	return names
}
