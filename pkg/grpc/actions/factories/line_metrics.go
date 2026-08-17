package factories

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

// lineMetricsTrendBuckets is the number of buckets the trailing window is
// sliced into for the sparkline/bar trend arrays. Bucketing (rather than one
// bucket per day) keeps low-volume lines from showing a jagged/empty-day
// trend — see docs/design/factory.md, "Line metrics".
const lineMetricsTrendBuckets = 10

// aggregateLineMetrics groups closed-work-order rows by line and computes
// every metric formula documented in docs/design/factory.md. It is pure and
// deterministic (given `now`), so it's unit-testable without a database.
//
// `rows` is expected to cover the full `[now - 2*windowDays, now]` range (the
// "current" and "prior" windows); rows outside that range are ignored.
// Lines with no rows in the current window are omitted from the result —
// the frontend renders dashes for a missing line id.
func aggregateLineMetrics(rows []models.ClosedWorkOrderMetricsRow, now time.Time, windowDays int) map[uuid.UUID]*pb.LineMetrics {
	if windowDays <= 0 {
		windowDays = 30
	}

	currentStart := now.AddDate(0, 0, -windowDays)
	priorStart := now.AddDate(0, 0, -2*windowDays)

	rowsByLine := map[uuid.UUID][]models.ClosedWorkOrderMetricsRow{}
	for _, row := range rows {
		if row.ClosedAt.Before(priorStart) || row.ClosedAt.After(now) {
			continue
		}
		rowsByLine[row.LineID] = append(rowsByLine[row.LineID], row)
	}

	result := make(map[uuid.UUID]*pb.LineMetrics, len(rowsByLine))
	for lineID, lineRows := range rowsByLine {
		var current, prior []models.ClosedWorkOrderMetricsRow
		for _, row := range lineRows {
			if !row.ClosedAt.Before(currentStart) {
				current = append(current, row)
			} else {
				prior = append(prior, row)
			}
		}

		if len(current) == 0 {
			continue
		}

		currentSummary := summarizeWindow(current)
		priorSummary := summarizeWindow(prior)

		metrics := &pb.LineMetrics{
			LineId:              lineID.String(),
			SuccessRatePct:      currentSummary.successRatePct,
			MergedCount:         int64(currentSummary.mergedCount),
			TotalClosedCount:    int64(currentSummary.totalCount),
			ReworkPerWorkOrder:  currentSummary.reworkPerWorkOrder,
			CostPerSuccessCents: currentSummary.costPerSuccessCents,
			ThroughputPerDay:    float64(currentSummary.mergedCount) / float64(windowDays),
			SuccessTrendPct:     buildSuccessTrend(current, currentStart, now),
			ThroughputTrend:     buildThroughputTrend(current, currentStart, now),
			SuccessDeltaPts:     windowDelta(priorSummary.totalCount, currentSummary.successRatePct, priorSummary.successRatePct),
			ReworkDelta:         windowDelta(priorSummary.totalCount, currentSummary.reworkPerWorkOrder, priorSummary.reworkPerWorkOrder),
			CostDeltaCents:      int64(windowDelta(priorSummary.totalCount, float64(currentSummary.costPerSuccessCents), float64(priorSummary.costPerSuccessCents))),
		}

		result[lineID] = metrics
	}

	return result
}

type windowSummary struct {
	totalCount          int
	mergedCount         int
	successRatePct      float64
	reworkPerWorkOrder  float64
	costPerSuccessCents int64
}

// summarizeWindow computes the point-in-time metrics for one window (current
// or prior) of a single line's closed work orders.
func summarizeWindow(rows []models.ClosedWorkOrderMetricsRow) windowSummary {
	summary := windowSummary{totalCount: len(rows)}
	if summary.totalCount == 0 {
		return summary
	}

	var totalCost int64
	var totalRework int
	for _, row := range rows {
		if row.Result == models.FactoryWorkOrderResultCompleted {
			summary.mergedCount++
		}
		totalCost += row.CostCents
		totalRework += row.ReworkCount
	}

	summary.successRatePct = float64(summary.mergedCount) / float64(summary.totalCount) * 100
	summary.reworkPerWorkOrder = float64(totalRework) / float64(summary.totalCount)
	if summary.mergedCount > 0 {
		summary.costPerSuccessCents = totalCost / int64(summary.mergedCount)
	}

	return summary
}

// windowDelta is `current - prior`, unless the prior window has no closed
// work orders at all — in that case there's no baseline to compare against,
// so the delta is reported as 0 rather than a swing from zero.
func windowDelta(priorTotalCount int, current, prior float64) float64 {
	if priorTotalCount == 0 {
		return 0
	}
	return current - prior
}

// bucketBounds splits [start, end] into lineMetricsTrendBuckets equal-width
// buckets and returns the (exclusive) upper bound of each one, so a row can
// be assigned to a bucket by finding the first bound it's before.
func bucketBounds(start, end time.Time) []time.Time {
	numBuckets := lineMetricsTrendBuckets
	span := end.Sub(start)
	bucketSpan := span / time.Duration(numBuckets)

	bounds := make([]time.Time, numBuckets)
	for i := 0; i < numBuckets; i++ {
		if i == numBuckets-1 {
			bounds[i] = end
			continue
		}
		bounds[i] = start.Add(bucketSpan * time.Duration(i+1))
	}
	return bounds
}

func bucketIndex(bounds []time.Time, at time.Time) int {
	for i, bound := range bounds {
		if at.Before(bound) || i == len(bounds)-1 {
			return i
		}
	}
	return len(bounds) - 1
}

// buildSuccessTrend buckets the current window and reports the success rate
// per bucket. An empty bucket carries forward the previous bucket's rate
// (there was no signal in that bucket, so "nothing changed" is a better
// default than a misleading drop to 0%); the first bucket defaults to 0 if
// it has no data.
func buildSuccessTrend(rows []models.ClosedWorkOrderMetricsRow, start, end time.Time) []float64 {
	bounds := bucketBounds(start, end)
	totals := make([]int, len(bounds))
	merged := make([]int, len(bounds))

	for _, row := range rows {
		idx := bucketIndex(bounds, row.ClosedAt)
		totals[idx]++
		if row.Result == models.FactoryWorkOrderResultCompleted {
			merged[idx]++
		}
	}

	trend := make([]float64, len(bounds))
	previous := 0.0
	for i := range bounds {
		if totals[i] == 0 {
			trend[i] = previous
			continue
		}
		trend[i] = float64(merged[i]) / float64(totals[i]) * 100
		previous = trend[i]
	}

	return trend
}

// buildThroughputTrend buckets the current window and reports the raw count
// of merged (completed) work orders per bucket — a sum, not a rate, so an
// empty bucket is legitimately 0.
func buildThroughputTrend(rows []models.ClosedWorkOrderMetricsRow, start, end time.Time) []float64 {
	bounds := bucketBounds(start, end)
	trend := make([]float64, len(bounds))

	for _, row := range rows {
		if row.Result != models.FactoryWorkOrderResultCompleted {
			continue
		}
		idx := bucketIndex(bounds, row.ClosedAt)
		trend[idx]++
	}

	return trend
}
