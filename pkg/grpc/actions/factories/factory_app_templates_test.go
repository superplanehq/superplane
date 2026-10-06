package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/expressionvalidation"
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
	assert.Equal(t, false, agent.Configuration["includeVisualEvidence"])
	assert.Contains(t, result.canvasYAML, "{{ task().description }}")
	assert.Contains(t, result.canvasYAML, `task().spec != "" ? "\n\nSpec:\n" + task().spec : ""`)
	assert.NotContains(t, result.canvasYAML, `title == "PLAN.md"`)
	assert.NotContains(t, result.canvasYAML, "Implementation plan:")
	assert.NotContains(t, result.canvasYAML, "task().visual_evidence_enabled")
	assert.NotContains(t, result.canvasYAML, "For a visual-only change")
	assert.NotContains(t, result.canvasYAML, "report_visual_evidence_unavailable")
	assert.NotContains(t, result.canvasYAML, "Visual evidence is unavailable:")
	assert.NotContains(t, result.canvasYAML, "setWorkOrderStatusNote")
	assert.NotContains(t, result.canvasYAML, "set-pr-closure-note")
	assert.NotContains(t, result.canvasYAML, "x-access-token")
	assert.NotContains(t, result.canvasYAML, `git rev-parse '@{upstream}'`)
	assert.NotContains(t, result.canvasYAML, "COMMIT_SHA")
	assert.Contains(t, result.canvasYAML, "implement and wire the change into the production page or component")
	assert.Contains(t, result.canvasYAML, "does not replace the product implementation unless the task explicitly requests")
	implementationPrompt, ok := implementationStep(t, agent, "Implementation")["prompt"].(string)
	require.True(t, ok)
	assert.Contains(t, implementationPrompt, runner.FactoryCommitIdentityPrompt)
	assert.Contains(t, implementationPrompt, runner.FactoryImaginedLimitPrompt)
	cloneCommand, ok := implementationStep(t, agent, "Clone Repo")["command"].(string)
	require.True(t, ok)
	assert.Contains(t, cloneCommand, runner.FactoryRepoCommitSetup())
	assert.Contains(t, result.canvasYAML, `title: ($title | gsub("[\\r\\n]"; "") | @base64)`)

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

	commentEvidence := findYAMLNode(t, canvas, "comment-visual-evidence")
	assert.Equal(t, "github.createIssueComment", commentEvidence.Component)
	assert.Equal(t, &yaml.IntegrationRef{ID: "github-1", Name: "acme-github"}, commentEvidence.Integration)
	assert.Contains(t, commentEvidence.Configuration["body"], `$["Create Pull Request"].data.head.sha[:7]`)
	assert.Contains(t, commentEvidence.Configuration["body"], "## Visual evidence")
	assert.Contains(t, canvas.Spec.Edges, yaml.Edge{SourceID: "attach-pr-artifact", TargetID: "has-visual-evidence", Channel: "default"})
	hasEvidence := findYAMLNode(t, canvas, "has-visual-evidence")
	assert.Equal(
		t,
		`($["Implement From Task Description"].data.result.visualEvidence.status == "captured" || $["Implement From Task Description"].data.result.visualEvidence.status == "partial") && len($["Implement From Task Description"].data.result.visualEvidence.artifacts) > 0`,
		hasEvidence.Configuration["expression"],
	)
	hasUpdatedEvidence := findYAMLNode(t, canvas, "has-visual-evidence-updated")
	assert.Equal(
		t,
		`($["Implement From Task Description"].data.result.visualEvidence.status == "captured" || $["Implement From Task Description"].data.result.visualEvidence.status == "partial") && len($["Implement From Task Description"].data.result.visualEvidence.artifacts) > 0`,
		hasUpdatedEvidence.Configuration["expression"],
	)
	updatedCommentEvidence := findYAMLNode(t, canvas, "comment-visual-evidence-updated")
	assert.Contains(t, updatedCommentEvidence.Configuration["body"], `$["Update Pull Request"].data.head.sha[:7]`)

	updatePR := findYAMLNode(t, canvas, "update-pr")
	updateBody, ok := updatePR.Configuration["body"].(string)
	require.True(t, ok, "expected update-pr body to be a string")
	assert.Contains(t, updateBody, `task().origin != nil ? "Closes " + task().origin.label : ""`)

	console, err := yaml.ConsoleFromYML([]byte(result.consoleYAML))
	require.NoError(t, err)
	assert.Equal(t, "app-1", console.Metadata.CanvasID)
	assert.Equal(t, "Implement refunds", console.Metadata.Name)

	requireValidCanvasExpressions(t, canvas)
}

