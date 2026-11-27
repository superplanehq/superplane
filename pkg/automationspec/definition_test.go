package automationspec_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automationspec"
)

func TestWorkspaceSourceSwitchPreservesModelOnlyYAML(t *testing.T) {
	source := definitionHeader + "agent: {model: old-model}\n" + manualTrigger + bashStep
	switched := &automationspec.Agent{
		Harness: automationspec.Codex, Credentials: &automationspec.Credentials{Source: automationspec.Subscription},
	}
	updated, err := automationspec.ReplaceExplicitAgent(source, switched)
	require.NoError(t, err)
	document, err := automationspec.Parse(updated)
	require.NoError(t, err)
	require.Nil(t, document.Agent)

	switched.Model = "new-model"
	updated, err = automationspec.ReplaceExplicitAgent(source, switched)
	require.NoError(t, err)
	document, err = automationspec.Parse(updated)
	require.NoError(t, err)
	require.Equal(t, &automationspec.Agent{Model: "new-model"}, document.Agent)
}

func TestAgentSourceSwitchPreservesExecutionOptions(t *testing.T) {
	source := definitionHeader + manualTrigger + bashStep + "agent: {skip_setup_commands: false, skip_repository_checkout: true}\n"
	updated, err := automationspec.ReplaceExplicitAgent(source, &automationspec.Agent{Harness: automationspec.Codex, Model: "gpt-5"})
	require.NoError(t, err)
	require.Equal(t, source, updated, "execution options must keep inheriting the workspace agent")
	document, err := automationspec.Parse(updated)
	require.NoError(t, err)
	require.Equal(t, "", string(document.Agent.Harness))
	require.Empty(t, document.Agent.Model)
	require.NotNil(t, document.Agent.SkipSetupCommands)
	require.False(t, *document.Agent.SkipSetupCommands)
	require.True(t, *document.Agent.SkipRepositoryCheckout)
}

func TestStoredAgentSelectionSurvivesFormatChange(t *testing.T) {
	generated := definitionHeader + "agent: {skip_setup_commands: false, skip_repository_checkout: true}\n" + manualTrigger + bashStep
	for _, selection := range []struct {
		name, agent string
		want        automationspec.Agent
	}{
		{"model equal to workspace default", "{model: workspace-default}", automationspec.Agent{Model: "workspace-default"}},
		{"explicit credential source", "{harness: codex, model: chosen, credentials: {source: integration, name: team}}", automationspec.Agent{Harness: automationspec.Codex, Model: "chosen", Credentials: &automationspec.Credentials{Source: automationspec.Integration, Name: "team"}}},
	} {
		t.Run(selection.name, func(t *testing.T) {
			stored := definitionHeader + "agent: " + selection.agent + "\ntriggers: [{use: manual}]\n" + bashStep
			_, err := automationspec.Parse(stored)
			require.Error(t, err)
			source, err := automationspec.RestoreStoredAgentSelection(generated, stored)
			require.NoError(t, err)
			document, err := automationspec.Parse(source)
			require.NoError(t, err)
			require.Equal(t, selection.want.Harness, document.Agent.Harness)
			require.Equal(t, selection.want.Model, document.Agent.Model)
			require.Equal(t, selection.want.Credentials, document.Agent.Credentials)
			require.NotNil(t, document.Agent.SkipSetupCommands)
			require.False(t, *document.Agent.SkipSetupCommands)
			require.NotNil(t, document.Agent.SkipRepositoryCheckout)
			require.True(t, *document.Agent.SkipRepositoryCheckout)
		})
	}
}
