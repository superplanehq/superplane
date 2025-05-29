package workers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__TimeWindowWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	//
	// Stage's time window is on week days, 08:00-17:00
	//
	conditions := []models.StageCondition{
		{
			Type: models.StageConditionTypeTimeWindow,
			TimeWindow: &models.TimeWindowCondition{
				Start:    "08:00",
				End:      "17:00",
				WeekDays: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
			},
		},
	}

	require.NoError(t, r.Canvas.CreateStage("stage-1", r.User.String(), conditions, support.ExecutorSpec(), []models.StageConnection{
		{
			SourceID:   r.Source.ID,
			SourceType: models.SourceTypeEventSource,
		},
	}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}))

	stage, err := r.Canvas.FindStageByName("stage-1")
	require.NoError(t, err)

	t.Run("event is not in time window -> does nothing", func(t *testing.T) {
		w, _ := NewTimeWindowWorker(
			func() time.Time {
				// 02:00, 1st of January of 2025 (Wednesday),
				// which is outside the 08:00-17:00 time window for the stage.
				return time.Date(2025, 1, 1, 2, 0, 0, 0, time.UTC)
			},
		)

		// Create stage event and move it to waiting state
		event := support.CreateStageEvent(t, r.Source, stage)
		require.NoError(t, event.UpdateState(models.StageEventStateWaiting, models.StageEventStateReasonTimeWindow))

		// Trigger the worker and verify event remains in waiting state
		require.NoError(t, w.Tick())
		event, err := models.FindStageEventByID(event.ID.String(), event.StageID.String())
		require.NoError(t, err)
		require.Equal(t, models.StageEventStateWaiting, event.State)
		require.Equal(t, models.StageEventStateReasonTimeWindow, event.StateReason)
	})

	t.Run("event is in time window -> moves to pending", func(t *testing.T) {
		w, _ := NewTimeWindowWorker(
			func() time.Time {
				// 10:00, 1st of January of 2025 (Wednesday)
				// which is inside the 08:00-17:00 time window for the stage.
				return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
			},
		)

		// Create stage event and move it to waiting state
		event := support.CreateStageEvent(t, r.Source, stage)
		require.NoError(t, event.UpdateState(models.StageEventStateWaiting, models.StageEventStateReasonTimeWindow))

		// Trigger the worker and verify event is moved to pending state
		require.NoError(t, w.Tick())
		event, err := models.FindStageEventByID(event.ID.String(), event.StageID.String())
		require.NoError(t, err)
		require.Equal(t, models.StageEventStatePending, event.State)
		require.Empty(t, event.StateReason)
	})
}
