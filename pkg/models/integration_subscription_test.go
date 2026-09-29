package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListIntegrationSubscriptions_OmitsDeletedCanvases(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	integration, err := models.CreateIntegration(
		uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"),
		map[string]any{},
	)
	require.NoError(t, err)

	liveCanvas, liveNodes := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger-live",
				Name:   "trigger-live",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onIssue"}}),
			},
		},
		nil,
	)
	require.NotNil(t, liveCanvas)

	deletedCanvas, deletedCanvasNodes := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger-deleted",
				Name:   "trigger-deleted",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onIssue"}}),
			},
		},
		nil,
	)
	require.NotNil(t, deletedCanvas)

	liveNode := liveNodes[0]
	liveNode.AppInstallationID = &integration.ID
	require.NoError(t, database.Conn().Save(&liveNode).Error)

	deletedNode := deletedCanvasNodes[0]
	deletedNode.AppInstallationID = &integration.ID
	require.NoError(t, database.Conn().Save(&deletedNode).Error)

	_, err = models.CreateIntegrationSubscription(&liveNode, integration, map[string]any{"resources": []string{"issue"}})
	require.NoError(t, err)

	_, err = models.CreateIntegrationSubscription(&deletedNode, integration, map[string]any{"resources": []string{"issue"}})
	require.NoError(t, err)

	subscriptions, err := models.ListIntegrationSubscriptions(database.Conn(), integration.ID)
	require.NoError(t, err)
	assert.Len(t, subscriptions, 2)

	require.NoError(t, database.Conn().Delete(&models.Canvas{}, deletedCanvas.ID).Error)

	subscriptions, err = models.ListIntegrationSubscriptions(database.Conn(), integration.ID)
	require.NoError(t, err)
	assert.Len(t, subscriptions, 1)
	assert.Equal(t, liveCanvas.ID, subscriptions[0].WorkflowID)
}
