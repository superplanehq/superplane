package factories

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	goyaml "gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func configurationCatalogFixture(t *testing.T) configurationFixture {
	t.Helper()
	f := newConfigurationFixture(t)
	require.NoError(t, models.EnableExperimentalFeature(f.r.Organization.ID, features.FeatureFactoryScheduledIntake))
	require.NoError(t, models.EnableExperimentalFeature(f.r.Organization.ID, features.FeatureFactoryReviewRequest))
	return f
}

func configurationCatalogEntry(t *testing.T, f configurationFixture, templateID string) factoryConfigurationAutomation {
	t.Helper()
	alias, ok := factoryConfigurationBuiltInAlias(templateID)
	require.True(t, ok)
	automation := factoryConfigurationAutomation{ID: "new-automation", Name: "New automation", Uses: alias}
	source := strings.TrimPrefix(templateID, "intake:")
	if source != templateID && source != models.FactoryIntakeSourceGitHubIssues && source != models.FactoryIntakeSourceDependabotAlerts {
		provider := configurationIntakeProvider(source)
		integrationID := createReadyOnboardingIntegration(t, f.r.Organization.ID, provider)
		configureConfigurationCatalogIntegration(t, f, integrationID, provider)
		automation.Binding = &factoryConfigurationBinding{IntegrationID: integrationID, ResourceID: "project"}
		if source == models.FactoryIntakeSourceLinearIssues {
			automation.Binding.ResourceID, automation.Binding.TeamID = "", "team"
			automation.With = map[string]any{"intake": map[string]any{"linear_project_ids": []any{"project", "second"}, "linear_include_issues_without_project": false}}
		}
		if source == models.FactoryIntakeSourceSentryExceptions {
			automation.Binding.ResourceID = ""
			automation.With = map[string]any{"intake": map[string]any{"sentry_project_ids": []any{"project", "second"}, "sentry_new_issues": false, "sentry_regressed_issues": true}}
		}
	}
	if templateID == prFeedbackChecksTemplateID {
		automation.With = map[string]any{"pr_feedback": map[string]any{"check_names": []any{"CI"}}}
	}
	if templateID == scheduledIntakeTemplateID {
		automation.With = map[string]any{"template": map[string]any{"prompt": ""}}
	}
	return automation
}

func configurationCatalogBundle(t *testing.T, f configurationFixture, automation factoryConfigurationAutomation, templateID string) *pb.FactoryConfigurationBundle {
	t.Helper()
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	stage := "verify"
	switch {
	case templateID == lineImplementationTemplateID:
		stage = "implementation"
	case templateID == "pr-closure":
		stage = "done"
	case templateID == models.FactoryAppTemplateBacklogID || templateID == scheduledIntakeTemplateID || templateID == "issue-intake" || strings.HasPrefix(templateID, "intake:"):
		stage = "backlog"
	}
	for i := range manifest.Stages {
		if manifest.Stages[i].Key != stage {
			continue
		}
		manifest.Stages[i].Automations = append(manifest.Stages[i].Automations, automation)
		if templateID == reviewRequestTemplateID {
			// A dependency can follow its consumer in the manifest.
			manifest.Stages[i].Automations = append(manifest.Stages[i].Automations, factoryConfigurationAutomation{ID: "confidence", Name: "Confidence", Uses: "@superplane/merge-confidence"})
		}
	}
	content, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	return bundle
}

