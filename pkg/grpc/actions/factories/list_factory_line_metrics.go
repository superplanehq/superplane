package factories

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListFactoryLineMetrics(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryLineMetricsRequest,
) (*pb.ListFactoryLineMetricsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory line metrics")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory line metrics")
	}

	db := database.DB(ctx)
	factoryModel, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory line metrics")
	}

	rows, err := models.ListFactoryLineMetrics(db, factoryModel.OrganizationID, factoryModel.ID, time.Now())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory line metrics")
	}

	return &pb.ListFactoryLineMetricsResponse{
		Lines: serializeFactoryLineMetrics(rows),
	}, nil
}

func serializeFactoryLineMetrics(rows []models.FactoryLineMetrics) []*pb.FactoryLineMetricsEntry {
	result := make([]*pb.FactoryLineMetricsEntry, len(rows))
	for i, row := range rows {
		entry := &pb.FactoryLineMetricsEntry{LineId: row.LineID.String()}
		if row.Present {
			entry.Metrics = serializeFactoryLineMetricsValue(row)
		}
		result[i] = entry
	}
	return result
}

func serializeFactoryLineMetricsValue(row models.FactoryLineMetrics) *pb.FactoryLineMetrics {
	trend := make([]int32, len(row.ThroughputTrend))
	for i, value := range row.ThroughputTrend {
		trend[i] = int32(value)
	}
	return &pb.FactoryLineMetrics{
		SuccessRatePct:     row.SuccessRatePct,
		MergedCount:        int32(row.MergedCount),
		TotalClosedCount:   int32(row.TotalClosedCount),
		ReworkPerWorkOrder: row.ReworkPerWorkOrder,
		CostPerSuccessUsd:  row.CostPerSuccessUsd,
		SuccessTrendPct:    row.SuccessTrendPct,
		SuccessDeltaPts:    row.SuccessDeltaPts,
		ReworkDelta:        row.ReworkDelta,
		CostDeltaUsd:       row.CostDeltaUsd,
		ThroughputPerDay:   row.ThroughputPerDay,
		ThroughputTrend:    trend,
	}
}
