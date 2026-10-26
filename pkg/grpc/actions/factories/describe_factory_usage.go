package factories

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/billing"
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

	period := clampUsagePeriodDays(int(req.GetPeriodDays()))
	since := time.Now().AddDate(0, 0, -period)

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}
	factoryID := factory.ID

	totals, byModel, err := billing.SummarizeUsage(db, billing.UsageReportFilter{
		OrganizationID: orgID,
		FactoryID:      &factoryID,
		Since:          since,
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	computeTotals, byMachine, err := billing.SummarizeComputeUsage(db, billing.UsageReportFilter{
		OrganizationID: orgID,
		FactoryID:      &factoryID,
		Since:          since,
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	credit, err := billing.DescribeCredit(db, orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	budget, err := billing.DescribeFactoryBudget(db, factory.OrganizationID, factory.ID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory usage")
	}

	ledger := totals.Add(computeTotals)
	resp := &pb.DescribeFactoryUsageResponse{
		TotalTokens:                   ledger.TotalTokens,
		TotalCostCents:                ledger.CostCents(),
		PeriodDays:                    int32(period),
		ByModel:                       serializeUsageByModel(byModel),
		RemainingCreditCents:          billing.MicrosToCents(credit.RemainingMicros),
		GrantTotalCents:               billing.MicrosToCents(credit.GrantMicros),
		HostedBilledCents:             billing.MicrosToCents(credit.BilledMicros),
		RemainingCreditWarning:        credit.Warning,
		FactoryHostedBilledCents:      billing.MicrosToCents(budget.BilledMicros),
		FactoryRemainingCreditCents:   billing.MicrosToCents(budget.RemainingMicros),
		FactoryRemainingCreditWarning: budget.Warning,
		TotalDurationSeconds:          ledger.DurationSeconds,
		ByMachineType:                 serializeUsageByMachineType(byMachine),
		HostedCreditAvailable:         !billing.Enabled() || credit.RemainingMicros > 0,
		FactoryHostedBudgetAvailable:  !billing.Enabled() || !budget.Capped || budget.RemainingMicros > 0,
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
