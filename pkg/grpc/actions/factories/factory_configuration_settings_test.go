package factories

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/datatypes"
)

// The public settings decoder and graph builder own presence semantics; a zero
// confidence value must not restore other defaults or turn issue events back on.
func TestFactoryConfigurationIntakeSettingsRetainExplicitValues(t *testing.T) {
	cases := []struct {
		source       string
		fields       map[string]any
		triggerField string
		expected     []any
	}{
		{source: models.FactoryIntakeSourceGitHubIssues, fields: map[string]any{"confidence_pct": 0, "labels": []any{}, "authors_with_access": false, "new_issues": false, "reopened_issues": true, "superplane_label_added": false}, triggerField: "actions", expected: []any{"reopened"}},
		{source: models.FactoryIntakeSourceJiraIssues, fields: map[string]any{"confidence_pct": 0, "labels": []any{}, "new_issues": false, "reopened_issues": true, "jira_move_on_complete": false, "jira_completion_column": ""}, triggerField: "events", expected: []any{"updated"}},
		{source: models.FactoryIntakeSourceSentryExceptions, fields: map[string]any{"sentry_new_issues": false, "sentry_regressed_issues": true, "sentry_assigned_issues": false, "sentry_levels": []any{}}, triggerField: "actions", expected: []any{"unresolved"}},
		{source: models.FactoryIntakeSourceGitHubIssues, fields: map[string]any{"new_issues": false, "reopened_issues": false, "superplane_label_added": false}, triggerField: "actions", expected: []any{}},
		{source: models.FactoryIntakeSourceSentryExceptions, fields: map[string]any{"sentry_new_issues": false, "sentry_regressed_issues": false, "sentry_assigned_issues": false}, triggerField: "actions", expected: []any{}},
		{source: models.FactoryIntakeSourceDatadog, fields: map[string]any{"datadog_triggered_alerts": false, "datadog_retriggered_alerts": true, "datadog_environments": []any{}}, triggerField: "alertTransitions", expected: []any{"Re-Triggered"}},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			settings, err := parseFactoryConfigurationIntakeSettings(tc.source, tc.fields)
			require.NoError(t, err)
			document, err := buildIntakeCanvas(intakeCanvasRequest{Source: tc.source, Settings: settings, Binding: &intakeBinding{Configuration: map[string]any{"repository": "acme/app", "project": "APP"}}})
			require.NoError(t, err)
			nodes := document.Nodes()
			trigger := findIntakeNode(nodes, intakeTriggerNodeID)
			require.NotNil(t, trigger)
			require.Equal(t, tc.expected, trigger.Configuration[tc.triggerField])
			spec := models.LiveCanvasSpec{Nodes: nodes, Edges: document.Edges()}
			restored := intakeSettingsFromGraph(tc.source, resolveIntakeGraph(tc.source, spec), spec)
			exported := exportFactoryConfigurationIntakeSettings(tc.source, restored)
			for field, value := range tc.fields {
				require.True(t, equalCanvasJSON(value, exported[field]), "%s: wanted %#v, got %#v", field, value, exported[field])
			}
		})
	}
}

func TestFactoryConfigurationIntakeSettingsRejectUnsupportedFieldsAndTypes(t *testing.T) {
	for _, tc := range []struct {
		source string
		field  string
		value  any
		fields map[string]any
	}{
		{field: "sentry_levels", value: []any{"error"}},
		{field: "labels", value: "bug"},
		{field: "authors_with_access", value: "false"},
		{field: "confidence_pct", value: 101},
		{field: "confidence_pct", value: 0.5},
		{field: "confidence_pct", value: math.NaN()},
		{field: "labels", value: nil},
		{field: "assignment", value: "unknown"},
		{source: models.FactoryIntakeSourceJiraIssues, field: "new_issues", value: false, fields: map[string]any{"reopened_issues": false}},
		{source: models.FactoryIntakeSourceDatadog, field: "datadog_triggered_alerts", value: false, fields: map[string]any{"datadog_retriggered_alerts": false}},
	} {
		t.Run(tc.field+"="+testingValueName(tc.value), func(t *testing.T) {
			if tc.source == "" {
				tc.source = models.FactoryIntakeSourceGitHubIssues
			}
			if tc.fields == nil {
				tc.fields = map[string]any{}
			}
			tc.fields[tc.field] = tc.value
			_, err := parseFactoryConfigurationIntakeSettings(tc.source, tc.fields)
			var setting *factoryConfigurationSettingError
			require.ErrorAs(t, err, &setting)
			require.Equal(t, "intake."+tc.field, setting.Field)
		})
	}
}

