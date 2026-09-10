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
	assert.Contains(t, result.canvasYAML, "{{ task().description }}")
	assert.NotContains(t, result.canvasYAML, `title == "PLAN.md"`)
	assert.NotContains(t, result.canvasYAML, "Implementation plan:")

	createPR := findYAMLNode(t, canvas, "create-pr")
	assert.Equal(t, "{{ task().repository }}", createPR.Configuration["repository"])
	assert.Equal(t, "{{ task().default_branch }}", createPR.Configuration["base"])
	assert.Equal(t, &yaml.IntegrationRef{ID: "github-1", Name: "acme-github"}, createPR.Integration)

	body, ok := createPR.Configuration["body"].(string)
	require.True(t, ok, "expected create-pr body to be a string")
	assert.Contains(t, body, `task().origin != nil ? "Closes " + task().origin.label : ""`)
	assert.Contains(t, body, "[{{ task().key }}]({{ task().url }})")
	assert.Less(t, strings.Index(body, "Closes"), strings.Index(body, "task().key"), "body: %s", body)
	assert.NotContains(t, body, "[Task](")

	updatePR := findYAMLNode(t, canvas, "update-pr")
	updateBody, ok := updatePR.Configuration["body"].(string)
	require.True(t, ok, "expected update-pr body to be a string")
	assert.Contains(t, updateBody, `task().origin != nil ? "Closes " + task().origin.label : ""`)

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

func TestMaterializePRCIosureTemplate_HasOriginClosingNodes(t *testing.T) {
	result, err := materializeFactoryTemplate("pr-closure", factoryTemplateInput{
		appID:   "app-1",
		appName: "PR Closure",
		installParams: map[string]string{
			"appRepository": "acme/app",
			"defaultBranch": "main",
		},
		integrations: map[string]factoryTemplateIntegration{
			"github": {id: "github-1", name: "acme-github"},
		},
	})
	require.NoError(t, err)

	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)

	hasOrigin := findYAMLNode(t, canvas, "has-github-origin")
	assert.Equal(t, "if", hasOrigin.Component)
	assert.Contains(t, hasOrigin.Configuration["expression"], "originRepository")

	comment := findYAMLNode(t, canvas, "comment-on-origin")
	assert.Equal(t, "github.createIssueComment", comment.Component)
	assert.Equal(t, "{{ $[\"Find Pull Request\"].data.workOrder.originRepository }}", comment.Configuration["repository"])
	assert.Equal(t, "{{ $[\"Find Pull Request\"].data.workOrder.originNumber }}", comment.Configuration["issueNumber"])
	assert.Equal(t, "{{ $[\"complete-work-order\"] != nil ? \"SuperPlane completed this task.\" : \"SuperPlane closed this task.\" }}", comment.Configuration["body"])
	assert.Equal(t, &yaml.IntegrationRef{ID: "github-1", Name: "acme-github"}, comment.Integration)

	closeOrigin := findYAMLNode(t, canvas, "close-origin")
	assert.Equal(t, "github.updateIssue", closeOrigin.Component)
	assert.Equal(t, "{{ $[\"Find Pull Request\"].data.workOrder.originRepository }}", closeOrigin.Configuration["repository"])
	assert.Equal(t, "{{ $[\"Find Pull Request\"].data.workOrder.originNumber }}", closeOrigin.Configuration["issueNumber"])
	assert.Equal(t, "closed", closeOrigin.Configuration["state"])
	assert.Equal(t, &yaml.IntegrationRef{ID: "github-1", Name: "acme-github"}, closeOrigin.Integration)

	// Edges from both status nodes to has-github-origin
	assertEdge(t, canvas, "complete-work-order", "has-github-origin", "default")
	assertEdge(t, canvas, "reject-work-order", "has-github-origin", "default")
	assertEdge(t, canvas, "has-github-origin", "comment-on-origin", "true")
	assertEdge(t, canvas, "comment-on-origin", "close-origin", "default")
}

func assertEdge(t *testing.T, canvas *yaml.Canvas, sourceID, targetID, channel string) {
	t.Helper()
	for _, edge := range canvas.Spec.Edges {
		if edge.SourceID == sourceID && edge.TargetID == targetID && edge.Channel == channel {
			return
		}
	}
	t.Fatalf("edge not found: %s -> %s (channel: %s)", sourceID, targetID, channel)
}

func TestMaterializeFactoryTemplateRejectsRetiredPlan(t *testing.T) {
	_, err := materializeFactoryTemplate("line-planning", factoryTemplateInput{
		appID:   "app-1",
		appName: "Plan",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown factory app template")
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
	assert.Equal(t, models.SuperPlaneRunnerComponent, agent.Component)
	assert.Nil(t, agent.Configuration["model"])
	assert.Nil(t, agent.Configuration["credentials"])
	require.NotNil(t, agent.Concurrency)
	require.NotNil(t, agent.Concurrency.Max)
	assert.Equal(t, 10, *agent.Concurrency.Max)
}

func TestDeriveFactoryInstallParamsSkipsRuntimeExpressions(t *testing.T) {
	params := deriveFactoryInstallParams([]models.Node{
		{
			ID: "find-pull-request",
			Configuration: map[string]any{
				"repository": "{{ root().data.repository.full_name }}",
				"base":       "{{ root().data.pull_request.base.ref }}",
			},
		},
		{
			ID: "runner",
			Configuration: map[string]any{
				"environment": []any{
					map[string]any{"name": "REPO", "value": "{{ install_params.appRepository }}"},
					map[string]any{"name": "BASE", "value": "{{ install_params.defaultBranch }}"},
				},
			},
		},
		{
			ID: "on-pr-closed",
			Configuration: map[string]any{
				"repository": "acme/widgets",
				"base":       "release",
			},
		},
	})

	assert.Equal(t, "acme/widgets", params["appRepository"])
	assert.Equal(t, "acme/widgets", params["backlogRepository"])
	assert.Equal(t, "release", params["defaultBranch"])
}

func TestDeriveFactoryInstallParamsWithOnlyExpressionsDerivesNothing(t *testing.T) {
	params := deriveFactoryInstallParams([]models.Node{
		{
			ID: "find-pull-request",
			Configuration: map[string]any{
				"repository": "{{ root().data.repository.full_name }}",
			},
		},
	})

	assert.Empty(t, params)
}

func TestMaterializeIntakeDefaults(t *testing.T) {
	source := models.FactoryIntakeSourceGitHubIssues
	current, err := buildIntakeCanvas(intakeCanvasRequest{Source: source})
	require.NoError(t, err)

	settings := intakeSettings{
		ConfidencePct:        80,
		Labels:               []string{"factory", "urgent"},
		LabelFilterMode:      "include",
		Assignment:           "assigned",
		AuthorsWithAccess:    true,
		SuperplaneLabelAdded: true,
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

	result, err := materializeBacklogDefaults(nil, nil, canvas, version)
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