func TestFactoryConfigurationInstallsSupportedCatalogEntries(t *testing.T) {
	for _, entry := range builtInAutomations {
		// Workspace webhook bindings are not supported by YAML restoration yet.
		if entry.ID == intakeTemplateID(models.FactoryIntakeSourceWebhook) {
			continue
		}
		for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
			if provider == models.ProviderBitbucket && !configurationCatalogSupportsBitbucket(entry.ID) {
				continue
			}
			t.Run(provider+"/"+entry.ID, func(t *testing.T) {
				f := configurationCatalogFixture(t)
				if provider == models.ProviderBitbucket {
					integrationID := createReadyOnboardingIntegration(t, f.r.Organization.ID, provider)
					configureConfigurationCatalogIntegration(t, f, integrationID, provider)
					require.NoError(t, f.factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{VCSProvider: &provider, VCSIntegrationID: &integrationID}))
				}
				automation := configurationCatalogEntry(t, f, entry.ID)
				bundle := configurationCatalogBundle(t, f, automation, entry.ID)
				before := configurationResourceCounts(t)
				requests := configurationCatalogHTTPRequestCount(f)
				plan := f.plan(t, bundle)
				require.Empty(t, plan.Diagnostics)
				require.NotEmpty(t, plan.BaseStateToken)
				require.True(t, slices.ContainsFunc(plan.Changes, func(change *pb.FactoryConfigurationChange) bool {
					return strings.Contains(change.RequestedValue, automation.Uses)
				}), "plan must include the missing built-in")
				require.Equal(t, before, configurationResourceCounts(t), "plan must not create provisioning or import records")
				require.Equal(t, requests, configurationCatalogHTTPRequestCount(f), "plan must not call provider APIs")
				result, err := f.apply(bundle, plan.BaseStateToken)
				require.NoError(t, err)
				require.True(t, result.Changed)
				db := database.DB(t.Context())
				canvas, err := models.FindCanvasByName(db, f.factory.OrganizationID, &f.factory.ID, automation.Name)
				require.NoError(t, err)
				live := resetTestLiveVersion(t, canvas)
				require.NotEmpty(t, live.Nodes)
				for _, node := range live.Nodes {
					require.Nil(t, node.ErrorMessage, node.Name)
				}
				configurationAssertCatalogSupportingRecords(t, f, canvas, entry.ID)
				resources := configurationResourceCounts(t)
				revision := configurationRevisionCount(t, canvas)
				plan = f.plan(t, bundle)
				require.Empty(t, plan.Diagnostics)
				require.Empty(t, plan.Changes)
				writes := configurationCountApplyWrites(t, func() {
					result, err = f.apply(bundle, plan.BaseStateToken)
					require.NoError(t, err)
					require.False(t, result.Changed)
				})
				require.Zero(t, writes, "repeated apply must not update ownership, subscriptions, or provisioning records")
				require.Equal(t, resources, configurationResourceCounts(t))
				require.Equal(t, revision, configurationRevisionCount(t, canvas))
				if entry.ID == intakeTemplateID(models.FactoryIntakeSourceLinearIssues) {
					var manifest factoryConfigurationManifest
					require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
					for i := range manifest.Stages {
						for j := range manifest.Stages[i].Automations {
							candidate := &manifest.Stages[i].Automations[j]
							if candidate.ID == automation.ID {
								candidate.With["intake"].(map[string]any)["linear_project_ids"] = []any{}
							}
						}
					}
					content, err := goyaml.Marshal(manifest)
					require.NoError(t, err)
					bundle.Files[0].Content = string(content)
					plan = f.plan(t, bundle)
					require.Empty(t, plan.Diagnostics)
					_, err = f.apply(bundle, plan.BaseStateToken)
					require.NoError(t, err)
					response, err := ListFactoryIntakes(f.ctx, f.factory.OrganizationID.String(), &pb.ListFactoryIntakesRequest{FactoryId: f.factory.ID.String()})
					require.NoError(t, err)
					require.Empty(t, response.Intakes[0].GetSettings().GetLinearProjectIds(), "an explicit empty filter must clear the installed project list")
					plan = f.plan(t, bundle)
					require.Empty(t, plan.Diagnostics)
					require.Empty(t, plan.Changes)
				}
				if entry.ID == intakeTemplateID(models.FactoryIntakeSourceDatadog) || entry.ID == intakeTemplateID(models.FactoryIntakeSourcePagerDutyIncidents) {
					configurationAssertCatalogBindingChange(t, f, canvas, automation, bundle, entry.ID)
				}
			})
		}
	}
}

