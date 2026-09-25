package organizations

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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
	plan, err := models.ResolveOrganizationBillingPlan(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization billing")
	}

	credit, err := models.DescribeOrganizationLLMCredit(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization billing")
	}

	billingEnabled, hasCustomer := billingState(ctx, organizationID)
	return &pb.DescribeOrganizationBillingResponse{
		Plan:                        plan.Plan,
		PlanSource:                  plan.PlanSource,
		PolarSubscriptionStatus:     plan.PolarSubscriptionStatus,
		TrialEndsAt:                 protoTimestamp(plan.TrialEndsAt),
		CurrentPeriodStart:          protoTimestamp(plan.CurrentPeriodStart),
		CurrentPeriodEnd:            protoTimestamp(plan.CurrentPeriodEnd),
		CreditPurchaseAllowed:       plan.AllowsCreditPurchase(),
		BillingEnabled:              billingEnabled,
		SubscriptionCheckoutEnabled: polar.SubscriptionCheckoutEnabled(),
		HasBillingCustomer:          hasCustomer,
		RemainingCreditCents:        pricebook.MicrosToCents(credit.RemainingMicros),
		IncludedRemainingCents:      pricebook.MicrosToCents(credit.IncludedRemainingMicros),
		PurchasedRemainingCents:     pricebook.MicrosToCents(credit.PurchasedRemainingMicros),
		WelcomeRemainingCents:       pricebook.MicrosToCents(credit.WelcomeRemainingMicros),
		AdminRemainingCents:         pricebook.MicrosToCents(credit.AdminRemainingMicros),
		CancelAtPeriodEnd:           plan.CancelAtPeriodEnd,
	}, nil
}

func SyncOrganizationBilling(
	ctx context.Context,
	orgID string,
	req *pb.SyncOrganizationBillingRequest,
) (*pb.DescribeOrganizationBillingResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if err := polar.SyncOrganizationSubscription(ctx, database.DB(ctx), organizationID); err != nil {
		log.WithError(err).WithField("organization_id", organizationID.String()).Warn("failed to sync Polar subscription")
	}

	return DescribeOrganizationBilling(ctx, orgID, &pb.DescribeOrganizationBillingRequest{Id: req.GetId()})
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

func CancelOrganizationSubscription(
	ctx context.Context,
	orgID string,
	req *pb.CancelOrganizationSubscriptionRequest,
) (*pb.DescribeOrganizationBillingResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if err := polar.CancelOrganizationSubscription(ctx, database.DB(ctx), organizationID); err != nil {
		return nil, subscriptionCancelError(err, "failed to cancel Business")
	}
	return DescribeOrganizationBilling(ctx, orgID, &pb.DescribeOrganizationBillingRequest{Id: req.GetId()})
}

func ResumeOrganizationSubscription(
	ctx context.Context,
	orgID string,
	req *pb.ResumeOrganizationSubscriptionRequest,
) (*pb.DescribeOrganizationBillingResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if err := polar.ResumeOrganizationSubscription(ctx, database.DB(ctx), organizationID); err != nil {
		return nil, subscriptionCancelError(err, "failed to keep Business")
	}
	return DescribeOrganizationBilling(ctx, orgID, &pb.DescribeOrganizationBillingRequest{Id: req.GetId()})
}

func subscriptionCancelError(err error, fallback string) error {
	switch {
	case errors.Is(err, polar.ErrSubscriptionCheckoutDisabled):
		return grpcerrors.FailedPrecondition(err, "Business checkout is not configured")
	case errors.Is(err, polar.ErrAdminPlanCannotCancel):
		return grpcerrors.FailedPrecondition(err, "An admin plan cannot be canceled from Billing.")
	case errors.Is(err, polar.ErrSubscriptionNotCancelable), polar.IsNotFound(err):
		return grpcerrors.FailedPrecondition(err, "This organization has no Business subscription to cancel.")
	case errors.Is(err, polar.ErrSubscriptionAlreadyCanceling):
		return grpcerrors.FailedPrecondition(err, "Business is already set to end at the period end.")
	case errors.Is(err, polar.ErrSubscriptionNotCanceling):
		return grpcerrors.FailedPrecondition(err, "Business is not set to end at the period end.")
	default:
		return polarBillingError(err, fallback)
	}
}
