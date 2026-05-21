package contexts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func Test_preferredEventIDByExecution(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	childEventID := uuid.New()

	executions := []models.CanvasNodeExecution{
		{ID: parentID},
		{
			ID:                  childID,
			EventID:             childEventID,
			PreviousExecutionID: &parentID,
		},
	}

	preferred := preferredEventIDByExecution(executions)
	assert.Equal(t, childEventID, preferred[parentID])
}

func Test_eventForExecution_prefersBranchEvent(t *testing.T) {
	parentExecutionID := uuid.New()
	branchEventID := uuid.New()
	otherEventID := uuid.New()
	now := time.Now()

	events := []models.CanvasEvent{
		{
			ID:          otherEventID,
			ExecutionID: &parentExecutionID,
			CreatedAt:   &now,
			Data:        datatypes.NewJSONType[any](map[string]any{"item": "a"}),
		},
		{
			ID:          branchEventID,
			ExecutionID: &parentExecutionID,
			CreatedAt:   &now,
			Data:        datatypes.NewJSONType[any](map[string]any{"item": "b"}),
		},
	}

	eventsByID := indexEventsByID(events)
	preferred := map[uuid.UUID]uuid.UUID{parentExecutionID: branchEventID}

	event, ok, err := eventForExecution(parentExecutionID, events, eventsByID, preferred)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, branchEventID, event.ID)
}

func Test_eventForExecution_singleEventWithoutPreferred(t *testing.T) {
	executionID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	events := []models.CanvasEvent{
		{ID: eventID, ExecutionID: &executionID, CreatedAt: &now},
	}

	event, ok, err := eventForExecution(executionID, events, indexEventsByID(events), nil)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, eventID, event.ID)
}

func Test_eventForExecution_ambiguousWithoutBranchLink(t *testing.T) {
	executionID := uuid.New()
	now := time.Now()

	events := []models.CanvasEvent{
		{ID: uuid.New(), ExecutionID: &executionID, CreatedAt: &now},
		{ID: uuid.New(), ExecutionID: &executionID, CreatedAt: &now},
	}

	_, ok, err := eventForExecution(executionID, events, indexEventsByID(events), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous outputs")
	assert.False(t, ok)
}