func testingValueName(value any) string {
	if value == nil {
		return "null"
	}
	switch value.(type) {
	case string:
		return "string"
	case float64:
		return "decimal"
	case int:
		return "integer"
	}
	return "list"
}

func TestFactoryConfigurationPinnedBindingsSurviveResetAndRepositoryRefresh(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	integration := createReadyOnboardingIntegration(t, f.r.Organization.ID, models.ProviderGitHub)
	repository := "acme/workspace"
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integration, AppRepository: &repository}))
	pinned := createReadyOnboardingIntegration(t, f.r.Organization.ID, models.ProviderGitHub)
	risk := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(doc *yaml.Canvas) {
		trigger := findYAMLNode(t, doc, riskScoreTriggerNodeID)
		trigger.Configuration["repository"] = models.WorkspaceRepositoriesExpression
		trigger.Integration = &yaml.IntegrationRef{ID: pinned}
	})
	owner := models.FactoryConfigurationAutomation{OrganizationID: f.r.Organization.ID, FactoryID: f.factory.ID, LineID: f.line.ID, StageKey: "verify", ManifestID: risk.ConfigurationID, CanvasID: risk.ID, Binding: datatypes.JSONMap{"integration_id": pinned, "resource_id": "acme/pinned"}}
	require.NoError(t, owner.Save(db))
	// Saving a moved automation replaces its earlier ownership record.
	laterOwner := owner
	laterOwner.StageKey = "zzz"
	require.NoError(t, laterOwner.Save(db))
	owners, err := models.ListFactoryConfigurationAutomations(db, f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	require.Len(t, owners, 1)
	require.Equal(t, "zzz", owners[0].StageKey)
	repository = "acme/changed"
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &repository}))
	require.NoError(t, resetTestOrganization(t, f.r))
	version := resetTestLiveVersion(t, risk)
	trigger := findIntakeNode(version.Nodes, riskScoreTriggerNodeID)
	require.Equal(t, "acme/pinned", trigger.Configuration["repository"])
	require.NotNil(t, trigger.IntegrationID)
	require.Equal(t, pinned, *trigger.IntegrationID)
	refreshWorkspaceRepositoryAutomations(f.ctx, testIntakeDependencies(f.r), f.factory)
	refreshed := resetTestLiveVersion(t, risk)
	require.Equal(t, version.ID, refreshed.ID, "a pinned repository must not receive a workspace-scope revision")
}

func TestFactoryConfigurationScheduleReportsTheInvalidField(t *testing.T) {
	for _, tc := range []struct {
		field    string
		value    any
		settings map[string]any
	}{
		{field: "hour", value: 24},
		{field: "minute", value: -1},
		{field: "timezone", value: "NaN"},
		{field: "timezone", value: ""},
		{field: "type", value: ""},
		{field: "cron_expression", value: "invalid", settings: map[string]any{"type": "cron"}},
	} {
		t.Run(tc.field+"="+testingValueName(tc.value), func(t *testing.T) {
			if tc.settings == nil {
				tc.settings = map[string]any{}
			}
			tc.settings[tc.field] = tc.value
			err := applyFactoryConfigurationTemplateSettings(scheduledIntakeTemplateID, tc.settings, &factoryTemplateInput{})
			var setting *factoryConfigurationSettingError
			require.ErrorAs(t, err, &setting)
			require.Equal(t, "template."+tc.field, setting.Field)
		})
	}
}

