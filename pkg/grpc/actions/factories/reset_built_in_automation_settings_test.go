package factories

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestResetBuiltInAutomationPreservesImplementationSettings(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	cases := []struct {
		provider string
		max      int
		canvas   *models.Canvas
	}{
		{provider: models.ProviderGitHub, max: 7},
		{provider: models.ProviderBitbucket, max: 9},
	}
	for i := range cases {
		tc := &cases[i]
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		tc.canvas = resetTestTemplate(t, r, factory, "line-implementation", tc.provider, func(doc *yaml.Canvas) {
			agent := findYAMLNode(t, doc, implementationAgentNodeID)
			agent.Concurrency = &yaml.ConcurrencySpec{Max: &tc.max, Key: "team"}
			agent.Configuration["model"] = models.FormatHostedLLMModelKey(models.UsageProviderOpenRouter, "anthropic/claude-sonnet-4-6")
			agent.Configuration["thinkingLevel"] = "low"
			agent.Configuration["executionTimeoutSeconds"] = 0
			agent.Configuration["disabledAgentResourceIds"] = []any{}
			agent.Configuration["disabledAgentResourceTools"] = map[string]any{"github-resource": []any{"delete_issue"}}
			agent.Configuration["enabledAgentResourceTools"] = map[string]any{"github-resource": []any{"get_issue"}}
			implementationStep(t, agent, "Implementation")["prompt"] = "Retired implementation prompt"
			findYAMLNode(t, doc, "create-pr").Configuration["draft"] = true
		})
	}
	require.NoError(t, resetTestOrganization(t, r))
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			version := resetTestLiveVersion(t, tc.canvas)
			agent := findIntakeNode(version.Nodes, implementationAgentNodeID)
			require.NotNil(t, agent)
			assert.Equal(t, "runnerSuperPlane", agent.ComponentName())
			assert.Equal(t, models.FormatHostedLLMModelKey(models.UsageProviderOpenRouter, "anthropic/claude-sonnet-4-6"), agent.Configuration["model"])
			assert.Equal(t, "low", agent.Configuration["thinkingLevel"])
			assert.EqualValues(t, 0, agent.Configuration["executionTimeoutSeconds"])
			assert.Equal(t, []any{}, agent.Configuration["disabledAgentResourceIds"])
			assert.Equal(t, map[string]any{"github-resource": []any{"delete_issue"}}, agent.Configuration["disabledAgentResourceTools"])
			assert.Equal(t, map[string]any{"github-resource": []any{"get_issue"}}, agent.Configuration["enabledAgentResourceTools"])
			require.NotNil(t, agent.Concurrency)
			assert.Equal(t, tc.max, agent.Concurrency.EffectiveMax())
			assert.Equal(t, "team", agent.Concurrency.Key)
			assert.Equal(t, true, findIntakeNode(version.Nodes, "create-pr").Configuration["draft"])
			assert.NotContains(t, liveCanvasJSON(t, tc.canvas), "retired implementation prompt")
		})
	}
}

func resetTestTemplate(t *testing.T, r *support.ResourceRegistry, factory *models.Factory, templateID, provider string, configure func(*yaml.Canvas)) *models.Canvas {
	t.Helper()
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, templateID)
	result, err := materializeFactoryTemplate(templateID, provider, factoryTemplateInput{
		appID: canvas.ID.String(), appName: canvas.Name,
		installParams: map[string]string{"appRepository": "acme/app", "defaultBranch": "main"},
		agent:         &factoryTemplateAgent{component: models.SuperPlaneRunnerComponent, credentialSource: "hosted"},
	})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	configure(doc)
	resetTestPublishCanvas(t, r, canvas, doc)
	return canvas
}

func resetTestLiveVersion(t *testing.T, canvas *models.Canvas) *models.CanvasVersion {
	t.Helper()
	db := database.DB(t.Context())
	current, err := models.FindCanvasInTransaction(db, canvas.OrganizationID, canvas.ID)
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, current)
	require.NoError(t, err)
	return version
}

