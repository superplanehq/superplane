package workers

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

func TestEventDeletionWorker_Tick(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	defer r.Close()

	canvasID := uuid.New()
	sourceID := uuid.New()

	oldTime := time.Now().Add(-4 * 30 * 24 * time.Hour)
	oldRejectedEvent := models.Event{
		SourceID:     sourceID,
		CanvasID:     canvasID,
		SourceName:   "test-source",
		SourceType:   models.SourceTypeEventSource,
		Type:         "test",
		State:        models.EventStateRejected,
		StateReason:  models.EventStateReasonFiltered,
		StateMessage: "",
		ReceivedAt:   &oldTime,
		Raw:          datatypes.JSON([]byte(`{"test": "data"}`)),
		Headers:      datatypes.JSON([]byte(`{}`)),
	}

	err := database.Conn().Create(&oldRejectedEvent).Error
	require.NoError(t, err)

	recentTime := time.Now().Add(-1 * 24 * time.Hour)
	recentRejectedEvent := models.Event{
		SourceID:     sourceID,
		CanvasID:     canvasID,
		SourceName:   "test-source",
		SourceType:   models.SourceTypeEventSource,
		Type:         "test",
		State:        models.EventStateRejected,
		StateReason:  models.EventStateReasonFiltered,
		StateMessage: "",
		ReceivedAt:   &recentTime,
		Raw:          datatypes.JSON([]byte(`{"test": "data"}`)),
		Headers:      datatypes.JSON([]byte(`{}`)),
	}

	err = database.Conn().Create(&recentRejectedEvent).Error
	require.NoError(t, err)

	oldProcessedEvent := models.Event{
		SourceID:     sourceID,
		CanvasID:     canvasID,
		SourceName:   "test-source",
		SourceType:   models.SourceTypeEventSource,
		Type:         "test",
		State:        models.EventStateProcessed,
		StateReason:  models.EventStateReasonOk,
		StateMessage: "",
		ReceivedAt:   &oldTime,
		Raw:          datatypes.JSON([]byte(`{"test": "data"}`)),
		Headers:      datatypes.JSON([]byte(`{}`)),
	}

	err = database.Conn().Create(&oldProcessedEvent).Error
	require.NoError(t, err)

	var eventCount int64
	err = database.Conn().Model(&models.Event{}).Count(&eventCount).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(3), eventCount)

	worker := NewEventDeletionWorker()
	err = worker.Tick()
	assert.NoError(t, err)

	err = database.Conn().Model(&models.Event{}).Count(&eventCount).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(2), eventCount)

	var remainingEvents []models.Event
	err = database.Conn().Find(&remainingEvents).Error
	assert.NoError(t, err)
	assert.Len(t, remainingEvents, 2)

	eventIDs := make([]uuid.UUID, len(remainingEvents))
	for i, event := range remainingEvents {
		eventIDs[i] = event.ID
	}

	assert.Contains(t, eventIDs, recentRejectedEvent.ID)
	assert.Contains(t, eventIDs, oldProcessedEvent.ID)
	assert.NotContains(t, eventIDs, oldRejectedEvent.ID)
}

func TestEventDeletionWorker_CustomRetention(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	defer r.Close()

	canvasID := uuid.New()
	sourceID := uuid.New()

	oldTime := time.Now().Add(-2 * 24 * time.Hour)
	rejectedEvent := models.Event{
		SourceID:     sourceID,
		CanvasID:     canvasID,
		SourceName:   "test-source",
		SourceType:   models.SourceTypeEventSource,
		Type:         "test",
		State:        models.EventStateRejected,
		StateReason:  models.EventStateReasonFiltered,
		StateMessage: "",
		ReceivedAt:   &oldTime,
		Raw:          datatypes.JSON([]byte(`{"test": "data"}`)),
		Headers:      datatypes.JSON([]byte(`{}`)),
	}

	err := database.Conn().Create(&rejectedEvent).Error
	require.NoError(t, err)

	worker := &EventDeletionWorker{
		RetentionDuration: 24 * time.Hour,
	}

	err = worker.Tick()
	assert.NoError(t, err)

	var eventCount int64
	err = database.Conn().Model(&models.Event{}).Count(&eventCount).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(0), eventCount)
}
