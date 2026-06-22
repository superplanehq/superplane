package contexts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test_consumedEventByParentFromRunChain(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	childEventID := uuid.New()

	runChain := []models.CanvasNodeExecution{
		{ID: parentID},
		{
			ID:                  childID,
			EventID:             childEventID,
			PreviousExecutionID: &parentID,
		},
	}

	consumedEventByParent := consumedEventByParentFromRunChain(runChain)
	assert.Equal(t, childEventID, consumedEventByParent[parentID])
}

func Test_outputEvent_usesConsumedEventWhenMultipleOutputs(t *testing.T) {
	parentExecutionID := uuid.New()
	consumedEventID := uuid.New()
	otherEventID := uuid.New()
	now := time.Now()

	events := []models.CanvasEvent{
		{
			ID:          otherEventID,
			ExecutionID: &parentExecutionID,
			CreatedAt:   &now,
			Data:        models.NewJSONValue(map[string]any{"item": "a"}),
		},
		{
			ID:          consumedEventID,
			ExecutionID: &parentExecutionID,
			CreatedAt:   &now,
			Data:        models.NewJSONValue(map[string]any{"item": "b"}),
		},
	}

	lookup := executionOutputLookup{
		eventsByID:            indexEventsByID(events),
		eventsByExecutionID:   indexEventsByExecutionID(events),
		consumedEventByParent: map[uuid.UUID]uuid.UUID{parentExecutionID: consumedEventID},
	}

	event, ok, err := lookup.outputEvent(parentExecutionID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, consumedEventID, event.ID)
}

func Test_outputEvent_singleOutput(t *testing.T) {
	executionID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	events := []models.CanvasEvent{
		{ID: eventID, ExecutionID: &executionID, CreatedAt: &now},
	}

	lookup := executionOutputLookup{
		eventsByID:            indexEventsByID(events),
		eventsByExecutionID:   indexEventsByExecutionID(events),
		consumedEventByParent: nil,
	}

	event, ok, err := lookup.outputEvent(executionID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, eventID, event.ID)
}

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

func Test_outputEvent_ambiguousWithoutConsumedEvent(t *testing.T) {
	executionID := uuid.New()
	now := time.Now()

	events := []models.CanvasEvent{
		{ID: uuid.New(), ExecutionID: &executionID, CreatedAt: &now},
		{ID: uuid.New(), ExecutionID: &executionID, CreatedAt: &now},
	}

	lookup := executionOutputLookup{
		eventsByID:            indexEventsByID(events),
		eventsByExecutionID:   indexEventsByExecutionID(events),
		consumedEventByParent: nil,
	}

	_, ok, err := lookup.outputEvent(executionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous outputs")
	assert.False(t, ok)
}

func Test_consumedEventIDsFromMap(t *testing.T) {
	first := uuid.New()
	second := uuid.New()

	ids := consumedEventIDsFromMap(map[uuid.UUID]uuid.UUID{
		uuid.New(): first,
		uuid.New(): second,
		uuid.New(): uuid.Nil,
	})

	assert.Len(t, ids, 2)
	assert.ElementsMatch(t, []uuid.UUID{first, second}, ids)
}

func Test_unionExecutionIDs(t *testing.T) {
	first := uuid.New()
	second := uuid.New()
	shared := uuid.New()

	result := unionExecutionIDs(
		[]uuid.UUID{first, shared},
		[]uuid.UUID{second, shared},
	)

	assert.Len(t, result, 3)
	assert.ElementsMatch(t, []uuid.UUID{first, second, shared}, result)
}
