package factories

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

const (
	usagePeriodDaysDefault = 30
	usagePeriodDaysMax     = 90
)

func DescribeFactoryUsage(
	ctx context.Context,
	organizationID string,
	req *pb.DescribeFactoryUsageRequest,
) (*pb.DescribeFactoryUsageResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	period := clampUsagePeriodDays(int(req.GetPeriodDays()))
	since := time.Now().AddDate(0, 0, -period)

	db := database.DB(ctx)
	if _, err := models.FindFactory(db, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	totals, byModel, err := models.SummarizeUsage(db, models.UsageReportFilter{
		OrganizationID: orgID,
		FactoryID:      &factoryID,
		Since:          since,
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	return &pb.DescribeFactoryUsageResponse{
		TotalTokens:    totals.TotalTokens,
		TotalCostCents: totals.CostCents(),
		PeriodDays:     int32(period),
		ByModel:        serializeUsageByModel(byModel),
	}, nil
}

func clampUsagePeriodDays(period int) int {
	if period <= 0 {
		return usagePeriodDaysDefault
	}
	if period > usagePeriodDaysMax {
		return usagePeriodDaysMax
	}
	return period
}

func serializeUsageByModel(rows []models.UsageByModel) []*pb.UsageByModel {
	out := make([]*pb.UsageByModel, 0, len(rows))
	for _, row := range rows {
		out = append(out, &pb.UsageByModel{
			Provider:    row.Provider,
			Model:       row.Model,
			TotalTokens: row.TotalTokens,
			CostCents:   row.CostCents(),
		})
	}
	return out
}
