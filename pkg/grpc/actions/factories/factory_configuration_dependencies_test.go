package factories

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	goyaml "gopkg.in/yaml.v3"
	"gorm.io/datatypes"
)

func TestFactoryConfigurationPreservesExplicitDefaultBindingIntent(t *testing.T) {
	f := configurationCatalogFixture(t)
	config := f.factory.OnboardingConfigValue()
	pinned := factoryConfigurationBinding{IntegrationID: config.VCSIntegrationID, ResourceID: config.AppRepository}
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		for i := range manifest.Stages {
			for j := range manifest.Stages[i].Automations {
				automation := &manifest.Stages[i].Automations[j]
				if automation.Uses == "@superplane/implement" {
					automation.Binding = &pinned
				}
			}
		}
	})
	before := configurationRevisionCount(t, f.canvas)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes, "an explicit pin changes future reset and repository refresh behavior")
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, before, configurationRevisionCount(t, f.canvas), "binding intent alone does not change the installed graph")
	require.Equal(t, &pinned, configurationManifestAutomation(t, f.export(t), "implementation", "implement").Binding)

	confidence := configurationCatalogEntry(t, f, "risk-score")
	confidence.ID, confidence.Name = "confidence", "Pinned confidence"
	db := database.DB(t.Context())
	_, err = f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/context", Access: models.FactoryContextRepoAccessWrite,
	})
	require.NoError(t, err)
	bundle = configurationCatalogBundle(t, f, confidence, "risk-score")
	applyConfigurationBundle(t, f, bundle)
	installed := configurationCanvasByName(t, f, confidence.Name)
	require.ElementsMatch(t, []string{pinned.ResourceID, "acme/context"}, configurationWebhookRepositories(t, installed, riskScoreTriggerNodeID))

	bundle = f.export(t)
	setConfigurationBinding(t, bundle, "verify", "confidence", &pinned)
	applyConfigurationBundle(t, f, bundle)
	require.Equal(t, []string{pinned.ResourceID}, configurationWebhookRepositories(t, installed, riskScoreTriggerNodeID), "pinning a default repository must narrow an existing webhook")

	newRepository := "acme/backlog"
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &newRepository}))
	require.NoError(t, republishWorkspaceRepositoryAutomations(f.ctx, db, testIntakeDependencies(f.r), f.factory))
	trigger := findIntakeNode(resetTestLiveVersion(t, installed).Nodes, riskScoreTriggerNodeID)
	require.NotNil(t, trigger)
	require.Equal(t, pinned.ResourceID, trigger.Configuration["repository"], "repository refresh must keep an explicit pin")
	boundRepositories, err := contexts.NewFactoryContext(db, f.canvas, nil).ListRepositories()
	require.NoError(t, err)
	require.Equal(t, []string{pinned.ResourceID}, boundRepositories, "the phase runner must execute against its pinned repository")

	_, err = ResetOrganizationBuiltInAutomations(f.ctx, f.r.Organization.ID, testIntakeDependencies(f.r))
	require.NoError(t, err)
	for _, automation := range []struct{ stage, id string }{{"implementation", "implement"}, {"verify", "confidence"}} {
		require.Equal(t, &pinned, configurationManifestAutomation(t, f.export(t), automation.stage, automation.id).Binding)
	}
	trigger = findIntakeNode(resetTestLiveVersion(t, installed).Nodes, riskScoreTriggerNodeID)
	require.Equal(t, pinned.ResourceID, trigger.Configuration["repository"], "admin reset must keep an explicit pin")
	require.Equal(t, []string{pinned.ResourceID}, configurationWebhookRepositories(t, installed, riskScoreTriggerNodeID))

	bundle = f.export(t)
	setConfigurationBinding(t, bundle, "verify", "confidence", nil)
	applyConfigurationBundle(t, f, bundle)
	require.ElementsMatch(t, []string{newRepository, "acme/context"}, configurationWebhookRepositories(t, installed, riskScoreTriggerNodeID), "removing a pin must set up the webhook with the new workspace scope")
	plan = f.plan(t, f.export(t))
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationKeepsSharedRunnerBindingsConsistentAcrossLines(t *testing.T) {
	f := configurationCatalogFixture(t)
	db := database.DB(t.Context())
	defaults := f.factory.OnboardingConfigValue()
	pinned := &factoryConfigurationBinding{IntegrationID: defaults.VCSIntegrationID, ResourceID: defaults.AppRepository}
	bundle := f.export(t)
	setConfigurationBinding(t, bundle, "implementation", "implement", pinned)
	applyConfigurationBundle(t, f, bundle)

	line, err := f.factory.CreateLine(db, "Alternate delivery", f.line.Steps)
	require.NoError(t, err)
	stages := line.StagesValue()
	for i := range stages {
		if stages[i].StepIndex != nil {
			stages[i].Name = "Build changes"
		}
	}
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	other := f
	other.line = line
	for _, change := range []struct {
		writer     configurationFixture
		binding    *factoryConfigurationBinding
		repository string
	}{
		{other, &factoryConfigurationBinding{IntegrationID: defaults.VCSIntegrationID, ResourceID: "acme/backlog"}, "acme/backlog"},
		{f, nil, defaults.AppRepository},
	} {
		bundle = change.writer.export(t)
		setConfigurationBinding(t, bundle, "implementation", "implement", change.binding)
		applyConfigurationBundle(t, change.writer, bundle)
		for _, reader := range []configurationFixture{f, other} {
			require.Equal(t, change.binding, configurationManifestAutomation(t, reader.export(t), "implementation", "implement").Binding)
		}
		repositories, err := contexts.NewFactoryContext(db, f.canvas, nil).ListRepositories()
		require.NoError(t, err)
		require.Equal(t, []string{change.repository}, repositories, "one shared canvas must use the same binding regardless of its ownership row order")
	}
}

