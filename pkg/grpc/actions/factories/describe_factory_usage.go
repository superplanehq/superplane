package factories

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
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
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
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

	computeTotals, byMachine, err := models.SummarizeComputeUsage(db, models.UsageReportFilter{
		OrganizationID: orgID,
		FactoryID:      &factoryID,
		Since:          since,
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	credit, err := models.DescribeOrganizationLLMCredit(db, orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	budget, err := models.DescribeFactoryHostedBudget(db, factory)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	ledger := totals.Add(computeTotals)
	resp := &pb.DescribeFactoryUsageResponse{
		TotalTokens:                   ledger.TotalTokens,
		TotalCostCents:                ledger.CostCents(),
		PeriodDays:                    int32(period),
		ByModel:                       serializeUsageByModel(byModel),
		RemainingCreditCents:          pricebook.MicrosToCents(credit.RemainingMicros),
		GrantTotalCents:               pricebook.MicrosToCents(credit.GrantMicros),
		HostedBilledCents:             pricebook.MicrosToCents(credit.BilledMicros),
		RemainingCreditWarning:        credit.Warning,
		FactoryHostedBilledCents:      pricebook.MicrosToCents(budget.BilledMicros),
		FactoryRemainingCreditCents:   pricebook.MicrosToCents(budget.RemainingMicros),
		FactoryRemainingCreditWarning: budget.Warning,
		TotalDurationSeconds:          ledger.DurationSeconds,
		ByMachineType:                 serializeUsageByMachineType(byMachine),
	}
	if budget.BudgetCents != nil {
		resp.HostedSpendBudgetCents = budget.BudgetCents
	}
	return resp, nil
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

func serializeUsageByMachineType(rows []models.UsageByMachineType) []*pb.UsageByMachineType {
	out := make([]*pb.UsageByMachineType, 0, len(rows))
	for _, row := range rows {
		out = append(out, &pb.UsageByMachineType{
			MachineType:     row.MachineType,
			DurationSeconds: row.DurationSeconds,
			CostCents:       row.CostCents(),
		})
	}
	return out
}
