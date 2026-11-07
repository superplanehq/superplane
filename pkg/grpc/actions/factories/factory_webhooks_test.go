package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestListFactoryWebhooksScopesAndDescribesRegistrations(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("other"), "", "")
	require.NoError(t, err)
	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), map[string]any{"host": "acme.ghe.com"})
	require.NoError(t, err)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)
	integrationID, provider, repository := integration.ID.String(), models.ProviderGitHub, "acme/app"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID, VCSProvider: &provider, AppRepository: &repository}))
	require.NoError(t, other.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID, VCSProvider: &provider, AppRepository: &repository}))

	first := support.CreateFactoryCanvas(t, r, factory.ID, "Issues")
	second := support.CreateFactoryCanvas(t, r, factory.ID, "Review")
	otherCanvas := support.CreateFactoryCanvas(t, r, other.ID, "Other")
	issues := failedFactoryWebhook(t, integration.ID, map[string]any{"repository": repository, "eventType": "issues"}, `The "issues" event cannot have more than 25 hooks`)
	pr := failedFactoryWebhook(t, integration.ID, map[string]any{"repository": repository, "eventTypes": []string{"pull_request"}}, "provider refused")
	multiRepository := failedFactoryWebhook(t, integration.ID, map[string]any{"repositories": []string{repository, "acme/web"}, "eventType": "issues"}, "error creating webhook in acme/web: provider refused")
	shared := failedFactoryWebhook(t, integration.ID, factoryMergeabilityWebhookConfiguration(repository), `The "pull_request" event cannot have more than 30 hooks`)
	otherShared := failedFactoryWebhook(t, integration.ID, factoryMergeabilityWebhookConfiguration("acme/other"), "provider refused")
	otherIntegration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), map[string]any{})
	require.NoError(t, err)
	otherIntegrationShared := failedFactoryWebhook(t, otherIntegration.ID, factoryMergeabilityWebhookConfiguration(repository), "provider refused")
	unrelated := failedFactoryWebhook(t, integration.ID, map[string]any{"repository": "acme/other", "eventType": "issues"}, "provider refused")
	deleted := failedFactoryWebhook(t, integration.ID, map[string]any{"repository": repository, "eventType": "issues"}, "provider refused")
	otherOnly := failedFactoryWebhook(t, integration.ID, map[string]any{"repository": repository, "eventType": "issues"}, "provider refused")
	attachFactoryWebhookNode(t, first.ID, issues.ID)
	attachFactoryWebhookNode(t, first.ID, pr.ID)
	attachFactoryWebhookNode(t, first.ID, multiRepository.ID)
	attachFactoryWebhookNode(t, second.ID, pr.ID)
	attachFactoryWebhookNode(t, first.ID, deleted.ID)
	attachFactoryWebhookNode(t, otherCanvas.ID, otherOnly.ID)
	require.NoError(t, db.Delete(&deleted).Error)

	response, err := ListFactoryWebhooks(t.Context(), r.Organization.ID.String(), &pb.ListFactoryWebhooksRequest{FactoryId: factory.ID.String()})
	require.NoError(t, err)
	require.Len(t, response.GetWebhooks(), 4)
	statuses := map[string]*pb.FactoryWebhookStatus{}
	for _, status := range response.GetWebhooks() {
		statuses[status.GetId()] = status
	}
	assert.NotContains(t, statuses, unrelated.ID.String())
	assert.NotContains(t, statuses, otherShared.ID.String())
	assert.NotContains(t, statuses, otherIntegrationShared.ID.String())
	assert.NotContains(t, statuses, deleted.ID.String())
	assert.NotContains(t, statuses, otherOnly.ID.String())
	assert.Equal(t, repository, statuses[issues.ID.String()].GetRepository())
	assert.Equal(t, "acme.ghe.com", statuses[issues.ID.String()].GetHost())
	assert.Equal(t, "GitHub reached the webhook limit for issues. Remove unused webhooks in the repository settings, then try again.", statuses[issues.ID.String()].GetMessage())
	assert.Equal(t, []string{"pull_request"}, statuses[pr.ID.String()].GetEventTypes())
	assert.Equal(t, "acme/web", statuses[multiRepository.ID.String()].GetRepository())
	assert.Equal(t, "GitHub could not register the webhook for pull requests. Check the repository settings, then try again.", statuses[pr.ID.String()].GetMessage())
	assert.Len(t, statuses[pr.ID.String()].GetAffectedAutomations(), 2)
	assert.Len(t, statuses[shared.ID.String()].GetAffectedAutomations(), 2)
	sharedAutomations := statuses[shared.ID.String()].GetAffectedAutomations()
	assert.ElementsMatch(t, []string{first.Name, second.Name}, []string{
		sharedAutomations[0].GetName(), sharedAutomations[1].GetName(),
	})
	assert.Equal(t, "GitHub reached the webhook limit for pull requests. Remove unused webhooks in the repository settings, then try again.", statuses[shared.ID.String()].GetMessage())
}

