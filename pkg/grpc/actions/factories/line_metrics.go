package factories

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

const lineMetricsWindowDays = 30

type lineMetrics struct {
	LineID               uuid.UUID
	SuccessRatePct       float64
	MergedCount          int32
	TotalClosedCount     int32
	DurationMinutes      *float64
	CostPerSuccessUsd    *float64
	SuccessTrendPct      []float64
	SuccessDeltaPts      *float64
	DurationDeltaMinutes *float64
	CostDeltaUsd         *float64
	ThroughputPerDay     float64
	ThroughputTrend      []int32
}

type lineWindowStats struct {
	closedCount     int
	mergedCount     int
	durationMinutes *float64
	costPerSuccess  *float64
}

func aggregateLineMetrics(rows []models.ClosedWorkOrderMetricRow, now time.Time) []lineMetrics {
	priorStart, currentStart, currentEnd := lineMetricsWindowBounds(now)
	buckets := buildLineMetricsDayBuckets(now)

	byLine := map[uuid.UUID][]models.ClosedWorkOrderMetricRow{}
	for _, row := range rows {
		byLine[row.LineID] = append(byLine[row.LineID], row)
	}

	lineIDs := make([]uuid.UUID, 0, len(byLine))
	for lineID := range byLine {
		lineIDs = append(lineIDs, lineID)
	}
	sort.Slice(lineIDs, func(i, j int) bool {
		return lineIDs[i].String() < lineIDs[j].String()
	})

	result := make([]lineMetrics, 0, len(lineIDs))
	for _, lineID := range lineIDs {
		lineRows := byLine[lineID]
		current := rowsInWindow(lineRows, currentStart, currentEnd)
		if len(current) == 0 {
			continue
		}

		prior := rowsInWindow(lineRows, priorStart, currentStart)
		currentStats := statsForClosedOrders(current)
		priorStats := statsForClosedOrders(prior)

		metrics := lineMetrics{
			LineID:            lineID,
			SuccessRatePct:    currentStats.successRatePct(),
			MergedCount:       int32(currentStats.mergedCount),
			TotalClosedCount:  int32(currentStats.closedCount),
			DurationMinutes:   currentStats.durationMinutes,
			CostPerSuccessUsd: currentStats.costPerSuccess,
			SuccessTrendPct:   successTrend(current, buckets),
			ThroughputPerDay:  float64(currentStats.mergedCount) / float64(lineMetricsWindowDays),
			ThroughputTrend:   throughputTrend(current, buckets),
		}
		if priorStats.closedCount > 0 {
			metrics.SuccessDeltaPts = floatPtr(metrics.SuccessRatePct - priorStats.successRatePct())
		}
		if currentStats.durationMinutes != nil && priorStats.durationMinutes != nil {
			metrics.DurationDeltaMinutes = floatPtr(*currentStats.durationMinutes - *priorStats.durationMinutes)
		}
		if currentStats.costPerSuccess != nil && priorStats.costPerSuccess != nil {
			metrics.CostDeltaUsd = floatPtr(*currentStats.costPerSuccess - *priorStats.costPerSuccess)
		}
		result = append(result, metrics)
	}
	return result
}

func rowsInWindow(rows []models.ClosedWorkOrderMetricRow, from, to time.Time) []models.ClosedWorkOrderMetricRow {
	matched := make([]models.ClosedWorkOrderMetricRow, 0, len(rows))
	for _, row := range rows {
		if !row.ClosedAt.Before(from) && row.ClosedAt.Before(to) {
			matched = append(matched, row)
		}
	}
	return matched
}

func statsForClosedOrders(rows []models.ClosedWorkOrderMetricRow) lineWindowStats {
	stats := lineWindowStats{closedCount: len(rows)}
	if stats.closedCount == 0 {
		return stats
	}

	var costCents int64
	durations := make([]float64, 0, len(rows))
	for _, row := range rows {
		costCents += row.CostCents
		if !row.Merged {
			continue
		}
		stats.mergedCount++
		if minutes, ok := durationMinutes(row); ok {
			durations = append(durations, minutes)
		}
	}
	if len(durations) > 0 {
		stats.durationMinutes = floatPtr(medianFloat(durations))
	}
	if stats.mergedCount > 0 && costCents > 0 {
		stats.costPerSuccess = floatPtr(float64(costCents) / float64(stats.mergedCount) / 100)
	}
	return stats
}