func resetTestPublishCanvas(t *testing.T, r *support.ResourceRegistry, canvas *models.Canvas, doc *yaml.Canvas) {
	t.Helper()
	doc.Metadata.ID, doc.Metadata.Name = canvas.ID.String(), canvas.Name
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	nodes, edges, err := doc.Parse(r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.NoError(t, canvases.PublishGeneratedCanvasNodesWithOwner(ctx, database.DB(t.Context()), canvas, nil, "Install test template", nodes, edges,
		changesets.CanvasPublisherOptions{Registry: r.Registry, OrgID: r.Organization.ID, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}))
	reloaded, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	*canvas = *reloaded
}

func resetTestFactory(t *testing.T) (*support.ResourceRegistry, *models.Factory) {
	t.Helper()
	r := support.Setup(t)
	factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	return r, factory
}

func resetTestOrganization(t *testing.T, r *support.ResourceRegistry) error {
	t.Helper()
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	_, err := ResetOrganizationBuiltInAutomations(ctx, r.Organization.ID, IntakeDependencies{
		Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000",
	})
	return err
}

func TestResetBuiltInVerificationSettings(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		for _, templateID := range []string{"risk-score", "pr-closure", "visual-evidence"} {
			template, ok := lookupFactoryAppTemplate(templateID, provider)
			if !ok {
				continue
			}
			t.Run(provider+"/"+templateID, func(t *testing.T) {
				r, factory := resetTestFactory(t)
				canvas := resetTestTemplate(t, r, factory, templateID, provider, func(doc *yaml.Canvas) {
					trigger := findYAMLNode(t, doc, template.entrypointNodeID)
					trigger.Configuration["onlyFactoryPullRequests"] = false
					trigger.Configuration["ignoreDrafts"] = false
					action := "opened"
					if provider == models.ProviderBitbucket {
						action = "created"
					}
					trigger.Configuration["actions"] = []any{action}
					for i := range doc.Spec.Nodes {
						node := &doc.Spec.Nodes[i]
						if !isFactoryAgentHarness(node.Component) {
							continue
						}
						node.Configuration["model"] = ""
						node.Configuration["includeVisualEvidence"] = false
						node.Configuration["executionTimeoutSeconds"] = 2400
						if templateID == "risk-score" {
							risk := maps.Clone(node.Configuration["steps"].([]any)[0].(map[string]any))
							risk["name"] = "Risk approval"
							risk["prompt"] = strings.ReplaceAll(risk["prompt"].(string), defaultRiskScoreRules, "Escalation = 5 (critical).") + "\nRetired system instruction"
							custom := map[string]any{"name": "Release policy", "type": "prompt", "prompt": "Merge check: custom-1.\nCheck the release policy."}
							node.Configuration["steps"] = []any{risk, custom}
						}
					}
				})
				require.NoError(t, resetTestOrganization(t, r))
				version := resetTestLiveVersion(t, canvas)
				trigger := findIntakeNode(version.Nodes, template.entrypointNodeID)
				require.NotNil(t, trigger)
				assert.Equal(t, false, trigger.Configuration["onlyFactoryPullRequests"])
				assert.Equal(t, false, trigger.Configuration["ignoreDrafts"])
				action := "opened"
				if provider == models.ProviderBitbucket {
					action = "created"
				}
				assert.Equal(t, []any{action}, trigger.Configuration["actions"])
				for _, node := range version.Nodes {
					if !isFactoryAgentHarness(node.ComponentName()) {
						continue
					}
					value, present := node.Configuration["model"]
					assert.True(t, present)
					assert.Equal(t, "", value)
					assert.Equal(t, false, node.Configuration["includeVisualEvidence"])
					assert.EqualValues(t, 2400, node.Configuration["executionTimeoutSeconds"])
					if templateID == "risk-score" {
						steps := node.Configuration["steps"].([]any)
						require.Len(t, steps, 2)
						assert.Equal(t, "Risk approval", steps[0].(map[string]any)["name"])
						prompt := steps[0].(map[string]any)["prompt"].(string)
						assert.Contains(t, prompt, "Escalation = 5 (critical).")
						assert.NotContains(t, prompt, "Retired system instruction")
						assert.Equal(t, "Merge check: custom-1.\nCheck the release policy.", steps[1].(map[string]any)["prompt"])
					}
				}
			})
		}
	}
}

