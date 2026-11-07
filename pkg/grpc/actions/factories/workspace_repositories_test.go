package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/gorm"
)

func TestRefreshWorkspaceRepositoryAutomationsListensOnEveryRepository(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, "test-private-key")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")

	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	db := database.DB(t.Context())
	factory := githubCatalogFactory(t, r, db)
	integrationID := factory.OnboardingConfigValue().VCSIntegrationID
	deps := IntakeDependencies{
		Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000",
	}
	closure := support.CreateFactoryCanvas(t, r, factory.ID, "Close")
	metadata := models.FactoryAppTemplateMetadataFor("pr-closure", 1, models.ProviderGitHub)
	metadata["repository"] = map[string]any{"id": 201, "name": "acme/api"}
	publishRepositoryCanvas(t, r, deps, closure, []models.Node{
		{
			ID: prClosureTriggerNodeID, Name: "On pull request closed", Type: models.NodeTypeTrigger,
			Ref:           models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPullRequest"}},
			IntegrationID: &integrationID,
			Configuration: map[string]any{"repository": "acme/api", "actions": []any{"closed"}},
			Metadata:      metadata,
		},
		{
			ID: "complete-work-order", Name: "Complete task", Type: models.NodeTypeComponent,
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: "updateWorkOrderStatus"}},
			Configuration: map[string]any{
				"orderId": "{{ root().data.workOrder.id }}", "status": "closed", "result": "completed",
			},
		},
	})
	risk := support.CreateFactoryCanvas(t, r, factory.ID, "Merge confidence")
	riskMetadata := models.FactoryAppTemplateMetadataFor("risk-score", 1, models.ProviderGitHub)
	riskMetadata["repository"] = map[string]any{"id": 201, "name": "acme/api"}
	publishRepositoryCanvas(t, r, deps, risk, []models.Node{{
		ID: riskScoreTriggerNodeID, Name: "On pull request", Type: models.NodeTypeTrigger,
		Ref:           models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPullRequest"}},
		IntegrationID: &integrationID,
		Configuration: map[string]any{"repository": "acme/api", "actions": []any{"opened"}},
		Metadata:      riskMetadata,
	}})
	visualEvidence := support.CreateFactoryCanvas(t, r, factory.ID, "Visual Evidence")
	visualEvidenceMetadata := models.FactoryAppTemplateMetadataFor("visual-evidence", 1, models.ProviderGitHub)
	visualEvidenceMetadata["repository"] = map[string]any{"id": 201, "name": "acme/api"}
	publishRepositoryCanvas(t, r, deps, visualEvidence, []models.Node{{
		ID: visualEvidenceTriggerNodeID, Name: "On pull request", Type: models.NodeTypeTrigger,
		Ref:           models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPullRequest"}},
		IntegrationID: &integrationID,
		Configuration: map[string]any{"repository": "acme/api", "actions": []any{"synchronize"}},
		Metadata:      visualEvidenceMetadata,
	}})

	_, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/web",
		Access:     models.FactoryContextRepoAccessWrite,
	})
	require.NoError(t, err)

	refreshWorkspaceRepositoryAutomations(context.Background(), deps, factory)

	live := liveRepositoryCanvas(t, db, r.Organization.ID, closure)
	trigger := findIntakeNode(live.Nodes, prClosureTriggerNodeID)
	require.NotNil(t, trigger)
	assert.Equal(t, models.WorkspaceRepositoriesExpression, trigger.Configuration["repository"])
	complete := findIntakeNode(live.Nodes, "complete-work-order")
	require.NotNil(t, complete)
	assert.Equal(t, true, complete.Configuration[allPullRequestsDoneField], "the task must stay open until every pull request is done")
	riskTrigger := findIntakeNode(liveRepositoryCanvas(t, db, r.Organization.ID, risk).Nodes, riskScoreTriggerNodeID)
	require.NotNil(t, riskTrigger)
	assert.Equal(t, models.WorkspaceRepositoriesExpression, riskTrigger.Configuration["repository"])
	visualEvidenceTrigger := findIntakeNode(liveRepositoryCanvas(t, db, r.Organization.ID, visualEvidence).Nodes, visualEvidenceTriggerNodeID)
	require.NotNil(t, visualEvidenceTrigger)
	assert.Equal(t, models.WorkspaceRepositoriesExpression, visualEvidenceTrigger.Configuration["repository"])

	assert.Equal(t, []any{"acme/api", "acme/web"}, nodeWebhookRepositories(t, db, closure.ID, prClosureTriggerNodeID))
}

