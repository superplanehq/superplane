package organizations

import (
	"context"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
)

func DescribeOrganizationBilling(
	ctx context.Context,
	orgID string,
	_ *pb.DescribeOrganizationBillingRequest,
) (*pb.DescribeOrganizationBillingResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)
	plan, err := models.FindOrganizationBillingPlan(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization billing")
	}

	credit, err := models.DescribeOrganizationLLMCredit(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization billing")
	}

	billingEnabled, hasCustomer := billingState(ctx, organizationID)
	resp := &pb.DescribeOrganizationBillingResponse{
		BillingEnabled:              billingEnabled,
		SubscriptionCheckoutEnabled: polar.SubscriptionCheckoutEnabled(),
		HasBillingCustomer:          hasCustomer,
		RemainingCreditCents:        pricebook.MicrosToCents(credit.RemainingMicros),
		IncludedRemainingCents:      pricebook.MicrosToCents(credit.IncludedRemainingMicros),
		PurchasedRemainingCents:     pricebook.MicrosToCents(credit.PurchasedRemainingMicros),
	}
	if plan != nil {
		resp.Plan = plan.Plan
		resp.PlanSource = plan.PlanSource
		resp.PolarSubscriptionStatus = plan.PolarSubscriptionStatus
		resp.TrialEndsAt = protoTimestamp(plan.TrialEndsAt)
		resp.CurrentPeriodStart = protoTimestamp(plan.CurrentPeriodStart)
		resp.CurrentPeriodEnd = protoTimestamp(plan.CurrentPeriodEnd)
		resp.CreditPurchaseAllowed = plan.AllowsCreditPurchase()
	}
	return resp, nil
}

func CreateBusinessCheckout(
	ctx context.Context,
	orgID string,
	_ *pb.CreateBusinessCheckoutRequest,
	accountID string,
	baseURL string,
) (*pb.CreateBusinessCheckoutResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !polar.SubscriptionCheckoutEnabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "Business checkout is not configured")
	}

	email := actingUserEmail(ctx, organizationID.String(), accountID)
	if email == "" {
		return nil, grpcerrors.FailedPrecondition(nil, "A user email is required to start checkout.")
	}

	organization, err := models.FindOrganizationByIDInTransaction(database.DB(ctx), organizationID.String())
	if err != nil {
		return nil, grpcerrors.NotFound(err, "organization not found")
	}

	client := polar.NewClientFromEnv()
	customer, err := client.EnsureCustomer(ctx, polar.CreateCustomerInput{
		ExternalID: organizationID.String(),
		Name:       organization.Name,
		OwnerEmail: email,
	})
	if err != nil {
		if polar.IsConflict(err) {
			return nil, grpcerrors.FailedPrecondition(err, "hosted billing customer already exists for another organization")
		}
		return nil, polarBillingError(err, "failed to create Business checkout")
	}
	if err := models.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, customer.ID); err != nil {
		return nil, grpcerrors.Internal(err, "failed to create Business checkout")
	}

	session, err := client.CreateCheckout(
		ctx,
		polar.BusinessProductID(),
		organizationID.String(),
		businessCheckoutSuccessURL(baseURL, organizationID),
		clientIPFromContext(ctx),
	)
	if err != nil {
		return nil, polarBillingError(err, "failed to create Business checkout")
	}
	if session.CustomerID != "" {
		if err := models.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, session.CustomerID); err != nil {
			return nil, grpcerrors.Internal(err, "failed to create Business checkout")
		}
	}
	return &pb.CreateBusinessCheckoutResponse{CheckoutUrl: session.URL}, nil
}

func businessCheckoutSuccessURL(baseURL string, organizationID uuid.UUID) string {
	origin := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if origin == "" {
		origin = strings.TrimRight(strings.TrimSpace(os.Getenv("BASE_URL")), "/")
	}
	return origin + "/" + organizationID.String() + "/organization/billing?subscribed=1"
}