func TestResetScheduledIntakeKeepsScheduleAndUserPrompt(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			r, factory := resetTestFactory(t)
			repository := "acme/app"
			require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{AppRepository: &repository, VCSProvider: &provider}))
			canvas := resetTestTemplate(t, r, factory, scheduledIntakeTemplateID, provider, func(doc *yaml.Canvas) {
				schedule := findYAMLNode(t, doc, "schedule")
				schedule.Configuration = map[string]any{"type": "cron", "cronExpression": "15 6 * * 1", "timezone": "5.5"}
				agent := findYAMLNode(t, doc, "scheduled-agent")
				agent.Configuration["executionTimeoutSeconds"] = 600
				implementationStep(t, agent, "Propose a task")["prompt"] = "Find accessibility defects." + scheduledIntakeOutputInstructions
			})
			require.NoError(t, resetTestOrganization(t, r))
			version := resetTestLiveVersion(t, canvas)
			schedule := findIntakeNode(version.Nodes, "schedule")
			assert.Equal(t, "cron", schedule.Configuration["type"])
			assert.Equal(t, "15 6 * * 1", schedule.Configuration["cronExpression"])
			assert.Equal(t, "5.5", schedule.Configuration["timezone"])
			agent := findIntakeNode(version.Nodes, "scheduled-agent")
			assert.EqualValues(t, 600, agent.Configuration["executionTimeoutSeconds"])
			steps := agent.Configuration["steps"].([]any)
			assert.Equal(t, "Find accessibility defects."+scheduledIntakeOutputInstructions, steps[1].(map[string]any)["prompt"])
		})
	}
}

func TestResetBuiltInSettingsFailureRollsBackOrganization(t *testing.T) {
	r, factory := resetTestFactory(t)
	for range 2 {
		resetTestTemplate(t, r, factory, "line-implementation", models.ProviderGitHub, func(doc *yaml.Canvas) {
			findYAMLNode(t, doc, "create-pr").Configuration["draft"] = true
		})
	}
	canvases, err := factory.ListCanvases(database.DB(t.Context()))
	require.NoError(t, err)
	require.Len(t, canvases, 2)
	invalid := &canvases[len(canvases)-1]
	version := resetTestLiveVersion(t, invalid)
	findIntakeNode(version.Nodes, implementationAgentNodeID).Configuration["executionTimeoutSeconds"] = -1
	require.NoError(t, database.DB(t.Context()).Model(version).Update("nodes", datatypes.NewJSONSlice(version.Nodes)).Error)
	err = resetTestOrganization(t, r)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "timeout")
	for _, canvas := range canvases {
		assert.Equal(t, *canvas.LiveVersionID, resetTestLiveVersion(t, &canvas).ID)
	}
}