func TestFactoryConfigurationResolvesDependenciesAcrossDeletionAndOverlappingIDs(t *testing.T) {
	f := configurationCatalogFixture(t)
	review := configurationCatalogEntry(t, f, reviewRequestTemplateID)
	instructions := "Keep __MANIFEST_AUTOMATION__verify/confidence-extra exactly as written."
	review.With = map[string]any{
		"template": map[string]any{"merge_confidence": map[string]any{"stage": "verify", "automation": "confidence-extra"}},
		"agents":   map[string]any{reviewRequestAgentNodeID: map[string]any{"additional_instructions": instructions}},
	}
	bundle := configurationCatalogBundle(t, f, review, reviewRequestTemplateID)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		for i := range manifest.Stages {
			if manifest.Stages[i].Key == "verify" {
				manifest.Stages[i].Automations = append(manifest.Stages[i].Automations, factoryConfigurationAutomation{
					ID: "confidence-extra", Name: "Extra confidence", Uses: "@superplane/merge-confidence",
				})
			}
		}
	})
	applyConfigurationBundle(t, f, bundle)
	source := configurationCanvasByName(t, f, "Extra confidence")
	consumer := configurationCanvasByName(t, f, review.Name)
	assertConfigurationReviewSource(t, consumer, source)
	live := resetTestLiveVersion(t, consumer)
	require.Equal(t, instructions, findIntakeNode(live.Nodes, reviewRequestAgentNodeID).Configuration["additionalInstructions"], "canvas references must not rewrite user instructions")

	// Keep Review request installed while its source is deleted. The same
	// manifest must create a fresh source and rewire the retained consumer.
	db := database.DB(t.Context())
	require.NoError(t, source.SoftDeleteInTransaction(db))
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	reinstalled := configurationCanvasByName(t, f, "Extra confidence")
	require.NotEqual(t, source.ID, reinstalled.ID)
	require.Equal(t, consumer.ID, configurationCanvasByName(t, f, review.Name).ID)
	assertConfigurationReviewSource(t, consumer, reinstalled)
	before := configurationResourceCounts(t)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	result, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, before, configurationResourceCounts(t))
}

