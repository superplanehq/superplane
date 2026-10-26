package enterprise

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

// BillingOperations implements organization billing RPCs. Billing source is covered
// by the Enterprise license; operations do not require a runtime license.
type BillingOperations interface {
	DescribeOrganizationWorkspaceUsage(ctx context.Context, orgID string, req *pb.DescribeOrganizationWorkspaceUsageRequest) (*pb.DescribeOrganizationWorkspaceUsageResponse, error)
	DescribeOrganizationSpendingReport(ctx context.Context,
		orgID string,
		req *pb.DescribeOrganizationSpendingReportRequest) (*pb.DescribeOrganizationSpendingReportResponse, error)
	ListHostedCreditProducts(ctx context.Context,
		orgID string,
		req *pb.ListHostedCreditProductsRequest) (*pb.ListHostedCreditProductsResponse, error)
	QuoteHostedCreditCheckout(ctx context.Context,
		orgID string,
		req *pb.QuoteHostedCreditCheckoutRequest) (*pb.QuoteHostedCreditCheckoutResponse, error)
	CreateHostedCreditCheckout(ctx context.Context,
		orgID string,
		req *pb.CreateHostedCreditCheckoutRequest,
		accountID string,
		baseURL string) (*pb.CreateHostedCreditCheckoutResponse, error)
	CreateBillingPortalSession(ctx context.Context,
		orgID string,
		req *pb.CreateBillingPortalSessionRequest) (*pb.CreateBillingPortalSessionResponse, error)
	ListOrganizationCreditGrants(ctx context.Context,
		orgID string,
		req *pb.ListOrganizationCreditGrantsRequest) (*pb.ListOrganizationCreditGrantsResponse, error)
	DescribeOrganizationBilling(ctx context.Context,
		orgID string,
		req *pb.DescribeOrganizationBillingRequest) (*pb.DescribeOrganizationBillingResponse, error)
	CreateBusinessCheckout(ctx context.Context,
		orgID string,
		req *pb.CreateBusinessCheckoutRequest,
		accountID string,
		baseURL string) (*pb.CreateBusinessCheckoutResponse, error)
	SelectBillingPlan(ctx context.Context, orgID string, req *pb.SelectBillingPlanRequest, accountID, baseURL string) (*pb.CreateBusinessCheckoutResponse, error)
	CreatePaymentMethodSession(ctx context.Context, orgID string, req *pb.CreatePaymentMethodSessionRequest, accountID string) (*pb.CreateBillingPortalSessionResponse, error)
	PreviewSeatPurchase(ctx context.Context, orgID string, req *pb.PreviewSeatPurchaseRequest) (*pb.PreviewSeatPurchaseResponse, error)
	PurchaseSeats(ctx context.Context, orgID string, req *pb.PurchaseSeatsRequest, userID string) (*pb.PurchaseSeatsResponse, error)
	SyncOrganizationBilling(ctx context.Context,
		orgID string,
		req *pb.SyncOrganizationBillingRequest) (*pb.DescribeOrganizationBillingResponse, error)
	CancelOrganizationSubscription(ctx context.Context,
		orgID string,
		req *pb.CancelOrganizationSubscriptionRequest) (*pb.DescribeOrganizationBillingResponse, error)
	ResumeOrganizationSubscription(ctx context.Context,
		orgID string,
		req *pb.ResumeOrganizationSubscriptionRequest) (*pb.DescribeOrganizationBillingResponse, error)
}