func TestFactoryConfigurationResetKeepsReboundIntakeTrigger(t *testing.T) {
	for _, source := range []string{models.FactoryIntakeSourceDatadog, models.FactoryIntakeSourcePagerDutyIncidents} {
		t.Run(source, func(t *testing.T) {
			f := configurationCatalogFixture(t)
			templateID := intakeTemplateID(source)
			automation := configurationCatalogEntry(t, f, templateID)
			bundle := configurationCatalogBundle(t, f, automation, templateID)
			plan := f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			_, err := f.apply(bundle, plan.BaseStateToken)
			require.NoError(t, err)
			canvas, err := models.FindCanvasByName(database.DB(t.Context()), f.factory.OrganizationID, &f.factory.ID, automation.Name)
			require.NoError(t, err)
			binding := *automation.Binding
			if source == models.FactoryIntakeSourceDatadog {
				binding.IntegrationID = createReadyOnboardingIntegration(t, f.r.Organization.ID, "datadog")
				configureConfigurationCatalogIntegration(t, f, binding.IntegrationID, "datadog")
			} else {
				binding.ResourceID = "second"
			}
			configurationUpdateCatalogAutomation(t, bundle, automation.ID, func(candidate *factoryConfigurationAutomation) { candidate.Binding = &binding })
			plan = f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			_, err = f.apply(bundle, plan.BaseStateToken)
			require.NoError(t, err)
			live := resetTestLiveVersion(t, canvas)
			graph := resolveIntakeGraph(source, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges})
			require.NotEqual(t, intakeTriggerNodeID, graph.TriggerNodeID, "the fixture must exercise a rebound trigger")
			before, err := models.FindCanvasNode(database.DB(t.Context()), canvas.ID, graph.TriggerNodeID)
			require.NoError(t, err)
			restored, err := MaterializeFactoryAutomationDefaults(f.ctx, f.r.Organization.ID.String(), &pb.MaterializeFactoryAutomationDefaultsRequest{FactoryId: f.factory.ID.String(), AutomationId: canvas.ID.String()})
			require.NoError(t, err)
			document, err := yaml.CanvasFromYAML([]byte(restored.CanvasYaml))
			require.NoError(t, err)
			restoredGraph := resolveIntakeGraph(source, models.LiveCanvasSpec{Nodes: document.Nodes(), Edges: document.Edges()})
			require.Equal(t, graph.TriggerNodeID, restoredGraph.TriggerNodeID, "restore defaults must keep the active provider resources")
			require.NoError(t, resetTestOrganization(t, f.r))
			live = resetTestLiveVersion(t, canvas)
			resetGraph := resolveIntakeGraph(source, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges})
			require.Equal(t, graph.TriggerNodeID, resetGraph.TriggerNodeID, "admin reset must keep the active trigger")
			require.True(t, resetGraph.Healthy(live.Edges))
			after, err := models.FindCanvasNode(database.DB(t.Context()), canvas.ID, resetGraph.TriggerNodeID)
			require.NoError(t, err)
			require.Equal(t, before.WebhookID, after.WebhookID)
			require.Equal(t, before.AppInstallationID, after.AppInstallationID)
		})
	}
}

func TestFactoryConfigurationRestoresLegacyPublicationSetting(t *testing.T) {
	f := newConfigurationFixture(t)
	canvas := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(doc *yaml.Canvas) {
		delete(findYAMLNode(t, doc, "assess-risk").Configuration, "publishResults")
		findYAMLNode(t, doc, "publish-merge-confidence").Configuration["enabled"] = false
	})
	bundle := f.export(t)
	_, err := DeleteFactoryAutomation(f.ctx, f.r.Organization.ID.String(), &pb.DeleteFactoryAutomationRequest{
		FactoryId: f.factory.ID.String(), AutomationId: canvas.ID.String(),
	})
	require.NoError(t, err)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	restored, err := models.FindCanvasByName(database.DB(t.Context()), f.r.Organization.ID, &f.factory.ID, canvas.Name)
	require.NoError(t, err)
	require.NoError(t, resetTestOrganization(t, f.r))
	live := resetTestLiveVersion(t, restored)
	require.Equal(t, false, findIntakeNode(live.Nodes, "publish-merge-confidence").Configuration["enabled"])
}
