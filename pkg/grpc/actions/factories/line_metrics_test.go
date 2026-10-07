package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__aggregateLineMetrics(t *testing.T) {
	now := time.Date(2026, 8, 19, 15, 0, 0, 0, time.Local)
	lineA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	lineB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	_, currentStart, currentEnd := lineMetricsWindowBounds(now)

	t.Run("empty rows produce no metrics", func(t *testing.T) {
		assert.Empty(t, aggregateLineMetrics(nil, now))
	})

	t.Run("omits lines with no closes in the current window", func(t *testing.T) {
		rows := []models.ClosedWorkOrderMetricRow{
			{
				WorkOrderID: uuid.New(),
				LineID:      lineA,
				ClosedAt:    currentStart.Add(-time.Hour),
				Merged:      true,
			},
		}
		assert.Empty(t, aggregateLineMetrics(rows, now))
	})

	t.Run("all failed closes yield zero success and omit duration and cost", func(t *testing.T) {
		rows := []models.ClosedWorkOrderMetricRow{
			{WorkOrderID: uuid.New(), LineID: lineA, ClosedAt: now.Add(-time.Hour)},
			{WorkOrderID: uuid.New(), LineID: lineA, ClosedAt: now.Add(-2 * time.Hour)},
		}
		got := aggregateLineMetrics(rows, now)
		require.Len(t, got, 1)
		assert.Equal(t, lineA, got[0].LineID)
		assert.Equal(t, float64(0), got[0].SuccessRatePct)
		assert.Equal(t, int32(0), got[0].MergedCount)
		assert.Equal(t, int32(2), got[0].TotalClosedCount)
		assert.Nil(t, got[0].DurationMinutes)
		assert.Nil(t, got[0].CostPerSuccessUsd)
		assert.Equal(t, 0.0, got[0].ThroughputPerDay)
		require.Len(t, got[0].SuccessTrendPct, lineMetricsWindowDays)
		require.Len(t, got[0].ThroughputTrend, lineMetricsWindowDays)
	})

	t.Run("computes success rate duration cost and prior-window deltas", func(t *testing.T) {
		priorClose := currentStart.Add(-24 * time.Hour)
		rows := []models.ClosedWorkOrderMetricRow{
			{
				WorkOrderID:      uuid.New(),
				LineID:           lineA,
				ClosedAt:         now.Add(-time.Hour),
				Merged:           true,
				ExecutionMinutes: 60,
				CostCents:        640,
			},
			{
				WorkOrderID: uuid.New(),
				LineID:      lineA,
				ClosedAt:    now.Add(-2 * time.Hour),
			},
			{
				WorkOrderID:      uuid.New(),
				LineID:           lineA,
				ClosedAt:         priorClose,
				Merged:           true,
				ExecutionMinutes: 20,
				CostCents:        200,
			},
		}

		got := aggregateLineMetrics(rows, now)
		require.Len(t, got, 1)
		assert.Equal(t, 50.0, got[0].SuccessRatePct)
		assert.Equal(t, int32(1), got[0].MergedCount)
		assert.Equal(t, int32(2), got[0].TotalClosedCount)
		require.NotNil(t, got[0].DurationMinutes)
		assert.InDelta(t, 60.0, *got[0].DurationMinutes, 0.01)
		require.NotNil(t, got[0].CostPerSuccessUsd)
		assert.InDelta(t, 6.4, *got[0].CostPerSuccessUsd, 0.001)
		assert.InDelta(t, 1.0/30.0, got[0].ThroughputPerDay, 0.0001)
		require.NotNil(t, got[0].SuccessDeltaPts)
		assert.Equal(t, -50.0, *got[0].SuccessDeltaPts)
		require.NotNil(t, got[0].DurationDeltaMinutes)
		assert.InDelta(t, 40.0, *got[0].DurationDeltaMinutes, 0.01)
		require.NotNil(t, got[0].CostDeltaUsd)
		assert.InDelta(t, 4.4, *got[0].CostDeltaUsd, 0.001)
	})

	t.Run("omits cost when summed cents are zero", func(t *testing.T) {
		rows := []models.ClosedWorkOrderMetricRow{
			{
				WorkOrderID:      uuid.New(),
				LineID:           lineA,
				ClosedAt:         now.Add(-time.Hour),
				Merged:           true,
				ExecutionMinutes: 60,
			},
		}
		got := aggregateLineMetrics(rows, now)
		require.Len(t, got, 1)
		assert.Nil(t, got[0].CostPerSuccessUsd)
		assert.Nil(t, got[0].CostDeltaUsd)
	})

	t.Run("fills thirty daily buckets and median duration", func(t *testing.T) {
		early := currentStart.Add(2 * time.Hour)
		late := currentEnd.Add(-2 * time.Hour)
		rows := []models.ClosedWorkOrderMetricRow{
			{
				WorkOrderID:      uuid.New(),
				LineID:           lineA,
				ClosedAt:         early,
				Merged:           true,
				ExecutionMinutes: 60,
				CostCents:        100,
			},
			{
				WorkOrderID:      uuid.New(),
				LineID:           lineA,
				ClosedAt:         late,
				Merged:           true,
				ExecutionMinutes: 60,
				CostCents:        100,
			},
			{
				WorkOrderID: uuid.New(),
				LineID:      lineB,
				ClosedAt:    late,
			},
		}

		got := aggregateLineMetrics(rows, now)
		require.Len(t, got, 2)
		assert.Equal(t, lineA, got[0].LineID)
		assert.Equal(t, lineB, got[1].LineID)
		require.Len(t, got[0].SuccessTrendPct, 30)
		require.Len(t, got[0].ThroughputTrend, 30)
		assert.Equal(t, int32(1), got[0].ThroughputTrend[0])
		assert.Equal(t, int32(1), got[0].ThroughputTrend[29])
		assert.Equal(t, 100.0, got[0].SuccessTrendPct[0])
		assert.Equal(t, 100.0, got[0].SuccessTrendPct[29])
		require.NotNil(t, got[0].DurationMinutes)
		assert.InDelta(t, 60.0, *got[0].DurationMinutes, 0.01)
		assert.Equal(t, float64(0), got[1].SuccessRatePct)
		assert.Equal(t, int32(0), got[1].ThroughputTrend[29])
		assert.Equal(t, int32(0), got[1].MergedCount)
	})
}