func TestFactoryConfigurationPreservesInstalledWebhookIntake(t *testing.T) {
	f := newConfigurationFixture(t)
	require.NoError(t, models.EnableExperimentalFeature(f.r.Organization.ID, features.FeatureFactoryWebhookIntake))
	deps := testIntakeDependencies(f.r)
	webhook, err := CreateFactoryWebhook(f.ctx, deps, f.r.Organization.ID.String(), &pb.CreateFactoryWebhookRequest{
		FactoryId: f.factory.ID.String(), Name: "Tracker", Authentication: pb.FactoryWebhook_AUTHENTICATION_BEARER,
	})
	require.NoError(t, err)
	created, err := CreateFactoryIntake(f.ctx, deps, f.r.Organization.ID.String(), &pb.CreateFactoryIntakeRequest{
		FactoryId: f.factory.ID.String(), Source: pb.FactoryIntake_SOURCE_WEBHOOK, ResourceId: webhook.Webhook.Id,
	})
	require.NoError(t, err)
	_, err = UpdateFactoryIntake(f.ctx, deps, f.r.Organization.ID.String(), &pb.UpdateFactoryIntakeRequest{
		FactoryId: f.factory.ID.String(), IntakeId: created.Intake.Id,
		Settings: &pb.FactoryIntake_Settings{WebhookTitlePath: proto.String("issue.summary")},
	})
	require.NoError(t, err)
	canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), f.r.Organization.ID, uuid.MustParse(created.Intake.CanvasId))
	require.NoError(t, err)
	before := resetTestLiveVersion(t, canvas)
	bundle := f.export(t)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)

	bundle.Files[0].Content = strings.ReplaceAll(bundle.Files[0].Content, "Original instructions", "Updated instructions")
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	result, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, before.ID, resetTestLiveVersion(t, canvas).ID, "an unrelated update must preserve the intake revision")

	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	for i := range manifest.Stages {
		for j := range manifest.Stages[i].Automations {
			automation := &manifest.Stages[i].Automations[j]
			if automation.Uses == "@superplane/intake/webhook" {
				automation.With["intake"] = map[string]any{"confidence_pct": 23}
			}
		}
	}
	content, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	live := resetTestLiveVersion(t, canvas)
	spec := models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges}
	graph := resolveIntakeGraph(models.FactoryIntakeSourceWebhook, spec)
	require.Equal(t, webhook.Webhook.Id, graph.TriggerResourceID(spec))
	require.Equal(t, 23, intakeSettingsFromGraph(models.FactoryIntakeSourceWebhook, graph, spec).ConfidencePct)
	require.Equal(t, "issue.summary", intakeSettingsFromGraph(models.FactoryIntakeSourceWebhook, graph, spec).WebhookTitlePath)
	plan = f.plan(t, f.export(t))
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	bundle = f.export(t)
	bundle.Files[0].Content = strings.ReplaceAll(bundle.Files[0].Content, webhook.Webhook.Id, uuid.NewString())
	plan = f.plan(t, bundle)
	require.Len(t, plan.Diagnostics, 1)
	require.Contains(t, plan.Diagnostics[0].Field, ".binding")
}

