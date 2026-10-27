package organizations

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/grpc/organizationref"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

func CreatePaymentMethodSession(ctx context.Context, orgID string, _ *pb.CreatePaymentMethodSessionRequest, accountID string) (*pb.CreateBillingPortalSessionResponse, error) {
	organizationID, err := organizationref.Resolve(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !billingmodels.BillingEnabled() || !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "Billing is not configured.")
	}
	if err := ensureOrganizationBillingCustomer(ctx, organizationID, accountID); err != nil {
		return nil, err
	}
	session, err := openPolarPortal(ctx, polar.NewClientFromEnv(), polar.CustomerSessionRequest{
		ExternalCustomerID: organizationID.String(),
		ExternalMemberID:   organizationID.String(),
	})
	if err != nil {
		return nil, polarBillingError(err, "Could not open card setup.")
	}
	if strings.TrimSpace(session.Token) == "" {
		return nil, grpcerrors.Internal(errors.New("Polar customer session has no token"), "Could not open card setup.")
	}
	return &pb.CreateBillingPortalSessionResponse{PortalUrl: session.PortalURL, SessionToken: session.Token}, nil
}

func ensureOrganizationBillingCustomer(ctx context.Context, orgID uuid.UUID, accountID string) error {
	email := actingUserEmail(ctx, orgID.String(), accountID)
	if email == "" {
		return grpcerrors.FailedPrecondition(nil, "A user email is required to set up billing.")
	}
	organization, err := models.FindOrganizationByIDInTransaction(database.DB(ctx), orgID.String())
	if err != nil {
		return grpcerrors.NotFound(err, "Organization not found.")
	}
	customer, err := polar.NewClientFromEnv().EnsureCustomer(ctx, polar.CreateCustomerInput{ExternalID: orgID.String(), Name: organization.Name, OwnerEmail: email})
	if err != nil {
		return polarBillingError(err, "Could not set up billing.")
	}
	if err := billingmodels.SetOrganizationPolarCustomerID(database.DB(ctx), orgID, customer.ID); err != nil {
		return grpcerrors.Internal(err, "Could not set up billing.")
	}
	return nil
}

