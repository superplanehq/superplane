package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestReconcileGeneratedFactoryAgentNodesUpdatesPlanningRunner(t *testing.T) {
	nodes := []models.Node{
		{
			ID:  "onrun-create-plan",
			Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}},
			Metadata: map[string]any{
				factoryTemplateMetadataKey: map[string]any{"id": "line-planning", "version": factoryTemplateVersion},
			},
		},
		{
			ID:            "planner-agent-no-issue",
			Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}},
			Configuration: map[string]any{"model": "opus", "credentials": map[string]any{"source": "integration"}},
		},
	}

	changed := reconcileGeneratedFactoryAgentNodes(nodes, factoryAgentPlan{
		component: "runnerCodex", integrationID: "integration-1", integrationName: "openai-production", model: "gpt-5", planningModel: "gpt-5",
	})

	require.True(t, changed)
	assert.Equal(t, "runnerCodex", nodes[1].ComponentName())
	assert.Equal(t, "gpt-5", nodes[1].Configuration["model"])
	assert.Equal(t, map[string]any{
		"source":      "integration",
		"integration": map[string]any{"name": "openai-production"},
	}, nodes[1].Configuration["credentials"])
}

func TestReconcileGeneratedFactoryAgentNodesUpdatesBacklogRunner(t *testing.T) {
	nodes := []models.Node{
		{ID: backlogTriggerNodeID, Name: backlogTriggerName, Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "onWorkOrder"}}},
		{
			ID: intakeAnalysisNodeID, Name: intakeAnalysisNodeName,
			Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}},
			Configuration: map[string]any{"model": "opus"},
		},
	}

	changed := reconcileGeneratedFactoryAgentNodes(nodes, factoryAgentPlan{component: "runnerOpenRouter", model: "anthropic/claude-sonnet-4-6"})

	require.True(t, changed)
	assert.Equal(t, "runnerOpenRouter", nodes[1].ComponentName())
	assert.Equal(t, "anthropic/claude-sonnet-4-6", nodes[1].Configuration["model"])
	assert.Equal(t, map[string]any{"source": "hosted"}, nodes[1].Configuration["credentials"])
}

func TestReconcileGeneratedFactoryAgentNodesLeavesCustomRunnerUnchanged(t *testing.T) {
	nodes := []models.Node{
		{
			ID:            "custom-agent",
			Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}},
			Configuration: map[string]any{"model": "custom-model"},
		},
	}

	changed := reconcileGeneratedFactoryAgentNodes(nodes, factoryAgentPlan{component: "runnerCodex", model: "gpt-5"})

	assert.False(t, changed)
	assert.Equal(t, "runnerClaudeCode", nodes[0].ComponentName())
	assert.Equal(t, "custom-model", nodes[0].Configuration["model"])
}

func TestFactoryHostedAgentModelsUsesProviderDefaults(t *testing.T) {
	model, planningModel, ok := factoryHostedAgentModels([]string{"claude-haiku", "claude-opus", "claude-sonnet"}, factoryAgentProviderSpecs["anthropic"])

	require.True(t, ok)
	assert.Equal(t, "claude-sonnet", model)
	assert.Equal(t, "claude-opus", planningModel)
}