func TestResetIntakePreservesGeneralSettings(t *testing.T) {
	cases := []struct {
		source          string
		restrictAuthors bool
	}{
		{source: models.FactoryIntakeSourceGitHubIssues},
		{source: models.FactoryIntakeSourceGitHubIssues, restrictAuthors: true},
		{source: models.FactoryIntakeSourceJiraIssues},
		{source: models.FactoryIntakeSourceSentryExceptions},
		{source: models.FactoryIntakeSourceDependabotAlerts},
		{source: models.FactoryIntakeSourceProductiveTasks},
		{source: models.FactoryIntakeSourceDatadog},
		{source: models.FactoryIntakeSourceLinearIssues},
		{source: models.FactoryIntakeSourcePagerDutyIncidents},
	}
	for _, tc := range cases {
		source := tc.source
		name := source
		if tc.restrictAuthors {
			name += "/authors-with-access"
		}
		t.Run(name, func(t *testing.T) {
			r, factory := resetTestFactory(t)
			settings := intakeSettingsOrDefault(source, intakeSettings{})
			switch source {
			case models.FactoryIntakeSourceGitHubIssues:
				settings.Labels = []string{"urgent"}
				settings.LabelFilterMode = intakeLabelFilterExclude
				settings.Assignment = intakeAssignmentUnassigned
				settings.AuthorsWithAccess = tc.restrictAuthors
				settings.NewIssues, settings.ReopenedIssues = true, false
				settings.SuperplaneLabelAdded = false
			case models.FactoryIntakeSourceJiraIssues:
				settings.Labels = []string{"urgent"}
				settings.NewIssues, settings.ReopenedIssues = false, true
				settings.JiraMoveOnComplete = false
				settings.JiraCompletionColumn = "QA"
			case models.FactoryIntakeSourceSentryExceptions:
				settings.SentryNewIssues, settings.SentryRegressedIssues, settings.SentryAssignedIssues = false, false, true
				settings.SentryLevels = []string{"fatal", "warning"}
				settings.SentryProjectIDs = []string{"payments", "checkout"}
			case models.FactoryIntakeSourceDependabotAlerts:
				settings.DependabotSeverities = []string{"high"}
			case models.FactoryIntakeSourceProductiveTasks:
				settings.ExcludeKeyTasks = false
				settings.TaskListIDs = []string{"42"}
			case models.FactoryIntakeSourceDatadog:
				settings.DatadogTriggeredAlerts, settings.DatadogRetriggeredAlerts = false, true
				settings.DatadogEnvironments = []string{"production"}
			case models.FactoryIntakeSourceLinearIssues:
				settings.LinearProjectIDs = []string{"project-one"}
				settings.LinearLabels = []string{}
			}
			binding := &intakeBinding{Configuration: map[string]any{"repository": "acme/app", "project": "APP"}}
			if source == models.FactoryIntakeSourceGitHubIssues {
				id := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
				binding.Integration = &yaml.IntegrationRef{ID: id}
			}
			doc, err := buildIntakeCanvas(intakeCanvasRequest{
				OrganizationID: r.Organization.ID, FactoryID: factory.ID, Source: source,
				Settings: settings, Binding: binding,
			})
			require.NoError(t, err)
			if source == models.FactoryIntakeSourceGitHubIssues {
				// Older installations can use different generated node ids.
				for i := range doc.Spec.Nodes {
					if doc.Spec.Nodes[i].ID == intakeTriggerNodeID {
						doc.Spec.Nodes[i].ID = "legacy-trigger"
					}
				}
				for i := range doc.Spec.Edges {
					if doc.Spec.Edges[i].SourceID == intakeTriggerNodeID {
						doc.Spec.Edges[i].SourceID = "legacy-trigger"
					}
				}
			}
			canvas := support.CreateFactoryCanvas(t, r, factory.ID, source)
			resetTestPublishCanvas(t, r, canvas, doc)
			intake, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, source)
			require.NoError(t, err)
			ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
			updated, err := UpdateFactoryIntake(ctx, IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}, r.Organization.ID.String(), &pb.UpdateFactoryIntakeRequest{
				FactoryId: factory.ID.String(), IntakeId: intake.ID.String(), Settings: serializeIntakeSettings(source, settings),
			})
			require.NoError(t, err)
			expected := updated.Intake.Settings
			if source == models.FactoryIntakeSourceGitHubIssues || source == models.FactoryIntakeSourceJiraIssues {
				// The editor action historically retains filters, but restores
				// default completion behavior and does not keep an author gate.
				response, err := MaterializeFactoryAutomationDefaults(ctx, r.Organization.ID.String(), &pb.MaterializeFactoryAutomationDefaultsRequest{
					FactoryId: factory.ID.String(), AutomationId: canvas.ID.String(),
				})
				require.NoError(t, err)
				defaults, err := yaml.CanvasFromYAML([]byte(response.CanvasYaml))
				require.NoError(t, err)
				defaultSpec := models.LiveCanvasSpec{Nodes: defaults.Nodes(), Edges: defaults.Edges()}
				restored := serializeIntakeSettings(source, intakeSettingsFromGraph(source, resolveIntakeGraph(source, defaultSpec), defaultSpec))
				assert.Equal(t, expected.GetLabels(), restored.GetLabels())
				assert.Equal(t, expected.GetLabelFilterMode(), restored.GetLabelFilterMode())
				assert.Equal(t, expected.GetAssignment(), restored.GetAssignment())
				assert.Equal(t, expected.GetReopenedIssues(), restored.GetReopenedIssues())
				if source == models.FactoryIntakeSourceJiraIssues {
					assert.True(t, restored.GetJiraMoveOnComplete())
					assert.Empty(t, restored.GetJiraCompletionColumn())
				} else {
					assert.False(t, restored.GetAuthorsWithAccess())
				}
			}
			require.NoError(t, resetTestOrganization(t, r))
			version := resetTestLiveVersion(t, canvas)
			spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
			actual := intakeSettingsFromGraph(source, resolveIntakeGraph(source, spec), spec)
			assert.Equal(t, expected, serializeIntakeSettings(source, actual))
			if tc.restrictAuthors {
				require.NotNil(t, findIntakeNode(version.Nodes, intakeAuthorPermissionNodeID))
				assert.True(t, hasCanvasPath(spec.Edges, intakeFilterNodeID, intakeAuthorPermissionNodeID))
				assert.True(t, hasCanvasPath(spec.Edges, intakeAuthorFilterNodeID, intakeCreateNodeID))
				assert.False(t, slices.ContainsFunc(spec.Edges, func(edge models.Edge) bool {
					return edge.SourceID == intakeFilterNodeID && edge.TargetID == intakeCreateNodeID
				}))
			}
		})
	}
}

