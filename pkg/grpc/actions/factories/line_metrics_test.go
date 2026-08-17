package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/models"
)

func TestAggregateLineMetrics_EmptyInput(t *testing.T) {
	metrics := aggregateLineMetrics(nil, time.Now(), 30)
	assert.Empty(t, metrics)
}

func TestAggregateLineMetrics_SingleCompletedOrder(t *testing.T) {
	now := time.Now()
	lineID := uuid.New()

	metrics := aggregateLineMetrics([]models.ClosedWorkOrderMetricsRow{
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultCompleted, ClosedAt: now.AddDate(0, 0, -1), CostCents: 1000},
	}, now, 30)

	require.Contains(t, metrics, lineID)
	line := metrics[lineID]
	assert.Equal(t, 100.0, line.SuccessRatePct)
	assert.EqualValues(t, 1, line.MergedCount)
	assert.EqualValues(t, 1, line.TotalClosedCount)
	assert.EqualValues(t, 1000, line.CostPerSuccessCents)
}

func TestAggregateLineMetrics_MixOfResults_DenominatorIsAllClosed(t *testing.T) {
	now := time.Now()
	lineID := uuid.New()

	rows := []models.ClosedWorkOrderMetricsRow{
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultCompleted, ClosedAt: now.AddDate(0, 0, -1)},
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultRejected, ClosedAt: now.AddDate(0, 0, -2)},
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultFailed, ClosedAt: now.AddDate(0, 0, -3)},
	}

	metrics := aggregateLineMetrics(rows, now, 30)
	line := metrics[lineID]
	require.NotNil(t, line)

	assert.EqualValues(t, 1, line.MergedCount)
	assert.EqualValues(t, 3, line.TotalClosedCount)
	assert.InDelta(t, 100.0/3.0, line.SuccessRatePct, 0.001)
}

func TestAggregateLineMetrics_PriorWindowEmpty_DeltasAreZero(t *testing.T) {
	now := time.Now()
	lineID := uuid.New()

	// Only current-window rows; nothing in days 31-60 back.
	rows := []models.ClosedWorkOrderMetricsRow{
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultCompleted, ClosedAt: now.AddDate(0, 0, -1), ReworkCount: 2, CostCents: 500},
	}

	metrics := aggregateLineMetrics(rows, now, 30)
	line := metrics[lineID]
	require.NotNil(t, line)

	assert.Zero(t, line.SuccessDeltaPts)
	assert.Zero(t, line.ReworkDelta)
	assert.Zero(t, line.CostDeltaCents)
}

func TestAggregateLineMetrics_DeltasComparePriorWindow(t *testing.T) {
	now := time.Now()
	lineID := uuid.New()

	rows := []models.ClosedWorkOrderMetricsRow{
		// Current window (last 30 days): 100% success.
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultCompleted, ClosedAt: now.AddDate(0, 0, -1)},
		// Prior window (31-60 days back): 0% success.
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultRejected, ClosedAt: now.AddDate(0, 0, -31)},
	}

	metrics := aggregateLineMetrics(rows, now, 30)
	line := metrics[lineID]
	require.NotNil(t, line)

	assert.Equal(t, 100.0, line.SuccessDeltaPts)
}

func TestAggregateLineMetrics_CostPerSuccessIncludesFailedOrderCost(t *testing.T) {
	now := time.Now()
	lineID := uuid.New()

	rows := []models.ClosedWorkOrderMetricsRow{
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultCompleted, ClosedAt: now.AddDate(0, 0, -1), CostCents: 100},
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultFailed, ClosedAt: now.AddDate(0, 0, -2), CostCents: 300},
	}

	metrics := aggregateLineMetrics(rows, now, 30)
	line := metrics[lineID]
	require.NotNil(t, line)

	// Denominator is mergedCount (1), not totalClosedCount (2), but the
	// numerator includes the failed order's cost too.
	assert.EqualValues(t, 1, line.MergedCount)
	assert.EqualValues(t, 400, line.CostPerSuccessCents)
}

func TestAggregateLineMetrics_ZeroMergedCount_CostPerSuccessIsZero(t *testing.T) {
	now := time.Now()
	lineID := uuid.New()

	rows := []models.ClosedWorkOrderMetricsRow{
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultFailed, ClosedAt: now.AddDate(0, 0, -1), CostCents: 300},
	}

	metrics := aggregateLineMetrics(rows, now, 30)
	line := metrics[lineID]
	require.NotNil(t, line)

	assert.Zero(t, line.MergedCount)
	assert.Zero(t, line.CostPerSuccessCents)
}