func TestRetryFactoryWebhookRevalidatesMembershipAndReusesRow(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("other"), "", "")
	require.NoError(t, err)
	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), map[string]any{})
	require.NoError(t, err)
	repository, integrationID, provider := "acme/app", integration.ID.String(), models.ProviderGitHub
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID, VCSProvider: &provider, AppRepository: &repository}))
	owned := failedFactoryWebhook(t, integration.ID, factoryMergeabilityWebhookConfiguration(repository), "provider refused")
	issues := failedFactoryWebhook(t, integration.ID, map[string]any{"repository": repository, "eventType": "issues"}, "provider refused")
	attachFactoryWebhookNode(t, support.CreateFactoryCanvas(t, r, factory.ID, "Issues").ID, issues.ID)
	foreign := failedFactoryWebhook(t, integration.ID, map[string]any{"repository": repository, "eventType": "issues"}, "provider refused")
	attachFactoryWebhookNode(t, support.CreateFactoryCanvas(t, r, other.ID, "Other").ID, foreign.ID)

	_, err = RetryFactoryWebhook(t.Context(), r.Organization.ID.String(), &pb.RetryFactoryWebhookRequest{FactoryId: factory.ID.String(), WebhookId: foreign.ID.String()})
	require.Error(t, err)
	response, err := RetryFactoryWebhook(t.Context(), r.Organization.ID.String(), &pb.RetryFactoryWebhookRequest{FactoryId: factory.ID.String(), WebhookId: owned.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStatePending, response.GetWebhook().GetState())
	issuesResponse, err := RetryFactoryWebhook(t.Context(), r.Organization.ID.String(), &pb.RetryFactoryWebhookRequest{FactoryId: factory.ID.String(), WebhookId: issues.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStatePending, issuesResponse.GetWebhook().GetState())
	_, err = RetryFactoryWebhook(t.Context(), r.Organization.ID.String(), &pb.RetryFactoryWebhookRequest{FactoryId: factory.ID.String(), WebhookId: owned.ID.String()})
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.Webhook{}).Where("id = ?", owned.ID).Update("state", models.WebhookStateProvisioning).Error)
	provisioning, err := RetryFactoryWebhook(t.Context(), r.Organization.ID.String(), &pb.RetryFactoryWebhookRequest{FactoryId: factory.ID.String(), WebhookId: owned.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStateProvisioning, provisioning.GetWebhook().GetState())
	var count int64
	require.NoError(t, db.Model(&models.Webhook{}).Where("id = ?", owned.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func failedFactoryWebhook(t *testing.T, integrationID uuid.UUID, configuration map[string]any, reason string) *models.Webhook {
	t.Helper()
	now := time.Now()
	webhook := &models.Webhook{ID: uuid.New(), State: models.WebhookStatePending, Secret: []byte("test-secret"), AppInstallationID: &integrationID, Configuration: datatypes.NewJSONType(any(configuration)), CreatedAt: &now, UpdatedAt: &now}
	require.NoError(t, database.DB(t.Context()).Create(webhook).Error)
	require.NoError(t, webhook.Fail(database.DB(t.Context()), reason))
	return webhook
}

func attachFactoryWebhookNode(t *testing.T, canvasID, webhookID uuid.UUID) {
	t.Helper()
	node := &models.CanvasNode{WorkflowID: canvasID, NodeID: uuid.NewString(), Type: models.NodeTypeTrigger, WebhookID: &webhookID}
	require.NoError(t, database.DB(t.Context()).Create(node).Error)
}
