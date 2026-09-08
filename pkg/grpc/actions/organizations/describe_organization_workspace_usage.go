package organizations

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
)

const (
	workspaceUsagePeriodDaysDefault = 30
	workspaceUsagePeriodDaysMax     = 90
)

func DescribeOrganizationWorkspaceUsage(
	ctx context.Context,
	orgID string,
	req *pb.DescribeOrganizationWorkspaceUsageRequest,
) (*pb.DescribeOrganizationWorkspaceUsageResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	period := clampWorkspaceUsagePeriodDays(int(req.GetPeriodDays()))
	since := time.Now().AddDate(0, 0, -period)

	db := database.DB(ctx)
	totals, byModel, err := models.SummarizeUsage(db, models.UsageReportFilter{
		OrganizationID: organizationID,
		Since:          since,
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization workspace usage")
	}

	computeTotals, byMachine, err := models.SummarizeComputeUsage(db, models.UsageReportFilter{
		OrganizationID: organizationID,
		Since:          since,
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization workspace usage")
	}

	billingEnabled, hasCustomer := billingState(ctx, organizationID)
	invoices := listHostedCreditInvoices(ctx, db, organizationID, billingEnabled, hasCustomer)

	credit, err := models.DescribeOrganizationLLMCredit(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization workspace usage")
	}

	ledger := totals.Add(computeTotals)
	defaultModel, err := models.GetInstallationDefaultHostedLLMModel(db)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization workspace usage")
	}

	return &pb.DescribeOrganizationWorkspaceUsageResponse{
		TotalTokens:            ledger.TotalTokens,
		TotalCostCents:         ledger.CostCents(),
		PeriodDays:             int32(period),
		ByModel:                serializeUsageByModel(byModel),
		RemainingCreditCents:   pricebook.MicrosToCents(credit.RemainingMicros),
		GrantTotalCents:        pricebook.MicrosToCents(credit.GrantMicros),
		HostedBilledCents:      pricebook.MicrosToCents(credit.BilledMicros),
		RemainingCreditWarning: credit.Warning,
		BillingEnabled:         billingEnabled,
		HasBillingCustomer:     hasCustomer,
		SuperplaneGrantCents:   pricebook.MicrosToCents(credit.SuperPlaneGrantMicros),
		PurchasedCreditCents:   pricebook.MicrosToCents(credit.PurchasedCreditMicros),
		Invoices:               invoices,
		TotalDurationSeconds:   ledger.DurationSeconds,
		ByMachineType:          serializeUsageByMachineType(byMachine),
		DefaultHostedProvider:  defaultModel.Provider,
		DefaultHostedModel:     defaultModel.Model,
	}, nil
}

func clampWorkspaceUsagePeriodDays(period int) int {
	if period <= 0 {
		return workspaceUsagePeriodDaysDefault
	}
	if period > workspaceUsagePeriodDaysMax {
		return workspaceUsagePeriodDaysMax
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

func listHostedCreditInvoices(
	ctx context.Context,
	db *gorm.DB,
	organizationID uuid.UUID,
	billingEnabled bool,
	hasCustomer bool,
) []*pb.HostedCreditInvoice {
	if !billingEnabled || !hasCustomer || !polar.Configured() {
		return nil
	}

	orders, err := polar.NewClientFromEnv().ListAndReconcileOrders(ctx, db, organizationID)
	if err != nil {
		log.WithError(err).WithField("organization_id", organizationID).Warn("failed to list polar hosted credit orders")
		return nil
	}

	invoices := make([]*pb.HostedCreditInvoice, 0, len(orders))
	for _, order := range orders {
		invoices = append(invoices, &pb.HostedCreditInvoice{
			Id:          order.ID,
			CreatedAt:   order.CreatedAt,
			AmountCents: order.AmountCents,
			Status:      order.Status,
			ProductName: order.ProductName,
		})
	}
	return invoices
}
