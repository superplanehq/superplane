package organizations

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
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

	productFeatures, err := plan.ResolveProductFeatures(db)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load product permissions.")
	}

	credit, err := models.DescribeOrganizationLLMCredit(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization billing")
	}

	billingEnabled, hasCustomer := billingState(ctx, organizationID)
	newBilling, err := models.OrganizationHasExperimentalFeatures(db, organizationID, features.FeatureNewBilling)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load billing settings.")
	}
	hasCard := false
	if billingEnabled && newBilling && hasCustomer {
		hasCard, err = polar.NewClientFromEnv().HasPaymentMethod(ctx, organizationID.String())
		if err != nil {
			return nil, polarBillingError(err, "Could not verify the payment method.")
		}
	}
	available := []*pb.BillingPlanOption{}
	checkoutEnabled := false
	if billingEnabled && !newBilling {
		productID, err := polar.BusinessCheckoutProductID(db)
		if err != nil {
			return nil, grpcerrors.Internal(err, "Could not load billing plans.")
		}
		checkoutEnabled = productID != ""
	}
	if billingEnabled && newBilling {
		checkoutEnabled = false
		catalog, err := models.ListBillingPlans(db)
		if err != nil {
			return nil, grpcerrors.Internal(err, "Could not load billing plans.")
		}
		override, err := models.FindOrganizationSeatLimitOverride(db, organizationID)
		if err != nil {
			return nil, grpcerrors.Internal(err, "Could not load seat limits.")
		}
		for _, item := range catalog {
			if item.Visibility != "public" || item.Category != "paid" {
				continue
			}
			checkoutEnabled = checkoutEnabled || item.PolarProductID != nil
			maximum := entitlements.SeatMaximum(item.SeatLimit, override)
			var seatMaximum *int32
			if maximum != nil {
				seatMaximum = new(int32(*maximum))
			}
			available = append(available, &pb.BillingPlanOption{
				Key: item.Key, Name: item.Name, PriceCents: item.PriceCents,
				PerSeat: item.PerSeat, SeatMaximum: seatMaximum,
				RequirePaymentMethod: item.RequirePaymentMethod,
				SubscriptionRequired: item.PerSeat || item.PolarProductID != nil || item.PriceCents != 0,
				OnboardingDefault:    item.OnboardingDefault,
				TrialCreditCents:     item.TrialCreditCents,
			})
		}
	}
	response := &pb.DescribeOrganizationBillingResponse{
		Plan:                        plan.Plan,
		ProductFeatures:             productFeatures,
		RequirePaymentMethod:        models.BillingEnabled() && newBilling && plan.Catalog.RequirePaymentMethod,
		HasPaymentMethod:            hasCard,
		AvailablePlans:              available,
		PlanName:                    plan.Catalog.Name,
		PlanCategory:                plan.Catalog.Category,
		PlanPriceCents:              plan.Catalog.PriceCents,
		PlanPerSeat:                 plan.Catalog.PerSeat,
		CatalogKey:                  plan.Catalog.Key,
		HostedExecutionAllowed:      !models.BillingEnabled() || plan.AllowsHostedExecution(time.Now()) && (!newBilling || !plan.Catalog.RequirePaymentMethod || hasCard),
		HostedCreditAvailable:       !models.BillingEnabled() || credit.RemainingMicros > 0,
		PlanSource:                  plan.PlanSource,
		PolarSubscriptionStatus:     plan.PolarSubscriptionStatus,
		TrialEndsAt:                 protoTimestamp(plan.TrialEndsAt),
		CurrentPeriodStart:          protoTimestamp(plan.CurrentPeriodStart),
		CurrentPeriodEnd:            protoTimestamp(plan.CurrentPeriodEnd),
		CreditPurchaseAllowed:       plan.AllowsCreditPurchase(),
		BillingEnabled:              billingEnabled,
		SubscriptionCheckoutEnabled: checkoutEnabled,
		HasBillingCustomer:          hasCustomer,
		RemainingCreditCents:        pricebook.MicrosToCents(credit.RemainingMicros),
		IncludedRemainingCents:      pricebook.MicrosToCents(credit.IncludedRemainingMicros),
		PurchasedRemainingCents:     pricebook.MicrosToCents(credit.PurchasedRemainingMicros),
		WelcomeRemainingCents:       pricebook.MicrosToCents(credit.WelcomeRemainingMicros),
		AdminRemainingCents:         pricebook.MicrosToCents(credit.AdminRemainingMicros),
		CancelAtPeriodEnd:           plan.CancelAtPeriodEnd,
	}
	if !models.BillingEnabled() {
		return response, nil
	}
	if err := describePastDue(db, plan, response); err != nil {
		return nil, err
	}

	lastSeatPurchase, err := models.FindLatestSeatPurchase(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load the last seat purchase.")
	}
	response.LastSeatPurchase = seatPurchaseToProto(lastSeatPurchase)

	seats, err := entitlements.ResolveSeatsForPlan(db, organizationID, plan)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load seat usage.")
	}
	response.OccupiedSeats = new(int32(seats.Occupied))
	response.PendingInvitationSeats = new(int32(seats.PendingInvitations))
	if seats.Invitable != nil {
		response.InvitableSeats = new(int32(*seats.Invitable))
	}
	if seats.PaymentBlocked {
		// The plan row can already be off per-seat, but the stored seats still
		// describe who can use the workspace.
		response.PurchasedSeats = new(int32(*seats.Purchased))
		describeBlockedRenewal(ctx, db, plan, seats, response)
		return response, nil
	}
	if !seats.PerSeat {
		return response, nil
	}

	response.PurchasedSeats = new(int32(*seats.Purchased))
	response.AvailableSeats = new(int32(*seats.Available))
	response.CurrentMonthlyTotalCents = new(int64(*seats.Purchased) * plan.Catalog.PriceCents)
	// A plan without paid seats has no renewal count until it is repaired.
	if seats.RenewalQuantity != nil {
		response.ExpectedRenewalSeats = new(int32(*seats.RenewalQuantity))
		response.ExpectedRenewalCents = new(int64(*seats.RenewalQuantity) * plan.Catalog.PriceCents)
	}
	if seats.PlanMaximum != nil {
		response.PlanMaximumSeats = new(int32(*seats.PlanMaximum))
	}
	if seats.Purchasable != nil {
		response.PurchasableSeats = new(int32(*seats.Purchasable))
	}
	return response, nil
}