func TestMaterializeRiskScoreTemplate(t *testing.T) {
	result, err := materializeFactoryTemplate("risk-score", factoryTemplateInput{
		appID:   "app-risk",
		appName: "Merge confidence",
		installParams: map[string]string{
			"appRepository": "acme/app",
			"defaultBranch": "main",
		},
		integrations: map[string]factoryTemplateIntegration{
			"github": {id: "github-1", name: "acme-github"},
		},
		agent: &factoryTemplateAgent{
			component:        models.SuperPlaneRunnerComponent,
			credentialSource: "hosted",
		},
	})
	require.NoError(t, err)

	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	assert.Equal(t, "Merge confidence", canvas.Metadata.Name)

	entrypoint := findYAMLNode(t, canvas, "on-pr-risk")
	assert.Equal(t, map[string]any{
		"id":      "risk-score",
		"version": float64(factoryTemplateVersion),
	}, entrypoint.Metadata[factoryTemplateMetadataKey])
	assert.Equal(t, "github.onPullRequest", entrypoint.Component)
	assert.Equal(t, true, entrypoint.Configuration["onlyFactoryPullRequests"])
	assert.Equal(t, "acme/app", entrypoint.Configuration["repository"])
	assert.Equal(t, true, entrypoint.Configuration["ignoreDrafts"])
	assert.Equal(t, &yaml.IntegrationRef{ID: "github-1", Name: "acme-github"}, entrypoint.Integration)
	assert.NotContains(t, result.canvasYAML, "should-assess")
	assert.Contains(t, canvas.Spec.Edges, yaml.Edge{SourceID: "on-pr-risk", TargetID: "assess-risk", Channel: "default"})

	agent := findYAMLNode(t, canvas, "assess-risk")
	assert.Equal(t, models.SuperPlaneRunnerComponent, agent.Component)
	assert.NotContains(t, agent.Configuration, "credentials")
	assert.NotContains(t, agent.Configuration, "model")
	prompt := agentPrompt(t, agent)
	assert.Contains(t, prompt, "Additive database changes = 3 (medium).")
	assert.Contains(t, prompt, "Authorization changes = 4 (high).")
	assert.Contains(t, prompt, "Enabled checks: risk, performance, security, drift, reversibility.")
	assert.Contains(t, prompt, "report_merge_check")
	assert.NotContains(t, prompt, "install_params.riskRules")
	assert.NotContains(t, prompt, "install_params.enabledChecks")
	assert.Contains(t, result.canvasYAML, `gh api -H "Accept: application/vnd.github.v3.diff"`)
	assert.Contains(t, result.canvasYAML, `/compare/${base_ref}...${revision}" > /tmp/pr.diff`)
	assert.Contains(t, result.canvasYAML, "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID")
	assert.Contains(t, result.canvasYAML, `root().data.workOrder.id`)
	assert.NotContains(t, result.canvasYAML, "findPullRequest")
	assert.NotContains(t, result.canvasYAML, "reportWorkOrderCheck")
	assert.NotContains(t, result.canvasYAML, "merge-confidence.json")
	assert.Len(t, canvas.Spec.Nodes, 2)

	assert.NotContains(t, result.canvasYAML, "github.createIssueComment")
	assert.NotContains(t, result.canvasYAML, "github.publishCommitStatus")
	assert.NotContains(t, result.canvasYAML, "superplane-pr-risk-review")
	assert.NotContains(t, result.canvasYAML, "discord")
	assert.NotContains(t, result.canvasYAML, "git push")
	assert.Contains(t, result.consoleYAML, "app-risk")

	requireValidCanvasExpressions(t, canvas)
}

func TestMaterializeLineImplementationKeepsVisualEvidence(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(visualEvidenceCaseName(enabled), func(t *testing.T) {
			seed, err := materializeFactoryTemplate("line-implementation", factoryTemplateInput{
				appID:   "app-1",
				appName: "Implement",
			})
			require.NoError(t, err)
			live, err := yaml.CanvasFromYAML([]byte(seed.canvasYAML))
			require.NoError(t, err)
			findYAMLNode(t, live, "implementation-agent-no-issue").Configuration["includeVisualEvidence"] = enabled

			includeVisualEvidence := canvasAgentIncludesVisualEvidence(live.Nodes())
			assert.Equal(t, enabled, includeVisualEvidence)

			result, err := materializeFactoryTemplate("line-implementation", factoryTemplateInput{
				appID:                 "app-1",
				appName:               "Implement",
				includeVisualEvidence: &includeVisualEvidence,
			})
			require.NoError(t, err)
			defaults, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
			require.NoError(t, err)
			assert.Equal(
				t,
				enabled,
				findYAMLNode(t, defaults, "implementation-agent-no-issue").Configuration["includeVisualEvidence"],
			)
		})
	}
}

