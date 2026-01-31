package contexts

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__IntegrationContext_ScheduleResync(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	//
	// Create app installation
	//
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("installation"),
		map[string]any{},
	)
	require.NoError(t, err)

	ctx := NewIntegrationContext(database.Conn(), nil, integration, r.Encryptor, r.Registry)

	t.Run("rejects short interval", func(t *testing.T) {
		err = ctx.ScheduleResync(500 * time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "interval must be bigger than 1s")
	})

	t.Run("completes previous request on new request", func(t *testing.T) {
		//
		// Create previous request
		//
		now := time.Now()
		require.NoError(t, integration.CreateSyncRequest(database.Conn(), &now))
		requests, err := integration.ListRequests(models.IntegrationRequestTypeSync)
		require.NoError(t, err)
		require.Len(t, requests, 1)
		previousRequest := &requests[0]

		//
		// Schedule new sync request.
		//
		require.NoError(t, ctx.ScheduleResync(2*time.Second))

		//
		// Verify previous request was completed.
		//
		previousRequest, err = integration.GetRequest(previousRequest.ID.String())
		require.NoError(t, err)
		require.Equal(t, models.IntegrationRequestStateCompleted, previousRequest.State)

		//
		// Verify new one was created
		//
		requests, err = integration.ListRequests(models.IntegrationRequestTypeSync)
		require.NoError(t, err)
		require.Len(t, requests, 2)
		newRequestIndex := slices.IndexFunc(requests, func(r models.IntegrationRequest) bool { return r.ID.String() != previousRequest.ID.String() })
		newRequest := requests[newRequestIndex]
		require.Equal(t, models.IntegrationRequestStatePending, newRequest.State)
	})
}

func Test__IntegrationContext_RequestWebhook_ReplacesWebhookOnConfigChange(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
		OnCompareWebhookConfig: func(a, b any) (bool, error) {
			return reflect.DeepEqual(a, b), nil
		},
	})

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("installation"),
		map[string]any{},
	)
	require.NoError(t, err)

	oldConfig := map[string]any{"repository": "old"}
	newConfig := map[string]any{"repository": "new"}

	webhookID := uuid.New()
	_, encryptedKey, err := crypto.NewRandomKey(context.Background(), r.Encryptor, webhookID.String())
	require.NoError(t, err)

	now := time.Now()
	webhook := models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStateReady,
		Secret:            encryptedKey,
		Configuration:     datatypes.NewJSONType[any](oldConfig),
		Metadata:          datatypes.NewJSONType[any](map[string]any{}),
		AppInstallationID: &integration.ID,
		CreatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	inputNode := models.WorkflowNode{
		NodeID:        "node-1",
		Name:          "Node 1",
		Type:          models.NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(models.Position{}),
	}

	workflow, nodes := support.CreateWorkflow(t, r.Organization.ID, r.User, []models.WorkflowNode{inputNode}, nil)
	require.NotNil(t, workflow)
	require.Len(t, nodes, 1)

	node := nodes[0]
	node.AppInstallationID = &integration.ID
	node.WebhookID = &webhookID
	require.NoError(t, database.Conn().Save(&node).Error)

	ctx := NewIntegrationContext(database.Conn(), &node, integration, r.Encryptor, r.Registry)
	require.NoError(t, ctx.RequestWebhook(newConfig))

	require.NotNil(t, node.WebhookID)
	require.NotEqual(t, webhookID, *node.WebhookID)

	var deletedWebhook models.Webhook
	require.NoError(t, database.Conn().Unscoped().First(&deletedWebhook, webhookID).Error)
	require.True(t, deletedWebhook.DeletedAt.Valid)

	newWebhook, err := models.FindWebhookInTransaction(database.Conn(), *node.WebhookID)
	require.NoError(t, err)
	require.False(t, newWebhook.DeletedAt.Valid)
	assert.Equal(t, newConfig, newWebhook.Configuration.Data())
}
