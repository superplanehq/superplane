package factories

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

const defaultLineMetricsWindowDays = 30

func ListLineMetrics(ctx context.Context, organizationID string, req *pb.ListLineMetricsRequest) (*pb.ListLineMetricsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line metrics")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line metrics")
	}

	windowDays := defaultLineMetricsWindowDays
	if req.WindowDays != nil {
		windowDays = int(req.GetWindowDays())
	}
	if windowDays <= 0 {
		return nil, factoryErrorToStatus(invalidArgument("window_days must be a positive number of days"), "failed to list line metrics")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line metrics")
	}

	// One query covers both the current and prior windows, so deltas can be
	// computed without a second round trip.
	now := time.Now()
	since := now.AddDate(0, 0, -2*windowDays)
	rows, err := factory.ListClosedWorkOrderMetricsRows(db, since)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line metrics")
	}

	metricsByLine := aggregateLineMetrics(rows, now, windowDays)

	response := &pb.ListLineMetricsResponse{
		Metrics: make([]*pb.LineMetrics, 0, len(metricsByLine)),
	}
	for _, metrics := range metricsByLine {
		response.Metrics = append(response.Metrics, metrics)
	}

	return response, nil
}