func TestResetPRFeedbackPreservesEachRunnerAndGeneralSettings(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		for _, source := range []string{models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion, models.FactoryPRFeedbackHandlerSourcePullRequestChecks} {
			t.Run(provider+"/"+source, func(t *testing.T) {
				r, factory := resetTestFactory(t)
				require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{VCSProvider: &provider}))
				request := prFeedbackBuildRequest{
					Repository: "acme/app", Mention: "@review", IgnoreBots: false, AllowedBots: []string{},
					CheckNames: []string{"required-ci"}, MaximumAttempts: 8, VCSProvider: provider,
					Agent: &intakeAgent{Component: models.SuperPlaneRunnerComponent},
				}
				doc := buildDiscussionPRFeedbackCanvas(request)
				if source == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
					doc = buildChecksPRFeedbackCanvas(request)
				}
				canvas := support.CreateFactoryCanvas(t, r, factory.ID, source)
				expected := map[string]int{}
				evidence := map[string]bool{}
				for i := range doc.Spec.Nodes {
					node := &doc.Spec.Nodes[i]
					if isFactoryAgentHarness(node.Component) {
						node.Configuration["executionTimeoutSeconds"] = 1200 + i
						node.Configuration["includeVisualEvidence"] = i%2 == 0
						node.Configuration["model"] = ""
						expected[node.ID] = 1200 + i
						evidence[node.ID] = i%2 == 0
						if source == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
							original := node.ID
							node.ID = "legacy-repair"
							for j := range doc.Spec.Edges {
								if doc.Spec.Edges[j].SourceID == original {
									doc.Spec.Edges[j].SourceID = node.ID
								}
								if doc.Spec.Edges[j].TargetID == original {
									doc.Spec.Edges[j].TargetID = node.ID
								}
							}
						}
					}
				}
				resetTestPublishCanvas(t, r, canvas, doc)
				handler, err := factory.CreatePRFeedbackHandler(database.DB(t.Context()), canvas.ID, models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, source)
				require.NoError(t, err)
				require.NoError(t, handler.SetMaximumAttempts(database.DB(t.Context()), 8))
				require.NoError(t, resetTestOrganization(t, r))
				version := resetTestLiveVersion(t, canvas)
				spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
				actual := prFeedbackSettingsFromGraph(resolvePRFeedbackGraph(spec), spec)
				assert.Equal(t, "acme/app", actual.Repository)
				if source == models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion {
					assert.Equal(t, "@review", actual.Mention)
					assert.False(t, actual.IgnoreBots)
					assert.Empty(t, actual.AllowedBots)
				} else {
					assert.Equal(t, []string{"required-ci"}, actual.CheckNames)
				}
				reloaded, err := models.FindPRFeedbackHandlerByCanvasID(database.DB(t.Context()), canvas.ID)
				require.NoError(t, err)
				assert.Equal(t, 8, *reloaded.MaximumAttempts)
				for id, timeout := range expected {
					node := findIntakeNode(version.Nodes, id)
					require.NotNil(t, node)
					assert.EqualValues(t, timeout, node.Configuration["executionTimeoutSeconds"])
					assert.Equal(t, evidence[id], node.Configuration["includeVisualEvidence"])
				}
			})
		}
	}
}

