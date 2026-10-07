package linear

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

func TestReconcileWebhookDelivery__AsksForAdminWhenTheSecretIsRemoved(t *testing.T) {
	r := support.Setup(t)
	integration := linearDBIntegration(t, r)
	webhookID := linearDBWebhook(t, integration.ID, map[string]any{
		"teamId":       "t1",
		"resourceType": IssueResourceType,
	}, map[string]any{"appLevel": true})

	integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{OAuthScopes: "read,write"})
	integrationContext.IntegrationID = integration.ID.String()

	err := (&Linear{}).reconcileWebhookDelivery(linearSyncContext(integrationContext, &contexts.HTTPContext{}))
	require.ErrorIs(t, err, errNeedsAdminAuthorization)
	require.NotNil(t, integrationContext.BrowserAction)
	assert.Contains(t, integrationContext.BrowserAction.URL, "admin")

	accessToken, _ := findSecret(integrationContext, OAuthAccessToken)
	assert.Empty(t, accessToken)

	saved := reloadLinearWebhook(t, webhookID)
	assert.Equal(t, models.WebhookStateReady, saved.State)
}

func TestReconcileWebhookDelivery__RecreatesAPIWebhooksWhenAdminIsGranted(t *testing.T) {
	r := support.Setup(t)
	integration := linearDBIntegration(t, r)
	webhookID := linearDBWebhook(t, integration.ID, map[string]any{
		"teamId":       "t1",
		"resourceType": IssueResourceType,
		"appLevel":     true,
	}, map[string]any{"appLevel": true})

	integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{OAuthScopes: "read,write,admin"})
	integrationContext.IntegrationID = integration.ID.String()

	err := (&Linear{}).reconcileWebhookDelivery(linearSyncContext(integrationContext, &contexts.HTTPContext{}))
	require.NoError(t, err)
	assert.Nil(t, integrationContext.BrowserAction)

	saved := reloadLinearWebhook(t, webhookID)
	assert.Equal(t, models.WebhookStatePending, saved.State)
	assert.False(t, linearWebhookConfig(t, saved).AppLevel)
}

func TestReconcileWebhookDelivery__DeletesAPIWebhooksWhenTheSecretIsSet(t *testing.T) {
	r := support.Setup(t)
	integration := linearDBIntegration(t, r)
	webhookID := linearDBWebhook(t, integration.ID, map[string]any{
		"teamId":       "t1",
		"resourceType": IssueResourceType,
	}, map[string]any{"id": "w1"})

	integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{OAuthScopes: "read,write,admin"})
	integrationContext.IntegrationID = integration.ID.String()
	integrationContext.Configuration["webhookSecret"] = "app-signing-secret"
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"webhookDelete":{"success":true}}}`),
		},
	}

	err := (&Linear{}).reconcileWebhookDelivery(linearSyncContext(integrationContext, httpContext))
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	saved := reloadLinearWebhook(t, webhookID)
	assert.Equal(t, models.WebhookStatePending, saved.State)
	assert.True(t, linearWebhookConfig(t, saved).AppLevel)
}

func TestReconcileWebhookDelivery__RecordsTheDeliveryModeWithoutRebuilding(t *testing.T) {
	r := support.Setup(t)
	integration := linearDBIntegration(t, r)
	webhookID := linearDBWebhook(t, integration.ID, map[string]any{
		"teamId":       "t1",
		"resourceType": IssueResourceType,
	}, map[string]any{"appLevel": true})

	integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{OAuthScopes: "read,write"})
	integrationContext.IntegrationID = integration.ID.String()
	integrationContext.Configuration["webhookSecret"] = "app-signing-secret"
	httpContext := &contexts.HTTPContext{}

	err := (&Linear{}).reconcileWebhookDelivery(linearSyncContext(integrationContext, httpContext))
	require.NoError(t, err)
	assert.Empty(t, httpContext.Requests)

	saved := reloadLinearWebhook(t, webhookID)
	assert.Equal(t, models.WebhookStateReady, saved.State)
	assert.True(t, linearWebhookConfig(t, saved).AppLevel)
}

func linearSyncContext(integration *contexts.IntegrationContext, httpContext core.HTTPContext) core.SyncContext {
	return core.SyncContext{
		BaseURL:     "https://sp.example.com",
		Integration: integration,
		HTTP:        httpContext,
		Logger:      logrus.NewEntry(logrus.New()),
	}
}

func linearDBIntegration(t *testing.T, r *support.ResourceRegistry) *models.Integration {
	t.Helper()
	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "linear", "linear-"+uuid.NewString()[:8], nil)
	require.NoError(t, err)
	return integration
}

func linearDBWebhook(t *testing.T, integrationID uuid.UUID, configuration, metadata map[string]any) uuid.UUID {
	t.Helper()
	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStateReady,
		Secret:            []byte("secret"),
		AppInstallationID: &integrationID,
		Configuration:     datatypes.NewJSONType[any](configuration),
		Metadata:          datatypes.NewJSONType[any](metadata),
	}).Error)
	return webhookID
}

func reloadLinearWebhook(t *testing.T, webhookID uuid.UUID) *models.Webhook {
	t.Helper()
	saved, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	return saved
}

func linearWebhookConfig(t *testing.T, webhook *models.Webhook) WebhookConfiguration {
	t.Helper()
	config := WebhookConfiguration{}
	require.NoError(t, mapstructure.Decode(webhook.Configuration.Data(), &config))
	return config
}
