package factories

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	goyaml "gopkg.in/yaml.v3"
	"gorm.io/datatypes"
)

func TestFactoryConfigurationResolvesDependenciesAcrossDeletionAndOverlappingIDs(t *testing.T) {
	f := configurationCatalogFixture(t)
	const stageID = "verify/priority"
	stages := f.line.StagesValue()
	stages[2].ConfigurationID = stageID
	require.NoError(t, f.line.UpdateConfiguration(database.DB(t.Context()), nil, nil, nil, stages))
	review := configurationCatalogEntry(t, f, reviewRequestTemplateID)
	instructions := "Keep __MANIFEST_AUTOMATION__verify/confidence-extra exactly as written."
	review.With = map[string]any{
		"template": map[string]any{"merge_confidence": map[string]any{"stage": stageID, "automation": "confidence-extra"}},
		"agent":    map[string]any{"additional_instructions": instructions},
	}
	// The fixture helper selects the default stage ID; keep selection explicit for this authored ID.
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		for i := range manifest.Stages {
			if manifest.Stages[i].ID == stageID {
				manifest.Stages[i].Automations = append(manifest.Stages[i].Automations, review, factoryConfigurationAutomation{ID: "confidence", Name: "Confidence", Uses: "@superplane/merge-confidence"})
			}
		}
	})
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		for i := range manifest.Stages {
			if manifest.Stages[i].ID == stageID {
				manifest.Stages[i].Automations = append(manifest.Stages[i].Automations, factoryConfigurationAutomation{
					ID: "confidence-extra", Name: "Extra confidence", Uses: "@superplane/merge-confidence",
				})
			}
		}
	})
	invalid := proto.Clone(bundle).(*pb.FactoryConfigurationBundle)
	editConfigurationManifest(t, invalid, func(manifest *factoryConfigurationManifest) {
		manifest.Stages[2].Automations[0].With["template"].(map[string]any)["merge_confidence"] = map[string]any{"stage": "verify", "automation": "priority/confidence-extra"}
	})
	rejected := f.plan(t, invalid)
	require.NotEmpty(t, rejected.Diagnostics, "stage and automation IDs must match independently")
	require.Equal(t, "stages[2].automations[0].with.template.merge_confidence", rejected.Diagnostics[0].Field)

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
	for _, input := range []string{"integration", "resources", "workspace defaults", "workspace repositories", "Datadog site", "PagerDuty authentication"} {
		t.Run(input, func(t *testing.T) {
			f := configurationCatalogFixture(t)
			templateID := intakeTemplateID(models.FactoryIntakeSourceGitHubIssues)
			switch input {
			case "workspace repositories":
				templateID = "risk-score"
			case "Datadog site":
				templateID = intakeTemplateID(models.FactoryIntakeSourceDatadog)
			case "PagerDuty authentication":
				templateID = intakeTemplateID(models.FactoryIntakeSourcePagerDutyIncidents)
			}
			automation := configurationCatalogEntry(t, f, templateID)
			bundle := configurationCatalogBundle(t, f, automation, templateID)
			editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) { m.Repositories = nil })
			plan := f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			require.NotEmpty(t, plan.BaseStateToken)
			db := database.DB(t.Context())
			if input == "Datadog site" || input == "PagerDuty authentication" {
				integration, err := models.FindIntegrationInTransaction(db, f.r.Organization.ID, uuid.MustParse(automation.Binding.IntegrationID))
				require.NoError(t, err)
				configuration := integration.Configuration.Data()
				if input == "Datadog site" {
					configuration["site"] = "datadoghq.eu"
				} else {
					configuration["authType"] = "appOAuth"
				}
				require.NoError(t, db.Model(integration).Update("configuration", datatypes.NewJSONType(configuration)).Error)
			} else if input == "workspace repositories" {
				_, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
					Repository: "acme/context", VCSProvider: models.ProviderGitHub, Access: models.FactoryContextRepoAccessWrite,
				})
				require.NoError(t, err)
			} else if input == "workspace defaults" {
				f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{gitHubBranches: func(string) ([]string, error) { return []string{"main", "next-main"}, nil }})
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

func TestFactoryConfigurationIgnoresCredentialRotationAndSyncTimestamps(t *testing.T) {
	f := configurationCatalogFixture(t)
	templateID := intakeTemplateID(models.FactoryIntakeSourceDatadog)
	automation := configurationCatalogEntry(t, f, templateID)
	bundle := configurationCatalogBundle(t, f, automation, templateID)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.BaseStateToken)
	db := database.DB(t.Context())
	integration, err := models.FindIntegrationInTransaction(db, f.r.Organization.ID, uuid.MustParse(automation.Binding.IntegrationID))
	require.NoError(t, err)
	credential, err := f.r.Encryptor.Encrypt(t.Context(), []byte("rotated-api-key"), []byte(integration.ID.String()))
	require.NoError(t, err)
	configuration := integration.Configuration.Data()
	configuration["apiKey"] = base64.StdEncoding.EncodeToString(credential)
	metadata := integration.Metadata.Data()
	metadata["refreshedAt"] = time.Now().Format(time.RFC3339Nano)
	require.NoError(t, db.Model(integration).Updates(map[string]any{
		"configuration": datatypes.NewJSONType(configuration),
		"metadata":      datatypes.NewJSONType(metadata),
	}).Error)
	fresh := f.plan(t, bundle)
	require.Empty(t, fresh.Diagnostics)
	require.Equal(t, plan.BaseStateToken, fresh.BaseStateToken)
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

func configurationManifestAutomation(t *testing.T, bundle *pb.FactoryConfigurationBundle, stageKey, id string) factoryConfigurationAutomation {
	t.Helper()
	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	for _, stage := range manifest.Stages {
		if stage.ID != stageKey {
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
