package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRefreshWorkspaceRepositoryAutomationsListensOnEveryRepository(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, "test-private-key")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")

	r := support.Setup(t)
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

	_, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/web",
	})
	require.NoError(t, err)

	refreshWorkspaceRepositoryAutomations(context.Background(), deps, factory)

	live := liveRepositoryCanvas(t, db, r.Organization.ID, closure)
	trigger := findIntakeNode(live.Nodes, prClosureTriggerNodeID)
	require.NotNil(t, trigger)
	assert.Equal(t, true, trigger.Configuration[githubcommon.WorkspaceRepositoriesField])
	assert.Equal(t, "acme/api", trigger.Configuration["repository"])
	complete := findIntakeNode(live.Nodes, "complete-work-order")
	require.NotNil(t, complete)
	assert.Equal(t, true, complete.Configuration[allPullRequestsDoneField], "the task must stay open until every pull request is done")
	riskTrigger := findIntakeNode(liveRepositoryCanvas(t, db, r.Organization.ID, risk).Nodes, riskScoreTriggerNodeID)
	require.NotNil(t, riskTrigger)
	assert.Equal(t, true, riskTrigger.Configuration[githubcommon.WorkspaceRepositoriesField])

	node, err := models.FindCanvasNode(db, closure.ID, prClosureTriggerNodeID)
	require.NoError(t, err)
	require.NotNil(t, node.WebhookID, "state=%s reason=%v", node.State, node.StateReason)
	webhook, err := models.FindWebhookInTransaction(db, *node.WebhookID)
	require.NoError(t, err)
	webhookConfig, ok := webhook.Configuration.Data().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"acme/api", "acme/web"}, webhookConfig["repositories"])
}
