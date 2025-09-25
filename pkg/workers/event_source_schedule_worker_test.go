package workers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__EventSourceScheduleWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})
	defer r.Close()

	now := time.Now()
	w, err := NewEventSourceScheduleWorker(func() time.Time {
		return now
	})
	require.NoError(t, err)

	t.Run("processes daily schedule and creates event", func(t *testing.T) {
		// Create event source with daily schedule at 1 hour ago (should trigger)
		pastHour := now.Add(-time.Hour).Format("15:04")
		schedule := &models.Schedule{
			Type: models.ScheduleTypeDaily,
			Daily: &models.DailySchedule{
				Time: pastHour,
			},
		}

		eventSource, _, err := builders.NewEventSourceBuilder(r.Encryptor, r.Registry).
			InCanvas(r.Canvas.ID).
			WithName("scheduled-source").
			WithSchedule(schedule).
			Create()
		require.NoError(t, err)

		// Manually set next trigger to past time to make it due
		pastTime := now.Add(-30 * time.Minute)
		err = eventSource.UpdateNextTrigger(pastTime)
		require.NoError(t, err)

		// Run worker tick
		err = w.Tick()
		require.NoError(t, err)

		// Verify event was created
		events, err := models.ListEventsBySourceID(eventSource.ID)
		require.NoError(t, err)
		require.Len(t, events, 1)

		event := events[0]
		assert.Equal(t, eventSource.ID, event.SourceID)
		assert.Equal(t, eventSource.CanvasID, event.CanvasID)
		assert.Equal(t, eventSource.Name, event.SourceName)
		assert.Equal(t, models.SourceTypeEventSource, event.SourceType)
		assert.Equal(t, "scheduled", event.Type)
		assert.Equal(t, models.EventStatePending, event.State)

		// Verify schedule was updated with next trigger (should be tomorrow at the same time)
		updatedEventSource, err := models.FindEventSource(eventSource.ID)
		require.NoError(t, err)
		assert.NotNil(t, updatedEventSource.NextTriggerAt)
		assert.True(t, updatedEventSource.NextTriggerAt.After(now))
		assert.NotNil(t, updatedEventSource.LastTriggeredAt)
	})

	t.Run("processes weekly schedule and creates event", func(t *testing.T) {
		// Create event source with weekly schedule for current weekday, 1 hour ago
		weekdayStr := models.WeekdayToString(now.Weekday())
		pastHour := now.Add(-time.Hour).Format("15:04")
		schedule := &models.Schedule{
			Type: models.ScheduleTypeWeekly,
			Weekly: &models.WeeklySchedule{
				WeekDay: weekdayStr,
				Time:    pastHour,
			},
		}

		eventSource, _, err := builders.NewEventSourceBuilder(r.Encryptor, r.Registry).
			InCanvas(r.Canvas.ID).
			WithName("weekly-scheduled-source").
			WithSchedule(schedule).
			Create()
		require.NoError(t, err)

		// Manually set next trigger to past time to make it due
		pastTime := now.Add(-30 * time.Minute)
		err = eventSource.UpdateNextTrigger(pastTime)
		require.NoError(t, err)

		// Run worker tick
		err = w.Tick()
		require.NoError(t, err)

		// Verify event was created
		events, err := models.ListEventsBySourceID(eventSource.ID)
		require.NoError(t, err)
		require.Len(t, events, 1)

		// Verify schedule was updated with next trigger (should be next week)
		updatedEventSource, err := models.FindEventSource(eventSource.ID)
		require.NoError(t, err)
		assert.NotNil(t, updatedEventSource.NextTriggerAt)
		assert.True(t, updatedEventSource.NextTriggerAt.After(now))
		assert.NotNil(t, updatedEventSource.LastTriggeredAt)
	})

	t.Run("does not process future schedules", func(t *testing.T) {
		// Create event source with daily schedule at 1 hour in the future
		futureHour := now.Add(time.Hour).Format("15:04")
		schedule := &models.Schedule{
			Type: models.ScheduleTypeDaily,
			Daily: &models.DailySchedule{
				Time: futureHour,
			},
		}

		eventSource, _, err := builders.NewEventSourceBuilder(r.Encryptor, r.Registry).
			InCanvas(r.Canvas.ID).
			WithName("future-scheduled-source").
			WithSchedule(schedule).
			Create()
		require.NoError(t, err)

		// Run worker tick
		err = w.Tick()
		require.NoError(t, err)

		// Verify no events were created
		events, err := models.ListEventsBySourceID(eventSource.ID)
		require.NoError(t, err)
		require.Len(t, events, 0)
	})

	t.Run("handles multiple due schedules", func(t *testing.T) {
		// Create two event sources with schedules due for processing
		pastHour1 := now.Add(-2 * time.Hour).Format("15:04")
		pastHour2 := now.Add(-time.Hour).Format("15:04")

		schedule1 := &models.Schedule{
			Type: models.ScheduleTypeDaily,
			Daily: &models.DailySchedule{
				Time: pastHour1,
			},
		}
		schedule2 := &models.Schedule{
			Type: models.ScheduleTypeDaily,
			Daily: &models.DailySchedule{
				Time: pastHour2,
			},
		}

		eventSource1, _, err := builders.NewEventSourceBuilder(r.Encryptor, r.Registry).
			InCanvas(r.Canvas.ID).
			WithName("multi-scheduled-source-1").
			WithSchedule(schedule1).
			Create()
		require.NoError(t, err)

		eventSource2, _, err := builders.NewEventSourceBuilder(r.Encryptor, r.Registry).
			InCanvas(r.Canvas.ID).
			WithName("multi-scheduled-source-2").
			WithSchedule(schedule2).
			Create()
		require.NoError(t, err)

		// Set both schedules to be due
		pastTime1 := now.Add(-45 * time.Minute)
		err = eventSource1.UpdateNextTrigger(pastTime1)
		require.NoError(t, err)

		pastTime2 := now.Add(-15 * time.Minute)
		err = eventSource2.UpdateNextTrigger(pastTime2)
		require.NoError(t, err)

		// Run worker tick
		err = w.Tick()
		require.NoError(t, err)

		// Verify events were created for both sources
		events1, err := models.ListEventsBySourceID(eventSource1.ID)
		require.NoError(t, err)
		require.Len(t, events1, 1)

		events2, err := models.ListEventsBySourceID(eventSource2.ID)
		require.NoError(t, err)
		require.Len(t, events2, 1)
	})
}