func TestDeriveFactoryTemplateInputCarriesVisualEvidence(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(visualEvidenceCaseName(enabled), func(t *testing.T) {
			input := deriveFactoryTemplateInput(
				nil,
				nil,
				&models.Canvas{ID: uuid.New(), Name: "Implement"},
				&models.CanvasVersion{Nodes: []models.Node{{
					ID:  implementationAgentNodeID,
					Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}},
					Configuration: map[string]any{
						"includeVisualEvidence": enabled,
					},
				}}},
				factoryAppTemplates["line-implementation"],
			)
			require.NotNil(t, input.includeVisualEvidence)
			assert.Equal(t, enabled, *input.includeVisualEvidence)
		})
	}
}

func TestCanvasAgentIncludesVisualEvidenceIgnoresUnrelatedRunners(t *testing.T) {
	nodes := []models.Node{
		{
			ID:  "extra-runner",
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerOpenRouter"}},
			Configuration: map[string]any{
				"includeVisualEvidence": true,
			},
		},
		{
			ID:  implementationAgentNodeID,
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}},
			Configuration: map[string]any{
				"includeVisualEvidence": false,
			},
		},
	}

	assert.False(t, canvasAgentIncludesVisualEvidence(nodes))

	input := deriveFactoryTemplateInput(
		nil,
		nil,
		&models.Canvas{ID: uuid.New(), Name: "Implement"},
		&models.CanvasVersion{Nodes: nodes},
		factoryAppTemplates["line-implementation"],
	)
	require.NotNil(t, input.includeVisualEvidence)
	assert.False(t, *input.includeVisualEvidence)
}

func TestMaterializePRFeedbackDefaultsRemovesSeparateEvidenceComment(t *testing.T) {
	canvasID := uuid.New()
	current := buildDiscussionPRFeedbackCanvas(prFeedbackBuildRequest{
		Repository: "acme/app",
		Agent: &intakeAgent{
			Component: "runnerOpenRouter",
			Model:     "anthropic/claude-sonnet-4-6",
		},
	})
	for _, nodeID := range []string{
		prFeedbackRunnerNodeID,
		prFeedbackReviewRunnerNodeID,
		prFeedbackReplyRunnerNodeID,
	} {
		findYAMLNode(t, current, nodeID).Configuration["includeVisualEvidence"] = true
	}
	legacyEvidenceFlows := []struct {
		runnerID  string
		gateID    string
		commentID string
	}{
		{prFeedbackRunnerNodeID, "has-pr-comment-visual-evidence", "comment-pr-comment-visual-evidence"},
		{prFeedbackReviewRunnerNodeID, "has-pr-review-visual-evidence", "comment-pr-review-visual-evidence"},
		{prFeedbackReplyRunnerNodeID, "has-pr-review-reply-visual-evidence", "comment-pr-review-reply-visual-evidence"},
	}
	legacyEvidenceNodeIDs := make([]string, 0, 2*len(legacyEvidenceFlows))
	for _, flow := range legacyEvidenceFlows {
		legacyEvidenceNodeIDs = append(legacyEvidenceNodeIDs, flow.gateID, flow.commentID)
		current.Spec.Nodes = append(current.Spec.Nodes,
			yaml.Node{ID: flow.gateID, Type: yaml.NodeTypeAction, Component: "if"},
			yaml.Node{ID: flow.commentID, Type: yaml.NodeTypeAction, Component: "github.createIssueComment"},
		)
		current.Spec.Edges = append(current.Spec.Edges,
			yaml.Edge{SourceID: flow.runnerID, TargetID: flow.gateID, Channel: "passed"},
			yaml.Edge{SourceID: flow.gateID, TargetID: flow.commentID, Channel: "true"},
		)
	}

	result, err := materializePRFeedbackDefaults(
		nil,
		&models.Factory{},
		&models.Canvas{ID: canvasID, Name: "Address PR feedback"},
		&models.CanvasVersion{Nodes: current.Nodes(), Edges: current.Edges()},
		&models.FactoryPRFeedbackHandler{Source: models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion},
	)
	require.NoError(t, err)

	defaults, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	for _, nodeID := range []string{
		prFeedbackRunnerNodeID,
		prFeedbackReviewRunnerNodeID,
		prFeedbackReplyRunnerNodeID,
	} {
		assert.Equal(t, true, findYAMLNode(t, defaults, nodeID).Configuration["includeVisualEvidence"])
	}
	for _, node := range defaults.Spec.Nodes {
		assert.NotContains(t, legacyEvidenceNodeIDs, node.ID)
	}
}

