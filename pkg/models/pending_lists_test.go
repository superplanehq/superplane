package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestPendingListsRespectLimit(t *testing.T) {
	t.Run("node executions", func(t *testing.T) {
		r := support.Setup(t)
		defer r.Close()

		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "component", Type: models.NodeTypeComponent},
		}, []models.Edge{{SourceID: "trigger", TargetID: "component", Channel: "default"}})

		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		for range 3 {
			support.CreateCanvasNodeExecution(t, canvas.ID, "component", rootEvent.ID, rootEvent.ID, nil)
		}

		executions, err := models.ListPendingNodeExecutions(2)
		require.NoError(t, err)
		assert.Len(t, executions, 2)
	})

	t.Run("canvas events", func(t *testing.T) {
		r := support.Setup(t)
		defer r.Close()

		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
		}, nil)

		for range 3 {
			support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		}

		events, err := models.ListPendingCanvasEvents(2)
		require.NoError(t, err)
		assert.Len(t, events, 2)
	})

	t.Run("node requests", func(t *testing.T) {
		r := support.Setup(t)
		defer r.Close()

		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
		}, nil)

		for range 3 {
			require.NoError(t, database.Conn().Create(&models.CanvasNodeRequest{
				ID:         uuid.New(),
				WorkflowID: canvas.ID,
				NodeID:     "trigger",
				Type:       models.NodeRequestTypeInvokeAction,
				State:      models.NodeExecutionRequestStatePending,
				RunAt:      time.Now().Add(-time.Second),
			}).Error)
		}

		requests, err := models.ListNodeRequests(2)
		require.NoError(t, err)
		assert.Len(t, requests, 2)
	})

	t.Run("integration requests", func(t *testing.T) {
		r := support.Setup(t)
		defer r.Close()

		integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
		require.NoError(t, err)

		for range 3 {
			require.NoError(t, database.Conn().Create(&models.IntegrationRequest{
				ID:                uuid.New(),
				AppInstallationID: integration.ID,
				State:             models.IntegrationRequestStatePending,
				Type:              models.IntegrationRequestTypeSync,
				RunAt:             time.Now().Add(-time.Second),
			}).Error)
		}

		requests, err := models.ListIntegrationRequests(2)
		require.NoError(t, err)
		assert.Len(t, requests, 2)
	})

	t.Run("webhooks", func(t *testing.T) {
		r := support.Setup(t)
		defer r.Close()

		for range 3 {
			require.NoError(t, database.Conn().Create(&models.Webhook{
				ID:         uuid.New(),
				State:      models.WebhookStatePending,
				Secret:     []byte("secret"),
				MaxRetries: 3,
			}).Error)
		}

		webhooks, err := models.ListPendingWebhooks(2)
		require.NoError(t, err)
		assert.Len(t, webhooks, 2)
	})

	t.Run("ready nodes", func(t *testing.T) {
		r := support.Setup(t)
		defer r.Close()

		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "component-1", Type: models.NodeTypeComponent},
			{NodeID: "component-2", Type: models.NodeTypeComponent},
			{NodeID: "component-3", Type: models.NodeTypeComponent},
		}, []models.Edge{
			{SourceID: "trigger", TargetID: "component-1", Channel: "default"},
			{SourceID: "trigger", TargetID: "component-2", Channel: "default"},
			{SourceID: "trigger", TargetID: "component-3", Channel: "default"},
		})

		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		support.CreateQueueItem(t, canvas.ID, "component-1", rootEvent.ID, rootEvent.ID)
		support.CreateQueueItem(t, canvas.ID, "component-2", rootEvent.ID, rootEvent.ID)
		support.CreateQueueItem(t, canvas.ID, "component-3", rootEvent.ID, rootEvent.ID)

		nodes, err := models.ListCanvasNodesReady(2)
		require.NoError(t, err)
		assert.Len(t, nodes, 2)
	})
}

func TestLockRequestsSkipStaleRows(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger},
	}, nil)

	nodeRequest := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     "trigger",
		Type:       models.NodeRequestTypeInvokeAction,
		State:      models.NodeExecutionRequestStateCompleted,
		RunAt:      time.Now().Add(-time.Second),
	}
	require.NoError(t, database.Conn().Create(&nodeRequest).Error)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, err := models.LockNodeRequest(tx, nodeRequest.ID)
		return err
	})
	require.Error(t, err)

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)

	integrationRequest := models.IntegrationRequest{
		ID:                uuid.New(),
		AppInstallationID: integration.ID,
		Type:              models.IntegrationRequestTypeSync,
		State:             models.IntegrationRequestStateCompleted,
		RunAt:             time.Now().Add(-time.Second),
	}
	require.NoError(t, database.Conn().Create(&integrationRequest).Error)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		_, err := models.LockIntegrationRequest(tx, integrationRequest.ID)
		return err
	})
	require.Error(t, err)
}
