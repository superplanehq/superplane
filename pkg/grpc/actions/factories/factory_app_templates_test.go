package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func TestMaterializeFactoryTemplate(t *testing.T) {
	result, err := materializeFactoryTemplate("line-implementation", factoryTemplateInput{
		appID:   "app-1",
		appName: "Implement refunds",
		installParams: map[string]string{
			"appRepository": "acme/refunds",
			"defaultBranch": "develop",
		},
		integrations: map[string]factoryTemplateIntegration{
			"github": {id: "github-1", name: "acme-github"},
		},
		agent: &factoryTemplateAgent{
			component:                 "runnerOpenRouter",
			model:                     "anthropic/claude-sonnet-4-6",
			credentialSource:          "integration",
			credentialIntegrationName: "acme-openrouter",
		},
	})
	require.NoError(t, err)

	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	assert.Equal(t, "Implement refunds", canvas.Metadata.Name)

	entrypoint := findYAMLNode(t, canvas, "onrun-implement")
	assert.Equal(t, map[string]any{
		"id":      "line-implementation",
		"version": float64(factoryTemplateVersion),
	}, entrypoint.Metadata[factoryTemplateMetadataKey])

	agent := findYAMLNode(t, canvas, "implementation-agent-no-issue")
	assert.Equal(t, "runnerOpenRouter", agent.Component)
	assert.Equal(t, "anthropic/claude-sonnet-4-6", agent.Configuration["model"])
	assert.Equal(t, map[string]any{
		"source":      "integration",
		"integration": map[string]any{"name": "acme-openrouter"},
	}, agent.Configuration["credentials"])

	createPR := findYAMLNode(t, canvas, "create-pr")
	assert.Equal(t, "{{ task().repository }}", createPR.Configuration["repository"])
	assert.Equal(t, "{{ task().default_branch }}", createPR.Configuration["base"])
	assert.Equal(t, &yaml.IntegrationRef{ID: "github-1", Name: "acme-github"}, createPR.Integration)

	body, ok := createPR.Configuration["body"].(string)
	require.True(t, ok, "expected create-pr body to be a string")
	assert.True(t, strings.HasPrefix(body, "[{{ task().key }}]({{ task().url }})"), "body: %s", body)
	assert.NotContains(t, body, "[Task](")

	console, err := yaml.ConsoleFromYML([]byte(result.consoleYAML))
	require.NoError(t, err)
	assert.Equal(t, "app-1", console.Metadata.CanvasID)
	assert.Equal(t, "Implement refunds", console.Metadata.Name)
}

func TestMaterializeFactoryTemplates(t *testing.T) {
	for id := range factoryAppTemplates {
		t.Run(id, func(t *testing.T) {
			result, err := materializeFactoryTemplate(id, factoryTemplateInput{
				appID:   "app-1",
				appName: id,
				installParams: map[string]string{
					"appRepository":     "acme/app",
					"backlogRepository": "acme/backlog",
					"defaultBranch":     "main",
				},
				integrations: map[string]factoryTemplateIntegration{
					"github": {id: "github-1", name: "acme-github"},
				},
			})
			require.NoError(t, err)
			_, err = yaml.CanvasFromYAML([]byte(result.canvasYAML))
			require.NoError(t, err)
			_, err = yaml.ConsoleFromYML([]byte(result.consoleYAML))
			require.NoError(t, err)
		})
	}
}

func TestMaterializePlanningTemplateUsesPlanningModel(t *testing.T) {
	result, err := materializeFactoryTemplate("line-planning", factoryTemplateInput{
		appID:   "app-1",
		appName: "Plan",
		installParams: map[string]string{
			"appRepository": "acme/app",
		},
		agent: &factoryTemplateAgent{
			component:        "runnerClaudeCode",
			model:            "claude-sonnet-4-6",
			planningModel:    "claude-opus-4-6",
			credentialSource: "hosted",
		},
	})
	require.NoError(t, err)
	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4-6", findYAMLNode(t, canvas, "planner-agent-no-issue").Configuration["model"])
}

