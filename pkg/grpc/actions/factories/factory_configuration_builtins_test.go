package factories

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	goyaml "gopkg.in/yaml.v3"
)

func TestFactoryConfigurationRejectsInvalidBuiltInModelWithoutWrites(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	found := false
	for _, stage := range manifest.Stages {
		for _, automation := range stage.Automations {
			if automation.Uses != "@superplane/implement" {
				continue
			}
			found = true
			agents := automation.With["agents"].(map[string]any)
			agent := agents[implementationAgentNodeID].(map[string]any)
			agent["model"] = "unknown-model"
		}
	}
	require.True(t, found)
	content, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	before := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.NotEmpty(t, plan.Diagnostics)
	require.Contains(t, plan.Diagnostics[0].Field, "agents."+implementationAgentNodeID)
	require.Contains(t, plan.Diagnostics[0].Message, "model")
	require.Empty(t, plan.BaseStateToken)
	_, err = f.apply(bundle, "invalid")
	require.Error(t, err)
	require.Equal(t, before, configurationResourceCounts(t))
}

func TestFactoryConfigurationBuiltInSettingsSurviveReset(t *testing.T) {
	f := newConfigurationFixture(t)
	canvas := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(doc *yaml.Canvas) {
		agent := findYAMLNode(t, doc, "assess-risk")
		agent.Configuration["additionalInstructions"] = "Keep the previous user instructions."
		agent.Configuration["skipSetupCommands"] = true
	})
	before := resetTestLiveVersion(t, canvas)
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.Key, LineId: f.line.ID.String()})
	require.NoError(t, err)
	var manifest map[string]any
	require.NoError(t, goyaml.Unmarshal([]byte(exported.Bundle.Files[0].Content), &manifest))
	found := false
	for _, rawStage := range manifest["stages"].([]any) {
		stage := rawStage.(map[string]any)
		automations, _ := stage["automations"].([]any)
		for _, rawAutomation := range automations {
			automation := rawAutomation.(map[string]any)
			if automation["uses"] != "@superplane/merge-confidence" {
				continue
			}
			found = true
			automation["with"] = map[string]any{"agents": map[string]any{"assess-risk": map[string]any{
				"additional_instructions": "", "model": "", "thinking_level": "low", "execution_timeout_seconds": 0,
				"skip_setup_commands": false, "skip_repository_checkout": true,
			}}}
		}
	}
	require.True(t, found, "Verify must export its built-in automation")
	encoded, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	exported.Bundle.Files[0].Content = string(encoded)
	plan := f.plan(t, exported.Bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes)
	applied, err := f.apply(exported.Bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	current := resetTestLiveVersion(t, canvas)
	agent := findIntakeNode(current.Nodes, "assess-risk")
	require.NotNil(t, agent)
	for field, expected := range map[string]any{
		"additionalInstructions": "", "model": "", "thinkingLevel": "low", "executionTimeoutSeconds": float64(0),
		"skipSetupCommands": false, "skipRepositoryCheckout": true,
	} {
		require.Equal(t, expected, agent.Configuration[field], field)
		findIntakeNode(before.Nodes, "assess-risk").Configuration[field] = expected
	}
	// Configuration application keeps the installed template and managed prompts.
	require.ElementsMatch(t, before.Nodes, current.Nodes)
	require.ElementsMatch(t, before.Edges, current.Edges)
	require.NoError(t, resetTestOrganization(t, f.r))
	agent = findIntakeNode(resetTestLiveVersion(t, canvas).Nodes, "assess-risk")
	require.NotNil(t, agent)
	require.Equal(t, "", agent.Configuration["additionalInstructions"])
	require.Equal(t, "", agent.Configuration["model"])
	require.Equal(t, "low", agent.Configuration["thinkingLevel"])
	require.EqualValues(t, 0, agent.Configuration["executionTimeoutSeconds"])
	require.Equal(t, false, agent.Configuration["skipSetupCommands"])
	require.Equal(t, true, agent.Configuration["skipRepositoryCheckout"])
}