// describeBlockedRenewal fills the renewal estimate while payment is blocked,
// so the admin can see what the recovered subscription charges. A plan
// without paid seats gets no estimate, and an unverified price leaves the
// amount empty instead of zero.
func describeBlockedRenewal(
	ctx context.Context,
	db *gorm.DB,
	plan *models.OrganizationBillingPlan,
	seats *entitlements.SeatUsage,
	response *pb.DescribeOrganizationBillingResponse,
) {
	if seats.RenewalQuantity == nil {
		return
	}
	response.ExpectedRenewalSeats = new(int32(*seats.RenewalQuantity))
	price := verifiedPaidSeatPriceCents(ctx, db, plan)
	if price == nil {
		return
	}
	response.ExpectedRenewalCents = new(int64(*seats.RenewalQuantity) * *price)
}

// verifiedPaidSeatPriceCents returns the per-seat price of the paid plan that
// the subscription renews with, or nil when it cannot be verified. A grace
// period keeps the paid catalog on the row. After the fallback, the row points
// to the access plan, so only the Polar subscription names the paid product.
// The access plan price is never a renewal price. A Polar failure only hides
// the amount: it must not fail the billing page.
func verifiedPaidSeatPriceCents(ctx context.Context, db *gorm.DB, plan *models.OrganizationBillingPlan) *int64 {
	if isPaidPerSeatCatalog(&plan.Catalog) {
		return new(plan.Catalog.PriceCents)
	}
	if plan.PolarSubscriptionID == nil || !polar.Configured() {
		return nil
	}
	subscriptionID := strings.TrimSpace(*plan.PolarSubscriptionID)
	if subscriptionID == "" {
		return nil
	}
	logger := log.WithFields(log.Fields{"organization_id": plan.OrganizationID.String(), "polar_subscription_id": subscriptionID})
	subscription, err := polar.NewClientFromEnv().GetSubscription(ctx, subscriptionID)
	if err != nil {
		logger.WithError(err).Info("renewal price is not available: could not read the Polar subscription")
		return nil
	}
	if subscription.ID != subscriptionID || strings.TrimSpace(subscription.ProductID) == "" {
		return nil
	}
	catalog, err := models.FindBillingPlanByProduct(db, subscription.ProductID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithError(err).Warn("renewal price is not available: could not load the billing plan")
		}
		return nil
	}
	if !isPaidPerSeatCatalog(catalog) {
		return nil
	}
	return new(catalog.PriceCents)
}

func isPaidPerSeatCatalog(catalog *models.BillingPlan) bool {
	return catalog.Category == "paid" && catalog.PerSeat
}