func TestMaterializeCreateWithAgentUsesPlanningModel(t *testing.T) {
	result, err := materializeFactoryTemplate("create-with-agent", factoryTemplateInput{
		appID:   "app-1",
		appName: "Create with an Agent",
		installParams: map[string]string{
			"appRepository": "acme/app",
		},
		agent: &factoryTemplateAgent{
			component:        "runnerClaudeCode",
			model:            "claude-sonnet-4-6",
			planningModel:    "claude-opus-4-6",
			credentialSource: "hosted",
		},
	})
	require.NoError(t, err)
	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	assert.Equal(t, "Create with an Agent", canvas.Metadata.Name)
	agent := findYAMLNode(t, canvas, "planning-agent")
	assert.Equal(t, "claude-opus-4-6", agent.Configuration["model"])
	require.NotNil(t, agent.Concurrency)
	require.NotNil(t, agent.Concurrency.Max)
	assert.Equal(t, 10, *agent.Concurrency.Max)
}

func TestMaterializeIntakeDefaults(t *testing.T) {
	source := models.FactoryIntakeSourceGitHubIssues
	current, err := buildIntakeCanvas(intakeCanvasRequest{Source: source})
	require.NoError(t, err)

	settings := intakeSettings{
		ConfidencePct:     80,
		Labels:            []string{"factory", "urgent"},
		LabelFilterMode:   "include",
		Assignment:        "assigned",
		AuthorsWithAccess: true,
	}
	findYAMLNode(t, current, intakeFilterNodeID).Configuration["expression"] = intakeFilterExpressionFor(source, settings)

	factoryID := uuid.New()
	canvasID := uuid.New()
	canvas := &models.Canvas{
		ID:             canvasID,
		OrganizationID: uuid.New(),
		FactoryID:      &factoryID,
		Name:           "GitHub backlog",
	}
	version := &models.CanvasVersion{
		Nodes: current.Nodes(),
		Edges: current.Edges(),
	}
	intake := &models.FactoryIntake{
		FactoryID: factoryID,
		CanvasID:  canvasID,
		Source:    source,
	}

	result, err := materializeIntakeDefaults(nil, canvas, version, intake)
	require.NoError(t, err)
	assert.Equal(t, "intake:"+source, result.templateID)

	defaults, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	assert.Equal(t, canvasID.String(), defaults.Metadata.ID)
	assert.Equal(t, "GitHub backlog", defaults.Metadata.Name)
	assert.Equal(
		t,
		intakeFilterExpressionFor(source, settings),
		findYAMLNode(t, defaults, intakeFilterNodeID).Configuration["expression"],
	)
}

func TestMaterializeBacklogDefaults(t *testing.T) {
	canvasID := uuid.New()
	canvas := &models.Canvas{ID: canvasID, Name: "Backlog scoring"}
	current := buildBacklogCanvas(backlogCanvasRequest{
		Agent: &intakeAgent{
			Component: "runnerOpenRouter",
			Model:     "anthropic/claude-opus-4-6",
			Credentials: map[string]any{
				"source":      "integration",
				"integration": map[string]any{"name": "acme-openrouter"},
			},
		},
	})
	version := &models.CanvasVersion{
		Nodes: current.Nodes(),
		Edges: current.Edges(),
	}

	result, err := materializeBacklogDefaults(canvas, version)
	require.NoError(t, err)
	assert.Equal(t, "backlog", result.templateID)

	defaults, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	assert.Equal(t, canvasID.String(), defaults.Metadata.ID)
	assert.Equal(t, "Backlog scoring", defaults.Metadata.Name)
	analysis := findYAMLNode(t, defaults, intakeAnalysisNodeID)
	assert.Equal(t, "runnerOpenRouter", analysis.Component)
	assert.Equal(t, "anthropic/claude-opus-4-6", analysis.Configuration["model"])
	assert.Equal(t, map[string]any{
		"source":      "integration",
		"integration": map[string]any{"name": "acme-openrouter"},
	}, analysis.Configuration["credentials"])
}

func findYAMLNode(t *testing.T, canvas *yaml.Canvas, id string) *yaml.Node {
	t.Helper()
	for i := range canvas.Spec.Nodes {
		if canvas.Spec.Nodes[i].ID == id {
			return &canvas.Spec.Nodes[i]
		}
	}
	t.Fatalf("node %q not found", id)
	return nil
}