func configurationAssertCatalogBindingChange(t *testing.T, f configurationFixture, canvas *models.Canvas, automation factoryConfigurationAutomation, bundle *pb.FactoryConfigurationBundle, templateID string) {
	t.Helper()
	db := database.DB(t.Context())
	source := strings.TrimPrefix(templateID, "intake:")
	intake, err := models.FindFactoryIntakeByCanvasID(db, canvas.ID)
	require.NoError(t, err)
	live := resetTestLiveVersion(t, canvas)
	graph := resolveIntakeGraph(source, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges})
	previous, err := models.FindCanvasNode(db, canvas.ID, graph.TriggerNodeID)
	require.NoError(t, err)
	binding := *automation.Binding
	if source == models.FactoryIntakeSourceDatadog {
		// A completed reconcile stores these values for post-commit retirement.
		metadata := previous.Metadata.Data()
		metadata["monitorId"], metadata["monitorService"] = "123", "project"
		require.NoError(t, db.Model(previous).Update("metadata", datatypes.NewJSONType(metadata)).Error)
		binding.IntegrationID = createReadyOnboardingIntegration(t, f.r.Organization.ID, "datadog")
		configureConfigurationCatalogIntegration(t, f, binding.IntegrationID, "datadog")
	} else {
		binding.ResourceID = "second"
	}
	configurationUpdateCatalogAutomation(t, bundle, automation.ID, func(candidate *factoryConfigurationAutomation) {
		candidate.Binding = &binding
	})
	resources := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes)
	require.Equal(t, resources, configurationResourceCounts(t), "planning a rebind must retain the active resources")
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	updatedIntake, err := models.FindFactoryIntakeByCanvasID(db, canvas.ID)
	require.NoError(t, err)
	require.Equal(t, intake.ID, updatedIntake.ID, "rebinding must retain the intake and canvas")
	live = resetTestLiveVersion(t, canvas)
	graph = resolveIntakeGraph(source, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges})
	require.True(t, graph.Healthy(live.Edges))
	require.NotEqual(t, previous.NodeID, graph.TriggerNodeID, "rebinding must retire provider runtime state")
	retired, err := models.FindUnscopedCanvasNode(db, canvas.ID, previous.NodeID)
	require.NoError(t, err)
	require.True(t, retired.DeletedAt.Valid)
	require.Equal(t, previous.AppInstallationID, retired.AppInstallationID)
	current, err := models.FindCanvasNode(db, canvas.ID, graph.TriggerNodeID)
	require.NoError(t, err)
	require.Equal(t, uuid.MustParse(binding.IntegrationID), *current.AppInstallationID)
	if source == models.FactoryIntakeSourceDatadog {
		require.Equal(t, "123", retired.Metadata.Data()["monitorId"], "cleanup needs the old monitor with its old integration")
		require.Empty(t, current.Metadata.Data()["monitorId"])
		var count int64
		require.NoError(t, db.Model(&models.IntegrationSubscription{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, retired.NodeID).Count(&count).Error)
		require.Zero(t, count, "the retired connection must stop delivering alerts")
	} else {
		require.Equal(t, "second", current.Configuration.Data()["service"])
		require.Equal(t, "second", current.Metadata.Data()["service"].(map[string]any)["id"])
		require.NotEqual(t, previous.WebhookID, current.WebhookID)
		var hook models.Webhook
		require.NoError(t, db.Unscoped().First(&hook, "id = ?", previous.WebhookID).Error)
		require.True(t, hook.DeletedAt.Valid, "an unused previous-service webhook must retire")
	}
	configurationAssertCatalogSupportingRecords(t, f, canvas, templateID)
	// A later settings rebuild must keep the active trigger and its resources.
	configurationUpdateCatalogAutomation(t, bundle, automation.ID, func(candidate *factoryConfigurationAutomation) {
		candidate.With = map[string]any{"intake": map[string]any{"confidence_pct": 17}}
	})
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	live = resetTestLiveVersion(t, canvas)
	graph = resolveIntakeGraph(source, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges})
	require.Equal(t, current.NodeID, graph.TriggerNodeID, "settings changes must not restore a retired trigger")
	configurationAssertCatalogSupportingRecords(t, f, canvas, templateID)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	revision := configurationRevisionCount(t, canvas)
	writes := configurationCountApplyWrites(t, func() {
		result, err := f.apply(bundle, plan.BaseStateToken)
		require.NoError(t, err)
		require.False(t, result.Changed)
	})
	require.Zero(t, writes)
	require.Equal(t, revision, configurationRevisionCount(t, canvas))
}

func configurationUpdateCatalogAutomation(t *testing.T, bundle *pb.FactoryConfigurationBundle, id string, update func(*factoryConfigurationAutomation)) {
	t.Helper()
	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	for i := range manifest.Stages {
		for j := range manifest.Stages[i].Automations {
			if manifest.Stages[i].Automations[j].ID == id {
				update(&manifest.Stages[i].Automations[j])
			}
		}
	}
	content, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
}

func configurationCatalogSupportsBitbucket(templateID string) bool {
	if strings.HasPrefix(templateID, "intake:") {
		source := strings.TrimPrefix(templateID, "intake:")
		return source != models.FactoryIntakeSourceGitHubIssues && source != models.FactoryIntakeSourceDependabotAlerts
	}
	return slices.Contains([]string{models.FactoryAppTemplateBacklogID, scheduledIntakeTemplateID, lineImplementationTemplateID, "pr-closure", "risk-score", prFeedbackDiscussionTemplateID, prFeedbackChecksTemplateID}, templateID)
}