func (s lineWindowStats) successRatePct() float64 {
	if s.closedCount == 0 {
		return 0
	}
	return float64(s.mergedCount) / float64(s.closedCount) * 100
}

func durationMinutes(row models.ClosedWorkOrderMetricRow) (float64, bool) {
	if row.ExecutionMinutes <= 0 {
		return 0, false
	}
	return row.ExecutionMinutes, true
}

func successTrend(rows []models.ClosedWorkOrderMetricRow, buckets []lineMetricsDayBucket) []float64 {
	trend := make([]float64, len(buckets))
	if len(buckets) == 0 {
		return trend
	}
	windowStart := buckets[0].start
	for i, bucket := range buckets {
		window := rowsInWindow(rows, windowStart, bucket.end)
		trend[i] = statsForClosedOrders(window).successRatePct()
	}
	return trend
}

func throughputTrend(rows []models.ClosedWorkOrderMetricRow, buckets []lineMetricsDayBucket) []int32 {
	trend := make([]int32, len(buckets))
	for i, bucket := range buckets {
		for _, row := range rows {
			if !row.Merged {
				continue
			}
			if !row.ClosedAt.Before(bucket.start) && row.ClosedAt.Before(bucket.end) {
				trend[i]++
			}
		}
	}
	return trend
}

func medianFloat(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

type lineMetricsDayBucket struct {
	start time.Time
	end   time.Time
}

func lineMetricsWindowBounds(now time.Time) (priorStart, currentStart, currentEnd time.Time) {
	today := startOfLocalDay(now)
	currentEnd = today.AddDate(0, 0, 1)
	currentStart = today.AddDate(0, 0, -(lineMetricsWindowDays - 1))
	priorStart = currentStart.AddDate(0, 0, -lineMetricsWindowDays)
	return priorStart, currentStart, currentEnd
}

func buildLineMetricsDayBuckets(now time.Time) []lineMetricsDayBucket {
	today := startOfLocalDay(now)
	buckets := make([]lineMetricsDayBucket, lineMetricsWindowDays)
	for i := 0; i < lineMetricsWindowDays; i++ {
		start := today.AddDate(0, 0, -(lineMetricsWindowDays - 1 - i))
		buckets[i] = lineMetricsDayBucket{start: start, end: start.AddDate(0, 0, 1)}
	}
	return buckets
}

func startOfLocalDay(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func floatPtr(value float64) *float64 {
	return &value
}

func loadFactoryLineMetrics(tx *gorm.DB, factoryID uuid.UUID) (map[uuid.UUID]*pb.FactoryLineMetrics, error) {
	now := timeNow()
	priorStart, _, currentEnd := lineMetricsWindowBounds(now)
	rows, err := models.ListClosedWorkOrderMetricRows(tx, factoryID, priorStart, currentEnd)
	if err != nil {
		return nil, err
	}

	aggregated := aggregateLineMetrics(rows, now)
	byLine := make(map[uuid.UUID]*pb.FactoryLineMetrics, len(aggregated))
	for _, item := range aggregated {
		byLine[item.LineID] = serializeLineMetrics(item)
	}
	return byLine, nil
}

func serializeLineMetrics(item lineMetrics) *pb.FactoryLineMetrics {
	return &pb.FactoryLineMetrics{
		SuccessRatePct:       item.SuccessRatePct,
		MergedCount:          item.MergedCount,
		TotalClosedCount:     item.TotalClosedCount,
		DurationMinutes:      item.DurationMinutes,
		CostPerSuccessUsd:    item.CostPerSuccessUsd,
		SuccessTrendPct:      item.SuccessTrendPct,
		SuccessDeltaPts:      item.SuccessDeltaPts,
		DurationDeltaMinutes: item.DurationDeltaMinutes,
		CostDeltaUsd:         item.CostDeltaUsd,
		ThroughputPerDay:     item.ThroughputPerDay,
		ThroughputTrend:      item.ThroughputTrend,
	}
}

var timeNow = func() time.Time {
	return time.Now().In(time.Local)
}