func TestAggregateLineMetrics_ThroughputPerDay(t *testing.T) {
	now := time.Now()
	lineID := uuid.New()

	rows := make([]models.ClosedWorkOrderMetricsRow, 0, 6)
	for i := 0; i < 6; i++ {
		rows = append(rows, models.ClosedWorkOrderMetricsRow{
			WorkOrderID: uuid.New(),
			LineID:      lineID,
			Result:      models.FactoryWorkOrderResultCompleted,
			ClosedAt:    now.AddDate(0, 0, -1),
		})
	}

	metrics := aggregateLineMetrics(rows, now, 30)
	line := metrics[lineID]
	require.NotNil(t, line)

	assert.Equal(t, 0.2, line.ThroughputPerDay) // 6 merged / 30 days
}

func TestAggregateLineMetrics_LinesWithNoCurrentWindowRowsAreOmitted(t *testing.T) {
	now := time.Now()
	lineID := uuid.New()

	// Only in the prior window: no current-window activity at all.
	rows := []models.ClosedWorkOrderMetricsRow{
		{WorkOrderID: uuid.New(), LineID: lineID, Result: models.FactoryWorkOrderResultCompleted, ClosedAt: now.AddDate(0, 0, -45)},
	}

	metrics := aggregateLineMetrics(rows, now, 30)
	assert.NotContains(t, metrics, lineID)
}

func TestAggregateLineMetrics_LinesDoNotLeakIntoEachOther(t *testing.T) {
	now := time.Now()
	lineA, lineB := uuid.New(), uuid.New()

	rows := []models.ClosedWorkOrderMetricsRow{
		{WorkOrderID: uuid.New(), LineID: lineA, Result: models.FactoryWorkOrderResultCompleted, ClosedAt: now.AddDate(0, 0, -1)},
		{WorkOrderID: uuid.New(), LineID: lineB, Result: models.FactoryWorkOrderResultRejected, ClosedAt: now.AddDate(0, 0, -1)},
	}

	metrics := aggregateLineMetrics(rows, now, 30)
	require.Contains(t, metrics, lineA)
	require.Contains(t, metrics, lineB)
	assert.EqualValues(t, 1, metrics[lineA].MergedCount)
	assert.EqualValues(t, 0, metrics[lineB].MergedCount)
}

func TestBuildSuccessTrend_BucketCountAndCarryForwardOnEmptyBucket(t *testing.T) {
	start := time.Now().AddDate(0, 0, -30)
	end := time.Now()

	// One completed order early (bucket 0), nothing else: every later bucket
	// should carry forward bucket 0's 100% rate rather than reading as 0%.
	rows := []models.ClosedWorkOrderMetricsRow{
		{LineID: uuid.New(), Result: models.FactoryWorkOrderResultCompleted, ClosedAt: start.Add(time.Hour)},
	}

	trend := buildSuccessTrend(rows, start, end)
	require.Len(t, trend, lineMetricsTrendBuckets)
	for _, value := range trend {
		assert.Equal(t, 100.0, value)
	}
}

func TestBuildSuccessTrend_FirstBucketEmptyDefaultsToZero(t *testing.T) {
	start := time.Now().AddDate(0, 0, -30)
	end := time.Now()

	trend := buildSuccessTrend(nil, start, end)
	require.Len(t, trend, lineMetricsTrendBuckets)
	for _, value := range trend {
		assert.Zero(t, value)
	}
}

func TestBuildThroughputTrend_IsASumNotACarryForward(t *testing.T) {
	start := time.Now().AddDate(0, 0, -30)
	end := time.Now()

	rows := []models.ClosedWorkOrderMetricsRow{
		{LineID: uuid.New(), Result: models.FactoryWorkOrderResultCompleted, ClosedAt: start.Add(time.Hour)},
		{LineID: uuid.New(), Result: models.FactoryWorkOrderResultCompleted, ClosedAt: start.Add(time.Hour)},
		{LineID: uuid.New(), Result: models.FactoryWorkOrderResultRejected, ClosedAt: start.Add(time.Hour)},
	}

	trend := buildThroughputTrend(rows, start, end)
	require.Len(t, trend, lineMetricsTrendBuckets)
	assert.Equal(t, 2.0, trend[0])
	for _, value := range trend[1:] {
		assert.Zero(t, value, "throughput buckets are sums, not carried forward")
	}
}