func configurationAssertCatalogSupportingRecords(t *testing.T, f configurationFixture, canvas *models.Canvas, templateID string) {
	t.Helper()
	db := database.DB(t.Context())
	if source := strings.TrimPrefix(templateID, "intake:"); source != templateID {
		intake, err := models.FindFactoryIntakeByCanvasID(db, canvas.ID)
		require.NoError(t, err)
		require.Equal(t, source, intake.Source)
		require.Equal(t, models.FactoryIntakeInitialImportStatusSkipped, intake.InitialImportStatus)
		response, err := ListFactoryIntakes(f.ctx, f.factory.OrganizationID.String(), &pb.ListFactoryIntakesRequest{FactoryId: f.factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, response.Intakes, 1)
		if source == models.FactoryIntakeSourceLinearIssues {
			require.Equal(t, []string{"project", "second"}, response.Intakes[0].GetSettings().GetLinearProjectIds())
		}
		if source == models.FactoryIntakeSourceSentryExceptions {
			require.Equal(t, []string{"project", "second"}, response.Intakes[0].GetSettings().GetSentryProjectIds())
		}
	}
	if templateID == prFeedbackDiscussionTemplateID || templateID == prFeedbackChecksTemplateID {
		handler, err := models.FindPRFeedbackHandlerByCanvasID(db, canvas.ID)
		require.NoError(t, err)
		require.NotNil(t, handler)
		expected := models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion
		if templateID == prFeedbackChecksTemplateID {
			expected = models.FactoryPRFeedbackHandlerSourcePullRequestChecks
		}
		require.Equal(t, expected, handler.Source)
	}
	if templateID == reviewRequestTemplateID {
		var subscription models.CanvasSubscription
		require.NoError(t, db.Where("target_canvas_id = ?", canvas.ID).First(&subscription).Error)
		confidence, err := models.FindCanvasByName(db, f.factory.OrganizationID, &f.factory.ID, "Confidence")
		require.NoError(t, err)
		require.Equal(t, confidence.ID, subscription.SourceCanvasID)
	}
	nodes, err := models.FindCanvasNodesInTransaction(db, canvas.ID)
	require.NoError(t, err)
	for _, node := range nodes {
		require.NotEqual(t, models.CanvasNodeStateError, node.State, node.NodeID)
		if node.Type != models.NodeTypeTrigger || node.AppInstallationID == nil {
			continue
		}
		if strings.HasPrefix(node.ComponentName(), "sentry.") || strings.HasPrefix(node.ComponentName(), "datadog.") {
			var subscription models.IntegrationSubscription
			require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", canvas.ID, node.NodeID).First(&subscription).Error)
			require.Equal(t, *node.AppInstallationID, subscription.InstallationID)
			if strings.HasPrefix(node.ComponentName(), "datadog.") {
				request, err := models.FindPendingRequestForNode(db, canvas.ID, node.NodeID)
				require.NoError(t, err)
				require.Equal(t, "reconcileIntakeMonitor", request.Spec.Data().InvokeAction.ActionName)
			}
			continue
		}
		require.NotNil(t, node.WebhookID, node.NodeID)
		webhook, err := models.FindWebhookInTransaction(db, *node.WebhookID)
		require.NoError(t, err)
		require.Equal(t, models.WebhookStatePending, webhook.State)
	}
}

func configurationCountApplyWrites(t *testing.T, apply func()) int {
	t.Helper()
	db := database.DB(t.Context())
	writes := 0
	name := "catalog_repeat_" + uuid.NewString()
	count := func(*gorm.DB) { writes++ }
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(name, count))
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register(name, count))
	require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register(name, count))
	defer func() {
		_ = db.Callback().Create().Remove(name)
		_ = db.Callback().Update().Remove(name)
		_ = db.Callback().Delete().Remove(name)
	}()
	apply()
	return writes
}

