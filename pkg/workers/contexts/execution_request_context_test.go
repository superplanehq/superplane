package contexts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ExecutionRequestContext_DoesNotScheduleRequestWhenPersistedExecutionIsFinished(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	ctx := NewExecutionRequestContext(database.Conn(), execution)
	err := ctx.ScheduleActionCall("poll", map[string]any{}, time.Second)
	require.NoError(t, err)

	pending, err := models.CountPendingRequestsForExecutionsInTransaction(database.Conn(), []uuid.UUID{execution.ID})
	require.NoError(t, err)
	assert.Zero(t, pending)
}
