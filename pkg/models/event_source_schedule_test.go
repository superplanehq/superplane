package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__CalculateNextTrigger(t *testing.T) {

	t.Run("daily schedule - same day, time not passed", func(t *testing.T) {
		now := time.Now().UTC()
		futureTime := now.Add(2 * time.Hour).Format("15:04")
		schedule := Schedule{
			Type: ScheduleTypeDaily,
			Daily: &DailySchedule{
				Time: futureTime,
			},
		}

		next, err := schedule.CalculateNextTrigger(now)
		require.NoError(t, err)

		// Should be today at the scheduled time
		assert.True(t, next.After(now))
		assert.True(t, next.Before(now.Add(24*time.Hour)))
	})

	t.Run("daily schedule - time already passed, next day", func(t *testing.T) {
		now := time.Now().UTC()
		pastTime := now.Add(-2 * time.Hour).Format("15:04")
		schedule := Schedule{
			Type: ScheduleTypeDaily,
			Daily: &DailySchedule{
				Time: pastTime,
			},
		}

		next, err := schedule.CalculateNextTrigger(now)
		require.NoError(t, err)

		// Should be tomorrow
		assert.True(t, next.After(now.Add(20*time.Hour)))
		assert.True(t, next.Before(now.Add(26*time.Hour)))
	})

	t.Run("daily schedule - exact time match, next day", func(t *testing.T) {
		now := time.Now().UTC()
		currentTime := now.Format("15:04")
		schedule := Schedule{
			Type: ScheduleTypeDaily,
			Daily: &DailySchedule{
				Time: currentTime,
			},
		}

		next, err := schedule.CalculateNextTrigger(now)
		require.NoError(t, err)

		// Should be tomorrow since exact match means it's already passed
		assert.True(t, next.After(now.Add(20*time.Hour)))
		assert.True(t, next.Before(now.Add(26*time.Hour)))
	})

	t.Run("weekly schedule - same day, time not passed", func(t *testing.T) {
		now := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 12, 0, 0, 0, time.UTC)
		weekdayStr := WeekdayToString(now.Weekday())
		futureTime := now.Add(2 * time.Hour).Format("15:04")
		schedule := Schedule{
			Type: ScheduleTypeWeekly,
			Weekly: &WeeklySchedule{
				WeekDay: weekdayStr,
				Time:    futureTime,
			},
		}

		next, err := schedule.CalculateNextTrigger(now)
		require.NoError(t, err)

		// Should be today at the scheduled time
		assert.True(t, next.After(now))
		assert.True(t, next.Before(now.Add(24*time.Hour)))
	})

	t.Run("weekly schedule - same day, time passed, next week", func(t *testing.T) {
		//
		// Use a fixed time to avoid midnight boundary issues
		//
		now := time.Date(2024, 1, 15, 15, 30, 0, 0, time.UTC) // Monday 3:30 PM UTC
		weekdayStr := WeekdayToString(now.Weekday())          // Monday
		pastTime := "10:00"                                   // 10 AM - clearly in the past
		schedule := Schedule{
			Type: ScheduleTypeWeekly,
			Weekly: &WeeklySchedule{
				WeekDay: weekdayStr,
				Time:    pastTime,
			},
		}

		next, err := schedule.CalculateNextTrigger(now)
		require.NoError(t, err)

		// Should be next Monday at 10:00 AM (7 days later)
		expectedNext := time.Date(2024, 1, 22, 10, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedNext, *next, "Expected next trigger to be next Monday at 10:00 AM")

		// Verify it's definitely next week
		assert.True(t, next.After(now.Add(6*24*time.Hour)), "Next trigger should be at least 6 days away")
		assert.True(t, next.Before(now.Add(8*24*time.Hour)), "Next trigger should be less than 8 days away")
	})

	t.Run("weekly schedule - different day in future", func(t *testing.T) {
		now := time.Now().UTC()

		// Get a future weekday (if today is Sunday, use Monday, otherwise use next day)
		currentWeekday := now.Weekday()
		var targetWeekday time.Weekday
		if currentWeekday == time.Sunday {
			targetWeekday = time.Monday
		} else {
			targetWeekday = currentWeekday + 1
		}

		schedule := Schedule{
			Type: ScheduleTypeWeekly,
			Weekly: &WeeklySchedule{
				WeekDay: WeekdayToString(targetWeekday),
				Time:    "14:00",
			},
		}

		next, err := schedule.CalculateNextTrigger(now)
		require.NoError(t, err)

		// Should be in the future but within a week
		assert.True(t, next.After(now))
		assert.True(t, next.Before(now.Add(7*24*time.Hour)))
	})

	t.Run("invalid time format", func(t *testing.T) {
		schedule := Schedule{
			Type: ScheduleTypeDaily,
			Daily: &DailySchedule{
				Time: "25:30",
			},
		}

		_, err := schedule.CalculateNextTrigger(time.Now().UTC())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hour must be between 0 and 23")
	})

	t.Run("invalid weekday", func(t *testing.T) {
		schedule := Schedule{
			Type: ScheduleTypeWeekly,
			Weekly: &WeeklySchedule{
				WeekDay: "invalid",
				Time:    "09:00",
			},
		}

		_, err := schedule.CalculateNextTrigger(time.Now().UTC())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid weekday")
	})

	t.Run("unsupported schedule type", func(t *testing.T) {
		schedule := Schedule{
			Type: "monthly",
		}

		_, err := schedule.CalculateNextTrigger(time.Now().UTC())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported schedule type")
	})

	t.Run("missing daily configuration", func(t *testing.T) {
		schedule := Schedule{
			Type: ScheduleTypeDaily,
		}

		_, err := schedule.CalculateNextTrigger(time.Now().UTC())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "daily schedule configuration is missing")
	})

	t.Run("missing weekly configuration", func(t *testing.T) {
		schedule := Schedule{
			Type: ScheduleTypeWeekly,
		}

		_, err := schedule.CalculateNextTrigger(time.Now().UTC())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "weekly schedule configuration is missing")
	})
}