func TestFactoryConfigurationDiagnosesProviderSelectionsWithoutWrites(t *testing.T) {
	for _, test := range []struct {
		name       string
		mutate     func(*testing.T, configurationFixture, *factoryConfigurationAutomation)
		field      string
		templateID string
	}{
		{name: "Sentry settings select unavailable projects", templateID: intakeTemplateID(models.FactoryIntakeSourceSentryExceptions), field: "with.intake.sentry_project_ids", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.With = map[string]any{"intake": map[string]any{"sentry_project_ids": []any{"missing"}}}
		}},
		{name: "Sentry binding and settings disagree", templateID: intakeTemplateID(models.FactoryIntakeSourceSentryExceptions), field: "with.intake.sentry_project_ids", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.Binding.ResourceID = "project"
		}},
		{name: "Linear settings select unavailable projects", templateID: intakeTemplateID(models.FactoryIntakeSourceLinearIssues), field: "with.intake.linear_project_ids", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.With = map[string]any{"intake": map[string]any{"linear_project_ids": []any{"missing"}}}
		}},
		{name: "Linear empty projects without team", templateID: intakeTemplateID(models.FactoryIntakeSourceLinearIssues), field: "with.intake.linear_project_ids", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.Binding.TeamID = ""
			automation.With = map[string]any{"intake": map[string]any{"linear_project_ids": []any{}}}
		}},
		{name: "unsupported webhook intake", field: "uses", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.Uses = "@superplane/intake/webhook"
			automation.Binding = nil
		}},
		{name: "missing provider", field: "binding.integration_id", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.Binding = nil
		}},
		{name: "wrong provider", field: "binding.integration_id", mutate: func(t *testing.T, f configurationFixture, automation *factoryConfigurationAutomation) {
			automation.Binding.IntegrationID = createReadyOnboardingIntegration(t, f.r.Organization.ID, "sentry")
		}},
		{name: "wrong organization", field: "binding.integration_id", mutate: func(t *testing.T, f configurationFixture, automation *factoryConfigurationAutomation) {
			other := support.CreateOrganization(t, f.r, f.r.User)
			automation.Binding.IntegrationID = createReadyOnboardingIntegration(t, other.ID, "jira")
		}},
		{name: "missing resource", field: "binding.resource_id", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.Binding.ResourceID = ""
		}},
		{name: "unavailable resource", field: "binding.resource_id", mutate: func(t *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			require.NoError(t, database.DB(t.Context()).Model(&models.Integration{}).Where("id = ?", automation.Binding.IntegrationID).Update("metadata", datatypes.NewJSONType(map[string]any{"projects": []any{map[string]any{"key": "OTHER"}}})).Error)
		}},
		{name: "disabled integration capability", field: "binding.integration_id", mutate: func(t *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			require.NoError(t, database.DB(t.Context()).Model(&models.Integration{}).Where("id = ?", automation.Binding.IntegrationID).Update("capabilities", datatypes.NewJSONSlice([]models.CapabilityState{{Name: "jira.onIssue", State: core.IntegrationCapabilityStateDisabled}})).Error)
		}},
		{name: "pending integration", field: "binding.integration_id", mutate: func(t *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			require.NoError(t, database.DB(t.Context()).Model(&models.Integration{}).Where("id = ?", automation.Binding.IntegrationID).Update("state", models.IntegrationStatePending).Error)
		}},
		{name: "unsupported setting", field: "with.intake.sentry_levels", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.With = map[string]any{"intake": map[string]any{"sentry_levels": []any{"error"}}}
		}},
		{name: "unsupported section", field: "with.pr_feedback", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.With = map[string]any{"pr_feedback": map[string]any{}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := configurationCatalogFixture(t)
			templateID := test.templateID
			if templateID == "" {
				templateID = intakeTemplateID(models.FactoryIntakeSourceJiraIssues)
			}
			automation := configurationCatalogEntry(t, f, templateID)
			test.mutate(t, f, &automation)
			bundle := configurationCatalogBundle(t, f, automation, templateID)
			before := configurationResourceCounts(t)
			plan := f.plan(t, bundle)
			require.Len(t, plan.Diagnostics, 1)
			require.Equal(t, "stages[0].automations[0]."+test.field, plan.Diagnostics[0].Field)
			require.Empty(t, plan.BaseStateToken)
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}

func TestFactoryConfigurationRejectsProviderAndStageRestrictions(t *testing.T) {
	for _, test := range []struct {
		name, templateID   string
		disableGate        bool
		wrongStage         bool
		bitbucket          bool
		unsupportedSetting string
	}{
		{name: "Bitbucket PR checks publication", templateID: "risk-score", bitbucket: true, unsupportedSetting: "publish_pr_check"},
		{name: "Bitbucket comment publication", templateID: "risk-score", bitbucket: true, unsupportedSetting: "results_comment"},
		{name: "scheduled gate", templateID: scheduledIntakeTemplateID, disableGate: true},
		{name: "review gate", templateID: reviewRequestTemplateID, disableGate: true},
		{name: "intake stage", templateID: intakeTemplateID(models.FactoryIntakeSourceGitHubIssues), wrongStage: true},
		{name: "github intake on bitbucket", templateID: intakeTemplateID(models.FactoryIntakeSourceGitHubIssues), bitbucket: true},
		{name: "review on bitbucket", templateID: reviewRequestTemplateID, bitbucket: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := configurationCatalogFixture(t)
			if test.disableGate {
				feature := features.FeatureFactoryScheduledIntake
				if test.templateID == reviewRequestTemplateID {
					feature = features.FeatureFactoryReviewRequest
				}
				require.NoError(t, models.DisableExperimentalFeature(f.r.Organization.ID, feature))
			}
			if test.bitbucket {
				provider := models.ProviderBitbucket
				integrationID := createReadyOnboardingIntegration(t, f.r.Organization.ID, provider)
				configureConfigurationCatalogIntegration(t, f, integrationID, provider)
				require.NoError(t, f.factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{VCSProvider: &provider, VCSIntegrationID: &integrationID}))
			}
			automation := configurationCatalogEntry(t, f, test.templateID)
			if test.unsupportedSetting != "" {
				automation.With = map[string]any{"template": map[string]any{test.unsupportedSetting: false}}
			}
			bundle := configurationCatalogBundle(t, f, automation, test.templateID)
			if test.wrongStage {
				var manifest factoryConfigurationManifest
				require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
				manifest.Stages[2].Automations = manifest.Stages[0].Automations
				manifest.Stages[0].Automations = []factoryConfigurationAutomation{}
				content, err := goyaml.Marshal(manifest)
				require.NoError(t, err)
				bundle.Files[0].Content = string(content)
			}
			before := configurationResourceCounts(t)
			plan := f.plan(t, bundle)
			require.Len(t, plan.Diagnostics, 1)
			if test.unsupportedSetting != "" {
				require.Contains(t, plan.Diagnostics[0].Field, ".with.template."+test.unsupportedSetting)
			} else {
				require.Contains(t, plan.Diagnostics[0].Field, ".uses")
			}
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}

// Only the provider HTTP boundary is replaced. The production trigger still
// creates its real subscriptions, webhook records, and provisioning requests.
type configurationCatalogTrigger struct {
	core.Trigger
	http *configurationCatalogHTTP
}

func (trigger configurationCatalogTrigger) Setup(ctx core.TriggerContext) error {
	ctx.HTTP = trigger.http
	return trigger.Trigger.Setup(ctx)
}

type configurationCatalogHTTP struct {
	requests int
}

func (client *configurationCatalogHTTP) Do(request *http.Request) (*http.Response, error) {
	client.requests++
	if request.Method != http.MethodGet {
		return nil, fmt.Errorf("unexpected provider write during configuration: %s %s", request.Method, request.URL)
	}
	var body string
	switch {
	case request.URL.Host == "api.github.com" && strings.HasPrefix(request.URL.Path, "/repos/acme/"):
		repository := strings.TrimPrefix(request.URL.Path, "/repos/")
		body = fmt.Sprintf(`{"id":101,"name":%q,"full_name":%q,"html_url":%q}`, strings.TrimPrefix(repository, "acme/"), repository, "https://github.com/"+repository)
	case request.URL.Host == "api.bitbucket.org" && request.URL.Path == "/2.0/repositories/acme":
		body = `{"values":[{"uuid":"{app}","slug":"app","name":"app","full_name":"acme/app"},{"uuid":"{backlog}","slug":"backlog","name":"backlog","full_name":"acme/backlog"}]}`
	case request.URL.Host == "api.atlassian.com" && strings.HasSuffix(request.URL.Path, "/rest/api/3/project"):
		body = `[{"id":"1","key":"project","name":"Project"}]`
	case request.URL.Host == "api.pagerduty.com" && (request.URL.Path == "/services/project" || request.URL.Path == "/services/second"):
		body = fmt.Sprintf(`{"service":{"id":%q,"name":"API"}}`, strings.TrimPrefix(request.URL.Path, "/services/"))
	default:
		return nil, fmt.Errorf("unexpected provider read during configuration: %s", request.URL)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
}

func configureConfigurationCatalogIntegration(t *testing.T, f configurationFixture, integrationID, provider string) {
	t.Helper()
	db := database.DB(t.Context())
	integration, err := models.FindIntegrationInTransaction(db, f.factory.OrganizationID, uuid.MustParse(integrationID))
	require.NoError(t, err)
	configuration := map[string]any{}
	metadata := map[string]any{}
	secret := func(name string) {
		require.NoError(t, contexts.NewIntegrationSecretStorage(db, f.r.Encryptor, integration).Create(core.IntegrationSecretDefinition{Name: name, Value: "test-token"}))
	}
	sensitive := func(name string) {
		value, err := f.r.Encryptor.Encrypt(t.Context(), []byte("test-token"), []byte(integration.ID.String()))
		require.NoError(t, err)
		configuration[name] = base64.StdEncoding.EncodeToString(value)
	}
	switch provider {
	case models.ProviderGitHub:
		setup := datatypes.NewJSONType(models.SetupState{})
		integration.SetupState = &setup
		integration.Properties = datatypes.JSONSlice[core.IntegrationPropertyDefinition]{
			{Name: githubcommon.PropertyAuthMethod, Value: githubcommon.AuthMethodPAT},
			{Name: githubcommon.PropertyOwner, Value: "acme"},
			{Name: githubcommon.PropertyOwnerType, Value: githubcommon.OwnerTypeOrganization},
		}
		implementation, err := f.r.Registry.GetIntegration(provider)
		require.NoError(t, err)
		for _, action := range implementation.Actions() {
			integration.Capabilities = append(integration.Capabilities, models.CapabilityState{Name: action.Name(), State: core.IntegrationCapabilityStateEnabled})
		}
		for _, trigger := range implementation.Triggers() {
			integration.Capabilities = append(integration.Capabilities, models.CapabilityState{Name: trigger.Name(), State: core.IntegrationCapabilityStateEnabled})
		}
		metadata["repositories"] = []githubcommon.Repository{
			{ID: 101, Name: "app", URL: "https://github.com/acme/app"},
			{ID: 102, Name: "backlog", URL: "https://github.com/acme/backlog"},
		}
		secret(githubcommon.SecretPAT)
	case models.ProviderBitbucket:
		configuration["authType"] = bitbucket.AuthTypeWorkspaceAccessToken
		configuration["workspace"] = "acme"
		sensitive("token")
		metadata["authType"] = bitbucket.AuthTypeWorkspaceAccessToken
		metadata["workspace"] = map[string]any{"slug": "acme"}
	case "jira":
		metadata["cloudId"] = "cloud"
		metadata["projects"] = []any{map[string]any{"id": "1", "key": "project", "name": "Project"}}
		secret(jira.SecretOAuthAccessToken)
	case "pagerduty":
		configuration["authType"] = "apiToken"
		sensitive("apiToken")
		metadata["services"] = []any{map[string]any{"id": "project", "name": "API"}, map[string]any{"id": "second", "name": "Worker"}}
	case "sentry":
		metadata["organization"] = map[string]any{"id": "1", "slug": "acme", "name": "Acme"}
		metadata["projects"] = []any{map[string]any{"id": "1", "slug": "project"}, map[string]any{"id": "2", "slug": "second"}}
		sensitive("userToken")
	case "linear":
		metadata["teams"] = []any{map[string]any{"id": "team", "name": "Team"}}
		metadata["projects"] = []any{map[string]any{"id": "project"}, map[string]any{"id": "second"}}
		secret("accessToken")
	case "datadog":
		configuration["site"] = "datadoghq.com"
		sensitive("apiKey")
		sensitive("applicationKey")
	}
	integration.Configuration = datatypes.NewJSONType(configuration)
	integration.Metadata = datatypes.NewJSONType(metadata)
	require.NoError(t, db.Save(integration).Error)
	client := &configurationCatalogHTTP{}
	original, err := f.r.Registry.GetIntegration(provider)
	require.NoError(t, err)
	f.r.Registry.Integrations[provider] = configurationCatalogIntegration{Integration: original, http: client}
	t.Cleanup(func() { f.r.Registry.Integrations[provider] = original })
}

// Registry resolves namespaced components through their integration. Delegate
// all behavior and replace only HTTP on their real setup calls.
type configurationCatalogIntegration struct {
	core.Integration
	http *configurationCatalogHTTP
}

func (integration configurationCatalogIntegration) Triggers() []core.Trigger {
	triggers := integration.Integration.Triggers()
	result := make([]core.Trigger, len(triggers))
	for i, trigger := range triggers {
		result[i] = configurationCatalogTrigger{Trigger: trigger, http: integration.http}
	}
	return result
}

func (integration configurationCatalogIntegration) Actions() []core.Action {
	actions := integration.Integration.Actions()
	result := make([]core.Action, len(actions))
	for i, action := range actions {
		result[i] = configurationCatalogAction{Action: action, http: integration.http}
	}
	return result
}

type configurationCatalogAction struct {
	core.Action
	http *configurationCatalogHTTP
}

func (action configurationCatalogAction) Setup(ctx core.SetupContext) error {
	ctx.HTTP = action.http
	return action.Action.Setup(ctx)
}

func (action configurationCatalogAction) ValidateNodeConfiguration(configuration map[string]any) error {
	if validator, ok := action.Action.(core.NodeConfigurationValidator); ok {
		return validator.ValidateNodeConfiguration(configuration)
	}
	return nil
}

func configurationCatalogHTTPRequestCount(f configurationFixture) int {
	requests := 0
	for _, integration := range f.r.Registry.Integrations {
		if adapter, ok := integration.(configurationCatalogIntegration); ok {
			requests += adapter.http.requests
		}
	}
	return requests
}
