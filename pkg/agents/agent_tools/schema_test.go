package agenttools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/registryimports"
)

var _ = registryimports.Loaded

func TestComponentSchemaAgentTool_ReturnsExactSlackSchema(t *testing.T) {
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

func TestAppAgentToolSchemaIncludesRuntimeReadAction(t *testing.T) {
	tool := NewAppAgentTool(AppAgentToolOptions{})

	schema := tool.InputSchema()
	actionSchema := schema.Properties["action"]
	resourceSchema := schema.Properties["resource"]

	assert.Contains(t, actionSchema.Enum, "create_draft")
	assert.Contains(t, actionSchema.Enum, "read_runtime")
	assert.Contains(t, actionSchema.Enum, "list_files")
	assert.Contains(t, actionSchema.Enum, "read_file")
	assert.Contains(t, actionSchema.Enum, "write_file")
	assert.Contains(t, actionSchema.Enum, "delete_file")
	assert.Contains(t, actionSchema.Enum, "commit_files")
	assert.ElementsMatch(t, []string{
		"memory",
		"runs",
		"event_executions",
		"node_executions",
		"node_queue_items",
		"node_events",
	}, resourceSchema.Enum)
	assert.Contains(t, schema.Properties, "namespace")
	assert.Contains(t, schema.Properties, "node_id")
	assert.Contains(t, schema.Properties, "event_id")
	assert.Contains(t, schema.Properties, "execution_id")
	assert.Contains(t, schema.Properties, "path")
	assert.Contains(t, schema.Properties, "paths")
	assert.Contains(t, schema.Properties, "content")
	assert.Contains(t, schema.Properties, "message")
	assert.Contains(t, schema.Properties, "query")
	assert.Contains(t, schema.Properties["path"].Description, "AGENTS.md")
	assert.Contains(t, schema.Properties["content"].Description, "write_file")
	assert.Contains(t, schema.Properties["message"].Description, "commit_files")
}

func TestAppAgentToolSchemaWarnsAgainstTemplateFieldsInCanvasYAML(t *testing.T) {
	tool := NewAppAgentTool(AppAgentToolOptions{})

	schema := tool.InputSchema()
	canvasYAMLSchema := schema.Properties["canvas_yaml"]

	assert.Contains(t, canvasYAMLSchema.Description, "canonical live canvas.yaml")
	assert.Contains(t, canvasYAMLSchema.Description, "Unknown fields are rejected")
	assert.Contains(t, canvasYAMLSchema.Description, "metadata.isTemplate")
}

func TestAppAgentToolSchemaIncludesDraftVersionIDForUpdates(t *testing.T) {
	tool := NewAppAgentTool(AppAgentToolOptions{})

	schema := tool.InputSchema()

	assert.Contains(t, schema.Properties, "version_id")
	assert.Contains(t, schema.Properties, "draft_version_id")
	assert.Contains(t, schema.Properties, "display_name")
	assert.Contains(t, schema.Properties["version_id"].Description, "For read, read_file")
	assert.Contains(t, schema.Properties["draft_version_id"].Description, "Alias")
	assert.Contains(t, schema.Properties["version_id"].Description, "read")
	assert.Contains(t, schema.Properties["version_id"].Description, "read returns source live")
	assert.Contains(t, schema.Properties["display_name"].Description, "For create_draft")
}

func TestComponentSchemaAgentTool_ReturnsCoreComponentSchema(t *testing.T) {
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

func TestComponentSchemaAgentTool_ReturnsVendorComponents(t *testing.T) {
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

func TestComponentSchemaAgentTool_ReportsMissingKeys(t *testing.T) {
	tool := newComponentSchemaTool(t)

	result := executeComponentSchemaTool(t, tool, superPlaneComponentSchemaInput{
		ComponentKeys: []string{"missing.component"},
	})

	assert.Empty(t, result.Components)
	assert.Equal(t, []string{"missing.component"}, result.Missing)
}

func TestComponentSchemaAgentTool_ReportsOmittedValidKeysWhenLimited(t *testing.T) {
	tool := newComponentSchemaTool(t)

	result := executeComponentSchemaTool(t, tool, superPlaneComponentSchemaInput{
		ComponentKeys: []string{"wait", "noop"},
		Limit:         1,
	})

	require.Len(t, result.Components, 1)
	assert.Equal(t, "wait", result.Components[0].Key)
	assert.Empty(t, result.Missing)
	assert.Equal(t, []string{"noop"}, result.Omitted)
	assert.True(t, result.Truncated)
	assert.Contains(t, result.Notes, "Result was truncated by limit; request omitted component_keys explicitly or raise limit up to 40 if you need more.")
}

func TestComponentSchemaAgentTool_ReportsOmittedVendorMatchesWhenLimited(t *testing.T) {
	tool := newComponentSchemaTool(t)

	result := executeComponentSchemaTool(t, tool, superPlaneComponentSchemaInput{
		Vendors: []string{"slack"},
		Limit:   1,
	})

	require.Len(t, result.Components, 1)
	require.NotEmpty(t, result.Omitted)
	assert.True(t, result.Truncated)
	for _, key := range result.Omitted {
		assert.Contains(t, key, "slack.")
	}
}

func TestComponentSchemaAgentTool_ReportsOmittedQueryMatchesWhenLimited(t *testing.T) {
	tool := newComponentSchemaTool(t)

	result := executeComponentSchemaTool(t, tool, superPlaneComponentSchemaInput{
		Query: "slack",
		Limit: 1,
	})

	require.Len(t, result.Components, 1)
	require.NotEmpty(t, result.Omitted)
	assert.True(t, result.Truncated)
	for _, key := range result.Omitted {
		assert.NotEmpty(t, key)
	}
}

func newComponentSchemaTool(t *testing.T) *ComponentSchemaAgentTool {
	t.Helper()

	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)
	return NewComponentSchemaAgentTool(reg)
}

func executeComponentSchemaTool(t *testing.T, tool *ComponentSchemaAgentTool, input superPlaneComponentSchemaInput) superPlaneComponentSchemaResult {
	t.Helper()

	toolResult, err := tool.Call(context.Background(), agents.AgentSessionContext{}, input)
	require.NoError(t, err)
	result, ok := toolResult.Payload.(superPlaneComponentSchemaResult)
	require.True(t, ok)
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
