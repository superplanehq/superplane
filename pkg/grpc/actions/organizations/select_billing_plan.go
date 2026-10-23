package organizations

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

func CreatePaymentMethodSession(ctx context.Context, orgID string, _ *pb.CreatePaymentMethodSessionRequest, accountID string) (*pb.CreateBillingPortalSessionResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !models.BillingEnabled() || !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "Billing is not configured.")
	}
	if err := requireNewBilling(ctx, organizationID); err != nil {
		return nil, err
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
	if err := models.SetOrganizationPolarCustomerID(database.DB(ctx), orgID, customer.ID); err != nil {
		return grpcerrors.Internal(err, "Could not set up billing.")
	}
	return nil
}

func SelectBillingPlan(ctx context.Context, orgID string, req *pb.SelectBillingPlanRequest, accountID, baseURL string) (*pb.CreateBusinessCheckoutResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !models.BillingEnabled() || !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "Billing is not configured.")
	}
	if err := requireNewBilling(ctx, organizationID); err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	current, err := models.ResolveOrganizationBillingPlan(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load the current plan.")
	}
	if strings.TrimSpace(req.GetPlanKey()) == models.BillingPlanNone {
		if req.Seats != nil {
			return nil, grpcerrors.InvalidArgument(nil, "Seats are only available for per-seat plans.")
		}
		if _, err := models.ActivateOrganizationBillingPlan(db, organizationID, models.BillingPlanNone); err != nil {
			return nil, grpcerrors.FailedPrecondition(err, "Could not select this plan.")
		}
		return &pb.CreateBusinessCheckoutResponse{}, nil
	}
	catalog, err := models.FindBillingPlan(db, strings.TrimSpace(req.GetPlanKey()))
	if err != nil || catalog.Category != "paid" || catalog.Visibility != "public" {
		return nil, grpcerrors.InvalidArgument(nil, "Select an available paid plan.")
	}
	if models.OrganizationBillingIsPolarManaged(current) {
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
		session, err := polar.NewClientFromEnv().CreateCheckout(ctx, polar.CheckoutInput{ProductID: *catalog.PolarProductID, ExternalCustomerID: organizationID.String(), SuccessURL: successURL, CustomerIP: clientIPFromContext(ctx), Seats: seats})
		if err != nil {
			return nil, polarBillingError(err, "Could not start subscription checkout.")
		}
		return &pb.CreateBusinessCheckoutResponse{CheckoutUrl: session.URL}, nil
	}
	if catalog.PriceCents != 0 {
		return nil, grpcerrors.FailedPrecondition(nil, "Checkout is not available for this plan. Contact an organization admin.")
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
	if _, err := models.ActivateOrganizationBillingPlan(db, organizationID, catalog.Key); err != nil {
		return nil, grpcerrors.FailedPrecondition(err, "Could not select this plan.")
	}
	return &pb.CreateBusinessCheckoutResponse{}, nil
}

func requireNewBilling(ctx context.Context, orgID uuid.UUID) error {
	enabled, err := models.OrganizationHasExperimentalFeatures(database.DB(ctx), orgID, features.FeatureNewBilling)
	if err != nil {
		return grpcerrors.Internal(err, "Could not load billing settings.")
	}
	if !enabled {
		return grpcerrors.FailedPrecondition(nil, "New billing is not enabled for this organization.")
	}
	return nil
}

func validateCheckoutSeats(db *gorm.DB, orgID uuid.UUID, catalog *models.BillingPlan, seats *int32) (*int, error) {
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
