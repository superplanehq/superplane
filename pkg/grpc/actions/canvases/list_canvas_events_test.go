package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListCanvasEvents__ReturnsEventsWithExecutions(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger-0",
				Name:   "Trigger 0",
				Type:   models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "start"},
				}),
			},
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "node-2",
				Name:   "Node 2",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// First root event
	//
	rootEvent1 := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-0", "default", nil)
	customName := "Root Event 1"
	rootEvent1.CustomName = &customName
	require.NoError(t, database.Conn().Save(rootEvent1).Error)

	firstExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent1.ID, rootEvent1.ID, nil)
	_, err := firstExecution.Pass(map[string][]any{
		"default": {map[string]any{"data": "first"}},
	})

	require.NoError(t, err)
	secondExecution := support.CreateNextNodeExecution(t, canvas.ID, "node-2", rootEvent1.ID, rootEvent1.ID, &firstExecution.ID)
	_, err = secondExecution.Pass(map[string][]any{
		"default": {map[string]any{"data": "second"}},
	})
	require.NoError(t, err)

	//
	// Second root event, no executions
	//
	rootEvent2 := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-0", "default", nil)
	customName2 := "Root Event 2"
	rootEvent2.CustomName = &customName2
	require.NoError(t, database.Conn().Save(rootEvent2).Error)

	//
	// Verify endpoint returns proper results
	//

	response, err := ListCanvasEvents(context.Background(), r.Registry, canvas.ID, 0, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Events, 2)

	event1 := findCanvasEventWithExecutions(response.Events, rootEvent1.ID.String())
	require.NotNil(t, event1)
	require.Len(t, event1.Executions, 2)
	assert.Equal(t, customName, event1.CustomName)

	event2 := findCanvasEventWithExecutions(response.Events, rootEvent2.ID.String())
	require.NotNil(t, event2)
	assert.Empty(t, event2.Executions)
}

func findCanvasEventWithExecutions(events []*pb.CanvasEventWithExecutions, id string) *pb.CanvasEventWithExecutions {
	for _, event := range events {
		if event.Id == id {
			return event
		}
	}

	return nil
}