func TestResetPreservesRunnerProviderAndCredentials(t *testing.T) {
	cases := []struct{ component, app, model, llmProvider string }{
		{component: "runnerClaudeCode", app: "claude", model: "claude-sonnet-4-6"},
		{component: "runnerCodex", app: "openai", model: "gpt-5"},
		{component: "runnerOpenRouter", app: "openrouter", model: "anthropic/claude-sonnet-4-6"},
		{component: "runnerOpenRouter", app: models.CustomLLMAppName, model: "local-model", llmProvider: models.UsageProviderCustom},
	}
	for _, tc := range cases {
		t.Run(tc.app, func(t *testing.T) {
			r, factory := resetTestFactory(t)
			integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, tc.app, "selected-agent", nil)
			require.NoError(t, err)
			credentials := map[string]any{"source": "integration", "integration": map[string]any{"name": integration.InstallationName}}
			canvas := resetTestTemplate(t, r, factory, "line-implementation", models.ProviderGitHub, func(doc *yaml.Canvas) {
				agent := findYAMLNode(t, doc, implementationAgentNodeID)
				agent.Component = tc.component
				agent.Configuration["credentials"] = credentials
				agent.Configuration["model"] = tc.model
				if tc.llmProvider != "" {
					agent.Configuration["llmProvider"] = tc.llmProvider
				}
			})
			enableInstanceSuperPlaneDefault(t)
			require.NoError(t, resetTestOrganization(t, r))
			node := findIntakeNode(resetTestLiveVersion(t, canvas).Nodes, implementationAgentNodeID)
			require.NotNil(t, node)
			assert.Equal(t, tc.component, node.ComponentName())
			assert.Equal(t, tc.model, node.Configuration["model"])
			assert.Equal(t, credentials, node.Configuration["credentials"])
			if tc.llmProvider != "" {
				assert.Equal(t, tc.llmProvider, node.Configuration["llmProvider"])
			}
			// An older saved option can become invalid under the current runner.
			version := resetTestLiveVersion(t, canvas)
			findIntakeNode(version.Nodes, implementationAgentNodeID).Configuration["thinkingLevel"] = "obsolete"
			require.NoError(t, database.DB(t.Context()).Model(version).Update("nodes", datatypes.NewJSONSlice(version.Nodes)).Error)
			err = resetTestOrganization(t, r)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "thinkingLevel")
			assert.Equal(t, version.ID, resetTestLiveVersion(t, canvas).ID)

		})
	}
}

func TestResetPlannerKeepsSettingsWhenGeneratedIDsChange(t *testing.T) {
	r, factory := resetTestFactory(t)
	doc := buildBacklogCanvas(backlogCanvasRequest{Agent: &intakeAgent{Component: models.SuperPlaneRunnerComponent}})
	agent := findYAMLNode(t, doc, backlogRefinementNodeID)
	agent.ID = "legacy-refiner"
	agent.Configuration["executionTimeoutSeconds"] = 1250
	agent.Configuration["disabledAgentResourceIds"] = []any{"skill-to-disable"}
	for i := range doc.Spec.Edges {
		if doc.Spec.Edges[i].SourceID == backlogRefinementNodeID {
			doc.Spec.Edges[i].SourceID = agent.ID
		}
		if doc.Spec.Edges[i].TargetID == backlogRefinementNodeID {
			doc.Spec.Edges[i].TargetID = agent.ID
		}
	}
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Planner")
	resetTestPublishCanvas(t, r, canvas, doc)
	require.NoError(t, resetTestOrganization(t, r))
	current := findIntakeNode(resetTestLiveVersion(t, canvas).Nodes, backlogRefinementNodeID)
	require.NotNil(t, current)
	assert.EqualValues(t, 1250, current.Configuration["executionTimeoutSeconds"])
	assert.Equal(t, []any{"skill-to-disable"}, current.Configuration["disabledAgentResourceIds"])
}

