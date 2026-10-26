package organizations

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

// Service implements the billing capability of the enterprise registry.
type Service struct{}

func New() *Service { return &Service{} }

func (*Service) DescribeOrganizationSpendingReport(ctx context.Context,
	orgID string,
	req *pb.DescribeOrganizationSpendingReportRequest) (*pb.DescribeOrganizationSpendingReportResponse, error) {
	return DescribeOrganizationSpendingReport(ctx, orgID, req)
}

func (*Service) ListHostedCreditProducts(ctx context.Context,
	orgID string,
	req *pb.ListHostedCreditProductsRequest) (*pb.ListHostedCreditProductsResponse, error) {
	return ListHostedCreditProducts(ctx, orgID, req)
}

func (*Service) QuoteHostedCreditCheckout(ctx context.Context,
	orgID string,
	req *pb.QuoteHostedCreditCheckoutRequest) (*pb.QuoteHostedCreditCheckoutResponse, error) {
	return QuoteHostedCreditCheckout(ctx, orgID, req)
}

func (*Service) CreateHostedCreditCheckout(ctx context.Context,
	orgID string,
	req *pb.CreateHostedCreditCheckoutRequest,
	accountID string,
	baseURL string) (*pb.CreateHostedCreditCheckoutResponse, error) {
	return CreateHostedCreditCheckout(ctx, orgID, req, accountID, baseURL)
}

func (*Service) CreateBillingPortalSession(ctx context.Context,
	orgID string,
	req *pb.CreateBillingPortalSessionRequest) (*pb.CreateBillingPortalSessionResponse, error) {
	return CreateBillingPortalSession(ctx, orgID, req)
}

func (*Service) ListOrganizationCreditGrants(ctx context.Context,
	orgID string,
	req *pb.ListOrganizationCreditGrantsRequest) (*pb.ListOrganizationCreditGrantsResponse, error) {
	return ListOrganizationCreditGrants(ctx, orgID, req)
}

func (*Service) DescribeOrganizationBilling(ctx context.Context,
	orgID string,
	req *pb.DescribeOrganizationBillingRequest) (*pb.DescribeOrganizationBillingResponse, error) {
	return DescribeOrganizationBilling(ctx, orgID, req)
}

func (*Service) CreateBusinessCheckout(ctx context.Context,
	orgID string,
	req *pb.CreateBusinessCheckoutRequest,
	accountID string,
	baseURL string) (*pb.CreateBusinessCheckoutResponse, error) {
	return CreateBusinessCheckout(ctx, orgID, req, accountID, baseURL)
}

func (*Service) SelectBillingPlan(ctx context.Context, orgID string, req *pb.SelectBillingPlanRequest, accountID, baseURL string) (*pb.CreateBusinessCheckoutResponse, error) {
	return SelectBillingPlan(ctx, orgID, req, accountID, baseURL)
}

func (*Service) CreatePaymentMethodSession(ctx context.Context, orgID string, req *pb.CreatePaymentMethodSessionRequest, accountID string) (*pb.CreateBillingPortalSessionResponse, error) {
	return CreatePaymentMethodSession(ctx, orgID, req, accountID)
}

func (*Service) PreviewSeatPurchase(ctx context.Context, orgID string, req *pb.PreviewSeatPurchaseRequest) (*pb.PreviewSeatPurchaseResponse, error) {
	return PreviewSeatPurchase(ctx, orgID, req)
}

func (*Service) PurchaseSeats(ctx context.Context, orgID string, req *pb.PurchaseSeatsRequest, userID string) (*pb.PurchaseSeatsResponse, error) {
	return PurchaseSeats(ctx, orgID, req, userID)
}

func (*Service) SyncOrganizationBilling(ctx context.Context,
	orgID string,
	req *pb.SyncOrganizationBillingRequest) (*pb.DescribeOrganizationBillingResponse, error) {
	return SyncOrganizationBilling(ctx, orgID, req)
}

func (*Service) CancelOrganizationSubscription(ctx context.Context,
	orgID string,
	req *pb.CancelOrganizationSubscriptionRequest) (*pb.DescribeOrganizationBillingResponse, error) {
	return CancelOrganizationSubscription(ctx, orgID, req)
}

func (*Service) ResumeOrganizationSubscription(ctx context.Context,
	orgID string,
	req *pb.ResumeOrganizationSubscriptionRequest) (*pb.DescribeOrganizationBillingResponse, error) {
	return ResumeOrganizationSubscription(ctx, orgID, req)
}

func (*Service) DescribeOrganizationWorkspaceUsage(ctx context.Context, orgID string, req *pb.DescribeOrganizationWorkspaceUsageRequest) (*pb.DescribeOrganizationWorkspaceUsageResponse, error) {
	return DescribeOrganizationWorkspaceUsage(ctx, orgID, req)
}
