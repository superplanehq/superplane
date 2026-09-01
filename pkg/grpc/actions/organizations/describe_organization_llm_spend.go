package organizations

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
)

const (
	llmSpendPeriodDaysDefault = 30
	llmSpendPeriodDaysMax     = 90
)

func DescribeOrganizationLLMSpend(
	ctx context.Context,
	orgID string,
	req *pb.DescribeOrganizationLLMSpendRequest,
) (*pb.DescribeOrganizationLLMSpendResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	period := clampLLMSpendPeriodDays(int(req.GetPeriodDays()))
	since := time.Now().AddDate(0, 0, -period)

	totals, byModel, err := models.SummarizeUsage(database.DB(ctx), models.UsageReportFilter{
		OrganizationID: organizationID,
		Since:          since,
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization LLM spend")
	}

	credit, err := models.DescribeOrganizationLLMCredit(database.DB(ctx), organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization LLM credit")
	}

	billingEnabled, hasCustomer := billingState(ctx, organizationID)

	return &pb.DescribeOrganizationLLMSpendResponse{
		TotalTokens:            totals.TotalTokens,
		TotalCostCents:         totals.CostCents(),
		PeriodDays:             int32(period),
		ByModel:                serializeLLMSpendByModel(byModel),
		RemainingCreditCents:   pricebook.MicrosToCents(credit.RemainingMicros),
		GrantTotalCents:        pricebook.MicrosToCents(credit.GrantMicros),
		HostedBilledCents:      pricebook.MicrosToCents(credit.BilledMicros),
		RemainingCreditWarning: credit.Warning,
		BillingEnabled:         billingEnabled,
		HasBillingCustomer:     hasCustomer,
		SuperplaneGrantCents:   pricebook.MicrosToCents(credit.SuperPlaneGrantMicros),
		PurchasedCreditCents:   pricebook.MicrosToCents(credit.PurchasedCreditMicros),
		Invoices:               listHostedCreditInvoices(ctx, organizationID, billingEnabled, hasCustomer),
	}, nil
}

func clampLLMSpendPeriodDays(period int) int {
	if period <= 0 {
		return llmSpendPeriodDaysDefault
	}
	if period > llmSpendPeriodDaysMax {
		return llmSpendPeriodDaysMax
	}
	return period
}

func serializeLLMSpendByModel(rows []models.UsageByModel) []*pb.LLMSpendByModel {
	out := make([]*pb.LLMSpendByModel, 0, len(rows))
	for _, row := range rows {
		out = append(out, &pb.LLMSpendByModel{
			Provider:    row.Provider,
			Model:       row.Model,
			TotalTokens: row.TotalTokens,
			CostCents:   row.CostCents(),
		})
	}
	return out
}

func listHostedCreditInvoices(
	ctx context.Context,
	organizationID uuid.UUID,
	billingEnabled bool,
	hasCustomer bool,
) []*pb.HostedCreditInvoice {
	if !billingEnabled || !hasCustomer || !polar.Configured() {
		return nil
	}

	orders, err := polar.NewClientFromEnv().ListOrders(ctx, organizationID.String())
	if err != nil {
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
