package eventdistributer

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func Test__RunStateToWsEvent(t *testing.T) {
	assert.Equal(t, RunPendingEvent, runStateToWsEvent(models.CanvasRunStatePending))
	assert.Equal(t, RunStartedEvent, runStateToWsEvent(models.CanvasRunStateStarted))
	assert.Equal(t, RunCancellingEvent, runStateToWsEvent(models.CanvasRunStateCancelling))
	assert.Equal(t, RunFinishedEvent, runStateToWsEvent(models.CanvasRunStateFinished))
	assert.Empty(t, runStateToWsEvent("unknown"))
}

func Test__GroupChildRunsByExecutionID(t *testing.T) {
	firstExecutionID := uuid.New()
	secondExecutionID := uuid.New()
	firstRun := models.CanvasRun{ID: uuid.New(), ParentExecutionID: &firstExecutionID}
	secondRun := models.CanvasRun{ID: uuid.New(), ParentExecutionID: &secondExecutionID}
	runWithoutParent := models.CanvasRun{ID: uuid.New()}

	grouped := groupChildRunsByExecutionID([]models.CanvasRun{firstRun, secondRun, runWithoutParent})

	require.Len(t, grouped, 2)
	assert.Equal(t, []models.CanvasRun{firstRun}, grouped[firstExecutionID.String()])
	assert.Equal(t, []models.CanvasRun{secondRun}, grouped[secondExecutionID.String()])
}

func Test__MarshalCanvasRunJSON__EmitsEmptyQueueItems(t *testing.T) {
	payload, err := marshalCanvasRunJSON(&pb.CanvasRun{
		Id:         "run-1",
		QueueItems: []*pb.CanvasNodeQueueItem{},
	})
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"queueItems":[]`)
}