func TestFactoryConfigurationRejectsStalePreparationInputs(t *testing.T) {
	for _, input := range []string{"integration", "resources", "workspace defaults"} {
		t.Run(input, func(t *testing.T) {
			f := configurationCatalogFixture(t)
			automation := configurationCatalogEntry(t, f, intakeTemplateID(models.FactoryIntakeSourceGitHubIssues))
			bundle := configurationCatalogBundle(t, f, automation, intakeTemplateID(models.FactoryIntakeSourceGitHubIssues))
			plan := f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			require.NotEmpty(t, plan.BaseStateToken)
			db := database.DB(t.Context())
			if input == "workspace defaults" {
				branch := "next-main"
				require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{DefaultBranch: &branch}))
			} else {
				integrationID := f.factory.OnboardingConfigValue().VCSIntegrationID
				integration, err := models.FindIntegrationInTransaction(db, f.r.Organization.ID, uuid.MustParse(integrationID))
				require.NoError(t, err)
				metadata := integration.Metadata.Data()
				if metadata == nil {
					metadata = map[string]any{}
				}
				if input == "integration" {
					metadata["installationId"] = "changed-installation"
				} else {
					metadata["repositories"] = []any{
						map[string]any{"name": "acme/app", "id": 101},
						map[string]any{"name": "acme/backlog", "id": 102},
						map[string]any{"name": "acme/extra", "id": 103},
					}
				}
				require.NoError(t, db.Model(integration).Update("metadata", datatypes.NewJSONType(metadata)).Error)
			}
			fresh := f.plan(t, bundle)
			require.Empty(t, fresh.Diagnostics, "the changed input remains valid; only the old plan is stale")
			require.NotEqual(t, plan.BaseStateToken, fresh.BaseStateToken)
			before := configurationResourceCounts(t)
			_, err := f.apply(bundle, plan.BaseStateToken)
			require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}

func editConfigurationManifest(t *testing.T, bundle *pb.FactoryConfigurationBundle, edit func(*factoryConfigurationManifest)) {
	t.Helper()
	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	edit(&manifest)
	encoded, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(encoded)
}

func setConfigurationBinding(t *testing.T, bundle *pb.FactoryConfigurationBundle, stageKey, id string, binding *factoryConfigurationBinding) {
	t.Helper()
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		for i := range manifest.Stages {
			if manifest.Stages[i].Key != stageKey {
				continue
			}
			for j := range manifest.Stages[i].Automations {
				if manifest.Stages[i].Automations[j].ID == id {
					manifest.Stages[i].Automations[j].Binding = binding
					return
				}
			}
		}
		t.Fatalf("automation %s/%s is missing", stageKey, id)
	})
}

func configurationManifestAutomation(t *testing.T, bundle *pb.FactoryConfigurationBundle, stageKey, id string) factoryConfigurationAutomation {
	t.Helper()
	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	for _, stage := range manifest.Stages {
		if stage.Key != stageKey {
			continue
		}
		for _, automation := range stage.Automations {
			if automation.ID == id {
				return automation
			}
		}
	}
	t.Fatalf("automation %s/%s is missing", stageKey, id)
	return factoryConfigurationAutomation{}
}

func applyConfigurationBundle(t *testing.T, f configurationFixture, bundle *pb.FactoryConfigurationBundle) {
	t.Helper()
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
}

func configurationCanvasByName(t *testing.T, f configurationFixture, name string) *models.Canvas {
	t.Helper()
	canvas, err := models.FindCanvasByName(database.DB(t.Context()), f.r.Organization.ID, &f.factory.ID, name)
	require.NoError(t, err)
	return canvas
}

func assertConfigurationReviewSource(t *testing.T, consumer, source *models.Canvas) {
	t.Helper()
	trigger := findIntakeNode(resetTestLiveVersion(t, consumer).Nodes, reviewRequestTriggerNodeID)
	require.NotNil(t, trigger)
	require.Equal(t, source.ID.String(), trigger.Configuration["app"])
	var subscriptions []models.CanvasSubscription
	require.NoError(t, database.DB(t.Context()).Where("target_canvas_id = ? AND target_node_id = ?", consumer.ID, reviewRequestTriggerNodeID).Find(&subscriptions).Error)
	require.Len(t, subscriptions, 1)
	require.Equal(t, source.ID, subscriptions[0].SourceCanvasID)
}

func configurationWebhookRepositories(t *testing.T, canvas *models.Canvas, nodeID string) []string {
	t.Helper()
	db := database.DB(t.Context())
	node, err := models.FindCanvasNode(db, canvas.ID, nodeID)
	require.NoError(t, err)
	require.NotNil(t, node.WebhookID)
	webhook, err := models.FindWebhookInTransaction(db, *node.WebhookID)
	require.NoError(t, err)
	encoded, err := json.Marshal(webhook.Configuration.Data())
	require.NoError(t, err)
	var configuration githubcommon.WebhookConfiguration
	require.NoError(t, json.Unmarshal(encoded, &configuration))
	return configuration.WebhookRepositories()
}
