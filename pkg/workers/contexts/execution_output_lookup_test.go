package contexts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test_outputEvent_resolvesByIncomingEventID(t *testing.T) {
	executionID := uuid.New()
	firstEventID := uuid.New()
	secondEventID := uuid.New()
	now := time.Now()

	events := []models.CanvasEvent{
		{ID: firstEventID, ExecutionID: &executionID, CreatedAt: &now},
		{ID: secondEventID, ExecutionID: &executionID, CreatedAt: &now},
	}

	lookup := executionOutputLookup{
		eventsByID:            indexEventsByID(events),
		eventsByExecutionID:   indexEventsByExecutionID(events),
		consumedEventByParent: nil,
		incomingEventID:       &secondEventID,
	}

	event, ok, err := lookup.outputEvent(executionID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, secondEventID, event.ID)
}
