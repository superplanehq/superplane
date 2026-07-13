package eventdistributer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func Test__RunStateToWsEvent(t *testing.T) {
	assert.Equal(t, RunStartedEvent, runStateToWsEvent(models.CanvasRunStateStarted))
	assert.Equal(t, RunFinishedEvent, runStateToWsEvent(models.CanvasRunStateFinished))
	assert.Empty(t, runStateToWsEvent("unknown"))
}

func Test__MarshalCanvasRunJSON__EmitsEmptyQueueItems(t *testing.T) {
	payload, err := marshalCanvasRunJSON(&pb.CanvasRun{
		Id:         "run-1",
		QueueItems: []*pb.CanvasNodeQueueItem{},
	})
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"queueItems":[]`)
}
