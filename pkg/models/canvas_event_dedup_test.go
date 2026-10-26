package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestCanvasNodeCreatesOneRootEventPerDedupKey(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger},
	}, nil)
	node := nodes[0]

	create := func(key string) bool {
		t.Helper()
		now := time.Now()
		event := &models.CanvasEvent{
			WorkflowID: canvas.ID,
			NodeID:     node.NodeID,
			Channel:    "default",
			Data:       models.NewJSONValue(map[string]any{"revision": key}),
			State:      models.CanvasEventStatePending,
			CreatedAt:  &now,
		}
		created, err := node.CreateDeduplicatedRootEvent(db, event, key)
		require.NoError(t, err)
		return created
	}

	assert.True(t, create("pull-request|abc123"))
	assert.False(t, create("pull-request|abc123"))
	assert.True(t, create("pull-request|def456"))
	assert.True(t, create(""))
	assert.True(t, create(""))

	var count int64
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
	assert.EqualValues(t, 4, count)
}