func TestRefreshWorkspaceRepositoryAutomationsCoversNewRepositoriesInCustomAutomations(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, "test-private-key")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")

	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	db := database.DB(t.Context())
	factory := githubCatalogFactory(t, r, db)
	integrationID := factory.OnboardingConfigValue().VCSIntegrationID
	deps := IntakeDependencies{
		Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000",
	}
	custom := support.CreateFactoryCanvas(t, r, factory.ID, "Custom push")
	publishRepositoryCanvas(t, r, deps, custom, []models.Node{{
		ID: "on-push", Name: "On push", Type: models.NodeTypeTrigger,
		Ref:           models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPush"}},
		IntegrationID: &integrationID,
		Configuration: map[string]any{"repository": models.WorkspaceRepositoriesExpression},
	}})

	_, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/web",
		Access:     models.FactoryContextRepoAccessWrite,
	})
	require.NoError(t, err)

	refreshWorkspaceRepositoryAutomations(context.Background(), deps, factory)

	assert.Equal(t, []any{"acme/api", "acme/web"}, nodeWebhookRepositories(t, db, custom.ID, "on-push"))
}

func TestRefreshWorkspaceRepositoryAutomationsListensOnEveryBitbucketRepository(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	db := database.DB(t.Context())
	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "bitbucket", support.RandomName("bitbucket"), nil)
	require.NoError(t, err)
	factory := bitbucketFactory(t, r)
	integrationID := integration.ID.String()
	repository := "acme/api"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID,
		AppRepository:    &repository,
	}))
	var setupRepositories [][]string
	r.Registry.Integrations["bitbucket"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Triggers: []core.Trigger{impl.NewDummyTrigger(impl.DummyTriggerOptions{
			Name: "bitbucket.onPullRequest",
			SetupFunc: func(ctx core.TriggerContext) error {
				repositories, err := ctx.Factory.ListRepositories()
				setupRepositories = append(setupRepositories, repositories)
				return err
			},
		})},
	})
	deps := IntakeDependencies{
		Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000",
	}
	closure := support.CreateFactoryCanvas(t, r, factory.ID, "Close")
	publishRepositoryCanvas(t, r, deps, closure, []models.Node{
		{
			ID: prClosureTriggerNodeID, Name: "On pull request closed", Type: models.NodeTypeTrigger,
			Ref:           models.NodeRef{Trigger: &models.TriggerRef{Name: "bitbucket.onPullRequest"}},
			IntegrationID: &integrationID,
			Configuration: map[string]any{"repository": "acme/api", "actions": []any{"merged"}},
			Metadata:      models.FactoryAppTemplateMetadataFor("pr-closure", 1, models.ProviderBitbucket),
		},
		{
			ID: "complete-work-order", Name: "Complete task", Type: models.NodeTypeComponent,
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: "updateWorkOrderStatus"}},
			Configuration: map[string]any{
				"orderId": "{{ root().data.workOrder.id }}", "status": "closed", "result": "completed",
			},
		},
	})

	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository:  "acme/web",
		Access:      models.FactoryContextRepoAccessWrite,
		VCSProvider: models.ProviderBitbucket,
	})
	require.NoError(t, err)
	setupRepositories = nil

	refreshWorkspaceRepositoryAutomations(context.Background(), deps, factory)

	live := liveRepositoryCanvas(t, db, r.Organization.ID, closure)
	trigger := findIntakeNode(live.Nodes, prClosureTriggerNodeID)
	require.NotNil(t, trigger)
	assert.Equal(t, models.WorkspaceRepositoriesExpression, trigger.Configuration["repository"])
	complete := findIntakeNode(live.Nodes, "complete-work-order")
	require.NotNil(t, complete)
	assert.Equal(t, true, complete.Configuration[allPullRequestsDoneField])
	require.NotEmpty(t, setupRepositories, "the trigger Setup must run again")
	assert.Equal(t, []string{"acme/api", "acme/web"}, setupRepositories[len(setupRepositories)-1])
}

func nodeWebhookRepositories(t *testing.T, db *gorm.DB, canvasID uuid.UUID, nodeID string) any {
	t.Helper()
	node, err := models.FindCanvasNode(db, canvasID, nodeID)
	require.NoError(t, err)
	require.NotNil(t, node.WebhookID, "state=%s reason=%v", node.State, node.StateReason)
	webhook, err := models.FindWebhookInTransaction(db, *node.WebhookID)
	require.NoError(t, err)
	webhookConfig, ok := webhook.Configuration.Data().(map[string]any)
	require.True(t, ok)
	return webhookConfig["repositories"]
}