func TestResetSupportsLegacyHostedRunnerSettings(t *testing.T) {
	r, factory := resetTestFactory(t)
	canvas := resetTestTemplate(t, r, factory, "line-implementation", models.ProviderGitHub, func(*yaml.Canvas) {})
	version := resetTestLiveVersion(t, canvas)
	agent := findIntakeNode(version.Nodes, implementationAgentNodeID)
	agent.Ref.Component.Name = "runnerCodex"
	agent.Configuration["credentials"] = map[string]any{"source": "hosted"}
	agent.Configuration["model"] = "gpt-5"
	agent.Configuration["executionTimeoutSeconds"] = 1500
	require.NoError(t, database.DB(t.Context()).Model(version).Update("nodes", datatypes.NewJSONSlice(version.Nodes)).Error)
	require.NoError(t, resetTestOrganization(t, r))
	current := findIntakeNode(resetTestLiveVersion(t, canvas).Nodes, implementationAgentNodeID)
	assert.Equal(t, models.SuperPlaneRunnerComponent, current.ComponentName())
	assert.Equal(t, models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderOpenAI, "gpt-5"), current.Configuration["model"])
	assert.EqualValues(t, 1500, current.Configuration["executionTimeoutSeconds"])
	assert.NotContains(t, current.Configuration, "credentials")
}

func TestResetUsesTemplateDefaultsForMissingSettings(t *testing.T) {
	r, factory := resetTestFactory(t)
	canvas := resetTestTemplate(t, r, factory, "line-implementation", models.ProviderGitHub, func(doc *yaml.Canvas) {
		agent := findYAMLNode(t, doc, implementationAgentNodeID)
		delete(agent.Configuration, "executionTimeoutSeconds")
		agent.Concurrency = nil
		delete(findYAMLNode(t, doc, "create-pr").Configuration, "draft")
	})
	require.NoError(t, resetTestOrganization(t, r))
	version := resetTestLiveVersion(t, canvas)
	agent := findIntakeNode(version.Nodes, implementationAgentNodeID)
	require.NotNil(t, agent)
	assert.EqualValues(t, 3600, agent.Configuration["executionTimeoutSeconds"])
	require.NotNil(t, agent.Concurrency)
	assert.Equal(t, 5, agent.Concurrency.EffectiveMax())
	assert.Equal(t, false, findIntakeNode(version.Nodes, "create-pr").Configuration["draft"])
}

func TestResetMergeConfidencePreservesDisabledChecks(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			r, factory := resetTestFactory(t)
			canvas := resetTestTemplate(t, r, factory, "risk-score", provider, func(doc *yaml.Canvas) {
				findYAMLNode(t, doc, "assess-risk").Configuration["steps"] = []any{
					map[string]any{
						"name": "Review", "type": "prompt",
						"prompt": "Enabled checks: none.\nDo not call the report_merge_check tool.\nRetired system instruction",
					},
				}
			})
			require.NoError(t, resetTestOrganization(t, r))
			node := findIntakeNode(resetTestLiveVersion(t, canvas).Nodes, "assess-risk")
			require.NotNil(t, node)
			var steps []runner.AgentStep
			decoder, err := runner.NewSpecDecoder(&steps)
			require.NoError(t, err)
			require.NoError(t, decoder.Decode(node.Configuration["steps"]))
			require.NotEmpty(t, steps)
			checks, _ := runner.MergeConfidenceChecksFromSteps(steps)
			assert.Empty(t, checks)
			assert.NotContains(t, liveCanvasJSON(t, canvas), "Retired system instruction")
		})
	}
}