func TestMaterializePRFeedbackChecksDefaultsKeepsVisualEvidence(t *testing.T) {
	canvasID := uuid.New()
	current := buildChecksPRFeedbackCanvas(prFeedbackBuildRequest{
		Repository:            "acme/app",
		IncludeVisualEvidence: true,
		Agent: &intakeAgent{
			Component: "runnerOpenRouter",
			Model:     "anthropic/claude-sonnet-4-6",
		},
	})
	assert.Equal(t, true, findYAMLNode(t, current, prFeedbackRunnerNodeID).Configuration["includeVisualEvidence"])

	result, err := materializePRFeedbackDefaults(
		nil,
		&models.Factory{},
		&models.Canvas{ID: canvasID, Name: "Fix pull request checks"},
		&models.CanvasVersion{Nodes: current.Nodes(), Edges: current.Edges()},
		&models.FactoryPRFeedbackHandler{Source: models.FactoryPRFeedbackHandlerSourcePullRequestChecks},
	)
	require.NoError(t, err)

	defaults, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	assert.Equal(t, true, findYAMLNode(t, defaults, prFeedbackRunnerNodeID).Configuration["includeVisualEvidence"])
}

func visualEvidenceCaseName(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func TestMaterializePRClosureClosesGitHubOriginAfterMerge(t *testing.T) {
	result, err := materializeFactoryTemplate("pr-closure", factoryTemplateInput{
		appID:   "app-1",
		appName: "PR Closure",
		installParams: map[string]string{
			"appRepository": "acme/refunds",
		},
		integrations: map[string]factoryTemplateIntegration{
			"github": {id: "github-1", name: "acme-github"},
		},
	})
	require.NoError(t, err)

	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)

	hasGitHubOrigin := findYAMLNode(t, canvas, "has-github-issue-origin")
	assert.Equal(t, "if", hasGitHubOrigin.Component)
	assert.Equal(
		t,
		`root().data.workOrder.origin != nil && split(root().data.workOrder.origin.url, "https://github.com/")[0] == "" && len(split(root().data.workOrder.origin.url, "/issues/")) == 2`,
		hasGitHubOrigin.Configuration["expression"],
	)

	comment := findYAMLNode(t, canvas, "comment-source-issue")
	assert.Equal(t, "github.createIssueComment", comment.Component)
	assert.Equal(t, &yaml.IntegrationRef{ID: "github-1", Name: "acme-github"}, comment.Integration)
	assert.Equal(t, `{{ split(split(root().data.workOrder.origin.url, "https://github.com/")[1], "/issues/")[0] }}`, comment.Configuration["repository"])
	assert.Equal(t, `{{ split(root().data.workOrder.origin.url, "/issues/")[1] }}`, comment.Configuration["issueNumber"])
	assert.Equal(
		t,
		`SuperPlane completed this task in pull request [#{{ root().data.pull_request.number }}]({{ root().data.pull_request.html_url }}).`,
		comment.Configuration["body"],
	)

	closeIssue := findYAMLNode(t, canvas, "close-source-issue")
	assert.Equal(t, "github.updateIssue", closeIssue.Component)
	assert.Equal(t, &yaml.IntegrationRef{ID: "github-1", Name: "acme-github"}, closeIssue.Integration)
	assert.Equal(t, comment.Configuration["repository"], closeIssue.Configuration["repository"])
	assert.Equal(t, comment.Configuration["issueNumber"], closeIssue.Configuration["issueNumber"])
	assert.Equal(t, "closed", closeIssue.Configuration["state"])

	assert.Contains(t, canvas.Spec.Edges, yaml.Edge{SourceID: "complete-work-order", TargetID: "has-github-issue-origin", Channel: "default"})
	assert.Contains(t, canvas.Spec.Edges, yaml.Edge{SourceID: "has-github-issue-origin", TargetID: "comment-source-issue", Channel: "true"})
	assert.Contains(t, canvas.Spec.Edges, yaml.Edge{SourceID: "comment-source-issue", TargetID: "close-source-issue", Channel: "default"})
	assert.NotContains(t, canvas.Spec.Edges, yaml.Edge{SourceID: "reject-work-order", TargetID: "has-github-issue-origin", Channel: "default"})
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

func TestMaterializeFactoryTemplateRejectsRetiredPlan(t *testing.T) {
	_, err := materializeFactoryTemplate("line-planning", factoryTemplateInput{
		appID:   "app-1",
		appName: "Plan",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown factory app template")
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
	refinement := findYAMLNode(t, defaults, backlogRefinementNodeID)
	assert.Equal(t, "runnerOpenRouter", refinement.Component)
	assert.Equal(t, "anthropic/claude-opus-4-6", refinement.Configuration["model"])
	assert.Equal(t, map[string]any{
		"source":      "integration",
		"integration": map[string]any{"name": "acme-openrouter"},
	}, refinement.Configuration["credentials"])
	prompt := implementationStep(t, refinement, "Refine Task")
	text, ok := prompt["prompt"].(string)
	require.True(t, ok)
	assert.Contains(t, text, runner.PlanningSessionUserPromptMarkdown())
	assert.Contains(t, text, "Task:\n{{ root().data.workOrder }}")
}

func TestBuildBacklogCanvasPlanningReviewPrompt(t *testing.T) {
	legacy := backlogRefinementPromptText(t, buildBacklogCanvas(backlogCanvasRequest{}))
	assert.Contains(t, legacy, runner.PlanningSessionUserPromptMarkdown())
	assert.NotContains(t, legacy, "Clarity, Complexity, and Verifiability")

	review := backlogRefinementPromptText(t, buildBacklogCanvas(backlogCanvasRequest{PlanningReview: true}))
	assert.Contains(t, review, runner.PlanningSessionUserPromptReviewMarkdown())
	assert.Contains(t, review, "Task:\n{{ root().data.workOrder }}")
	assert.NotContains(t, review, "1 through 5")
}

func backlogRefinementPromptText(t *testing.T, canvas *yaml.Canvas) string {
	t.Helper()
	refinement := findYAMLNode(t, canvas, backlogRefinementNodeID)
	text, ok := implementationStep(t, refinement, "Refine Task")["prompt"].(string)
	require.True(t, ok)
	return text
}

func agentPrompt(t *testing.T, agent *yaml.Node) string {
	t.Helper()
	prompt, ok := implementationStep(t, agent, "Review Pull Request")["prompt"].(string)
	require.True(t, ok)
	return prompt
}

func implementationStep(t *testing.T, agent *yaml.Node, name string) map[string]any {
	t.Helper()
	steps, ok := agent.Configuration["steps"].([]any)
	require.True(t, ok)
	for _, step := range steps {
		item, ok := step.(map[string]any)
		if !ok || item["name"] != name {
			continue
		}
		return item
	}
	t.Fatalf("step %q not found", name)
	return nil
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

func requireValidCanvasExpressions(t *testing.T, canvas *yaml.Canvas) {
	t.Helper()

	knownNodeNames := make(map[string]struct{}, len(canvas.Spec.Nodes))
	for _, node := range canvas.Spec.Nodes {
		if node.Name == "" {
			continue
		}
		knownNodeNames[node.Name] = struct{}{}
	}

	checked := 0
	for _, node := range canvas.Spec.Nodes {
		checked += requireValidConfigurationExpressions(t, node.Configuration, knownNodeNames)
	}
	require.Greater(t, checked, 0)
}

func requireValidConfigurationExpressions(t *testing.T, value any, knownNodeNames map[string]struct{}) int {
	t.Helper()

	switch v := value.(type) {
	case string:
		return requireValidTemplateExpressionsWithNodes(t, v, knownNodeNames)
	case map[string]any:
		checked := 0
		for _, child := range v {
			checked += requireValidConfigurationExpressions(t, child, knownNodeNames)
		}
		return checked
	case []any:
		checked := 0
		for _, child := range v {
			checked += requireValidConfigurationExpressions(t, child, knownNodeNames)
		}
		return checked
	default:
		return 0
	}
}

func requireValidTemplateExpressionsWithNodes(t *testing.T, value string, knownNodeNames map[string]struct{}) int {
	t.Helper()

	matches := configuration.ExpressionPlaceholderRegex.FindAllString(value, -1)
	for _, match := range matches {
		source := match[2 : len(match)-2]
		require.NoError(t, expressionvalidation.ValidateExpression(source, knownNodeNames), match)
	}
	return len(matches)
}