// describePastDue reports when a failed renewal payment started and, while a
// grace period keeps Team features on, when they end.
func describePastDue(db *gorm.DB, plan *models.OrganizationBillingPlan, response *pb.DescribeOrganizationBillingResponse) error {
	if !models.PolarSubscriptionIsPastDue(plan.PolarSubscriptionStatus) || plan.PastDueSince == nil {
		return nil
	}
	response.PaymentFailedAt = protoTimestamp(plan.PastDueSince)
	if !plan.IsActiveBusiness() {
		return nil
	}
	metadata, err := models.GetInstallationMetadata(db)
	if err != nil {
		return grpcerrors.Internal(err, "Could not load billing settings.")
	}
	response.PaidAccessEndsAt = protoTimestamp(plan.PastDueAccessEndsAt(metadata.PastDueGraceDays))
	return nil
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

	if !models.BillingEnabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "hosted billing is not configured")
	}

	if err := polar.SyncOrganizationSubscription(ctx, database.DB(ctx), organizationID); err != nil {
		log.WithError(err).WithField("organization_id", organizationID.String()).Warn("failed to sync Polar subscription")
	}
	_ = reconcileSeatPurchase(ctx, database.DB(ctx), organizationID)
	// Sync can apply an activation or a payment recovery that no webhook
	// delivered. Either one can change the renewal seat count.
	seatrenewal.ScheduleAsync(ctx, organizationID)

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
	if !models.BillingEnabled() || !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "Business checkout is not configured")
	}
	productID, err := polar.BusinessCheckoutProductID(database.DB(ctx))
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load billing plans.")
	}
	if productID == "" {
		return nil, grpcerrors.FailedPrecondition(nil, "Business checkout is not configured")
	}

	current, err := models.FindOrganizationBillingPlan(database.DB(ctx), organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load the current plan.")
	}
	if current != nil && models.PolarSubscriptionIsPastDue(current.PolarSubscriptionStatus) {
		return nil, grpcerrors.FailedPrecondition(nil, "The last renewal payment failed. Update the payment method to recover the current subscription.")
	}
	newBilling, err := models.OrganizationHasExperimentalFeatures(database.DB(ctx), organizationID, features.FeatureNewBilling)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load billing settings.")
	}
	if newBilling {
		return nil, grpcerrors.FailedPrecondition(nil, "Select a plan in Billing.")
	}
	// A paused subscription has no paid access but can resume billing. Checkout
	// must not start a second subscription while any live one exists.
	if current.HasLivePolarSubscription(time.Now()) {
		return nil, grpcerrors.FailedPrecondition(nil, "Manage your current subscription in the billing portal before selecting another plan.")
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

	session, err := startSubscriptionCheckout(ctx, subscriptionCheckoutRequest{
		OrganizationID: organizationID,
		PlanKey:        models.BillingPlanBusiness,
		ProductID:      productID,
		SuccessURL:     businessCheckoutSuccessURL(baseURL, organizationID),
		CustomerIP:     clientIPFromContext(ctx),
	})
	if err != nil {
		return nil, err
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
	return origin + "/" + organizationID.String() + "/organization/billing?subscribed=1&checkout_id={CHECKOUT_ID}"
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
	if !models.BillingEnabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "Subscription checkout is not configured.")
	}
	if err := polar.CancelOrganizationSubscription(ctx, database.DB(ctx), organizationID); err != nil {
		return nil, subscriptionCancelError(err, "Could not cancel the subscription.")
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
	if !models.BillingEnabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "Subscription checkout is not configured.")
	}
	if err := polar.ResumeOrganizationSubscription(ctx, database.DB(ctx), organizationID); err != nil {
		return nil, subscriptionCancelError(err, "Could not resume the subscription.")
	}
	// The resumed subscription renews again, so schedule the next-period
	// seat count. A Polar failure here does not fail the resume.
	seatrenewal.ScheduleAsync(ctx, organizationID)
	return DescribeOrganizationBilling(ctx, orgID, &pb.DescribeOrganizationBillingRequest{Id: req.GetId()})
}

func subscriptionCancelError(err error, fallback string) error {
	switch {
	case errors.Is(err, polar.ErrSubscriptionCheckoutDisabled):
		return grpcerrors.FailedPrecondition(err, "Subscription checkout is not configured.")
	case errors.Is(err, polar.ErrAdminPlanCannotCancel):
		return grpcerrors.FailedPrecondition(err, "An admin plan cannot be canceled from Billing.")
	case errors.Is(err, polar.ErrSubscriptionNotCancelable), polar.IsNotFound(err):
		return grpcerrors.FailedPrecondition(err, "This organization has no subscription to cancel.")
	case errors.Is(err, polar.ErrSubscriptionAlreadyCanceling):
		return grpcerrors.FailedPrecondition(err, "The subscription is already set to end at the period end.")
	case errors.Is(err, polar.ErrSubscriptionNotCanceling):
		return grpcerrors.FailedPrecondition(err, "The subscription is not set to end at the period end.")
	default:
		return polarBillingError(err, fallback)
	}
}