func SelectBillingPlan(ctx context.Context, orgID string, req *pb.SelectBillingPlanRequest, accountID, baseURL string) (*pb.CreateBusinessCheckoutResponse, error) {
	organizationID, err := organizationref.Resolve(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !billingmodels.BillingEnabled() || !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "Billing is not configured.")
	}
	db := database.DB(ctx)
	current, err := billingmodels.ResolveOrganizationBillingPlan(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load the current plan.")
	}
	// While Polar retries a failed renewal, any selection would clear the
	// stored subscription state and release the payment lock. Recovery comes
	// from an updated payment method or a cancellation in the billing portal.
	if billingmodels.PolarSubscriptionIsPastDue(current.PolarSubscriptionStatus) {
		return nil, grpcerrors.FailedPrecondition(nil, "The last renewal payment failed. Update the payment method to recover the current subscription.")
	}
	// A live subscription can still charge, and a local plan change would
	// clear its stored state. This covers billed states like paused that
	// OrganizationBillingIsPolarManaged does not. The subscription must be
	// canceled in the billing portal before any plan selection.
	if current.HasLivePolarSubscription(time.Now()) {
		return nil, grpcerrors.FailedPrecondition(nil, "Manage your current subscription in the billing portal before selecting another plan.")
	}
	if strings.TrimSpace(req.GetPlanKey()) == billingmodels.BillingPlanNone {
		if req.Seats != nil {
			return nil, grpcerrors.InvalidArgument(nil, "Seats are only available for per-seat plans.")
		}
		if err := reconcileCheckoutBeforeLocalSelection(ctx, organizationID); err != nil {
			return nil, err
		}
		if _, err := billingmodels.ActivateOrganizationBillingPlan(db, organizationID, billingmodels.BillingPlanNone); err != nil {
			return nil, grpcerrors.FailedPrecondition(err, "Could not select this plan.")
		}
		return &pb.CreateBusinessCheckoutResponse{}, nil
	}
	catalog, err := billingmodels.FindBillingPlan(db, strings.TrimSpace(req.GetPlanKey()))
	if err != nil || catalog.Category != "paid" || catalog.Visibility != "public" {
		return nil, grpcerrors.InvalidArgument(nil, "Select an available paid plan.")
	}
	if billingmodels.OrganizationBillingIsPolarManaged(current) {
		return nil, grpcerrors.FailedPrecondition(nil, "Manage your current subscription in the billing portal before selecting another plan.")
	}
	seats, err := validateCheckoutSeats(db, organizationID, catalog, req.Seats)
	if err != nil {
		return nil, err
	}

	successURL := businessCheckoutSuccessURL(baseURL, organizationID)
	if returnPath := req.GetReturnPath(); returnPath != "" {
		target, err := url.Parse(returnPath)
		if err != nil || !strings.HasPrefix(returnPath, "/") || strings.HasPrefix(returnPath, "//") || strings.Contains(returnPath, `\`) || target.IsAbs() || target.Host != "" || target.Fragment != "" {
			return nil, grpcerrors.InvalidArgument(nil, "The checkout return path must be a local path.")
		}
		origin, err := url.Parse(successURL)
		if err != nil {
			return nil, grpcerrors.Internal(err, "Could not start subscription checkout.")
		}
		target.RawQuery += "&subscribed=1&checkout_id={CHECKOUT_ID}"
		successURL = origin.ResolveReference(target).String()
	}
	if catalog.PolarProductID != nil {
		if err := ensureOrganizationBillingCustomer(ctx, organizationID, accountID); err != nil {
			return nil, err
		}
		session, err := startSubscriptionCheckout(ctx, subscriptionCheckoutRequest{
			OrganizationID: organizationID,
			PlanKey:        catalog.Key,
			ProductID:      *catalog.PolarProductID,
			Seats:          seats,
			SuccessURL:     successURL,
			CustomerIP:     clientIPFromContext(ctx),
		})
		if err != nil {
			return nil, err
		}
		if account, err := uuid.Parse(accountID); err == nil {
			if err := db.Transaction(func(tx *gorm.DB) error {
				return billingmodels.GrantWelcomeCredit(tx, organizationID, account)
			}); err != nil {
				return nil, grpcerrors.Internal(err, "Could not apply signup credit.")
			}
		}
		return &pb.CreateBusinessCheckoutResponse{CheckoutUrl: session.URL}, nil
	}
	if catalog.PriceCents != 0 {
		return nil, grpcerrors.FailedPrecondition(nil, "Checkout is not available for this plan. Contact an organization admin.")
	}
	if err := reconcileCheckoutBeforeLocalSelection(ctx, organizationID); err != nil {
		return nil, err
	}
	if catalog.RequirePaymentMethod {
		hasCard, err := polar.NewClientFromEnv().HasPaymentMethod(ctx, organizationID.String())
		if err != nil {
			return nil, polarBillingError(err, "Could not verify the payment method.")
		}
		if !hasCard {
			return nil, grpcerrors.FailedPrecondition(nil, "Add a payment method before selecting this plan.")
		}
	}
	if err := activatePlanWithWelcomeCredit(db, organizationID, accountID, catalog); err != nil {
		return nil, grpcerrors.FailedPrecondition(err, "Could not select this plan.")
	}
	return &pb.CreateBusinessCheckoutResponse{}, nil
}

// activatePlanWithWelcomeCredit selects a plan that needs no checkout. The
// one-time signup credit is also granted if an earlier onboarding deferred it.
func activatePlanWithWelcomeCredit(db *gorm.DB, organizationID uuid.UUID, accountID string, catalog *billingmodels.BillingPlan) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if _, err := billingmodels.ActivateOrganizationBillingPlan(tx, organizationID, catalog.Key); err != nil {
			return err
		}
		account, err := uuid.Parse(accountID)
		if err != nil {
			return nil
		}
		return billingmodels.GrantWelcomeCredit(tx, organizationID, account)
	})
}

func validateCheckoutSeats(db *gorm.DB, orgID uuid.UUID, catalog *billingmodels.BillingPlan, seats *int32) (*int, error) {
	if !catalog.PerSeat {
		if seats != nil {
			return nil, grpcerrors.InvalidArgument(nil, "Seats are only available for per-seat plans.")
		}
		return nil, nil
	}
	if seats == nil {
		return nil, grpcerrors.InvalidArgument(nil, "Select a seat count for this plan.")
	}
	usage, err := entitlements.ResolveSeats(db, orgID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load seat usage.")
	}
	minimum := max(usage.Occupied, 1)
	quantity := int(*seats)
	if quantity < minimum {
		return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("Select at least %d seats to cover current members.", minimum))
	}
	maximum, err := entitlements.SeatMaximumForPlan(db, orgID, catalog)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load seat limits.")
	}
	if maximum != nil && quantity > *maximum {
		return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("The %s plan allows up to %d seats.", catalog.Name, *maximum))
	}
	if catalog.PolarProductID == nil || strings.TrimSpace(*catalog.PolarProductID) == "" {
		return nil, grpcerrors.FailedPrecondition(nil, "Checkout is not available for this plan.")
	}
	return &quantity, nil
}
