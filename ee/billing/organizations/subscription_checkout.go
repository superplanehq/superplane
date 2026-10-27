package organizations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"gorm.io/gorm"
)

// maxCheckoutClaimAttempts bounds the claim and reconcile loop. Each repeat
// means that another request changed the intent in between.
const maxCheckoutClaimAttempts = 3

type subscriptionCheckoutRequest struct {
	OrganizationID uuid.UUID
	PlanKey        string
	ProductID      string
	Seats          *int
	SuccessURL     string
	CustomerIP     string
}

// startSubscriptionCheckout returns a payable Polar checkout for the request.
// An organization has at most one unresolved checkout intent, so a repeated or
// concurrent request reuses the pending checkout or waits for its outcome.
//
// The intent is claimed under the organization lock. Every Polar call runs
// after that transaction ended.
func startSubscriptionCheckout(ctx context.Context, request subscriptionCheckoutRequest) (*polar.CheckoutSession, error) {
	client := polar.NewClientFromEnv()
	lifecycle := polar.NewCheckoutLifecycle(client)
	if err := lifecycle.AdoptPreexisting(ctx, request.OrganizationID); err != nil {
		return nil, polarBillingError(err, "Could not verify existing checkouts.")
	}

	for range maxCheckoutClaimAttempts {
		intent, claimed, err := billingmodels.ClaimOrganizationBillingCheckout(
			database.DB(ctx), request.OrganizationID, request.PlanKey, request.ProductID, request.Seats,
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "Organization not found.")
		}
		if errors.Is(err, billingmodels.ErrBillingCheckoutLiveSubscription) {
			return nil, liveSubscriptionSelectionError()
		}
		if errors.Is(err, billingmodels.ErrBillingCheckoutSeatsBelowMembers) {
			return nil, grpcerrors.InvalidArgument(err, "Members changed. Refresh and select a seat for each member.")
		}
		if err != nil {
			return nil, grpcerrors.Internal(err, "Could not start subscription checkout.")
		}
		if claimed {
			return createClaimedCheckout(ctx, client, intent, request)
		}
		session, err := reconcileUnresolvedCheckout(ctx, lifecycle, intent, request)
		if err != nil {
			return nil, err
		}
		if session != nil {
			return session, nil
		}
		// The intent is resolved now. Claim again.
	}
	return nil, checkoutInProgressError()
}

// createClaimedCheckout makes the one Polar create call for a claimed intent.
func createClaimedCheckout(ctx context.Context, client *polar.Client, intent *billingmodels.OrganizationBillingCheckout, request subscriptionCheckoutRequest) (*polar.CheckoutSession, error) {
	session, err := client.CreateCheckout(ctx, polar.CheckoutInput{
		ProductID:          request.ProductID,
		ExternalCustomerID: request.OrganizationID.String(),
		SuccessURL:         request.SuccessURL,
		CustomerIP:         request.CustomerIP,
		Seats:              request.Seats,
		Metadata:           map[string]string{polar.CheckoutIntentMetadataKey: intent.ID.String()},
	})
	// A canceled request must still record the outcome.
	db := database.DB(context.WithoutCancel(ctx))
	if err != nil {
		recordFailedCheckoutCreate(db, intent, err)
		return nil, polarBillingError(err, "Could not start subscription checkout.")
	}
	if err := intent.MarkOpen(db, polar.CheckoutSnapshot(session)); err != nil {
		// Polar holds the checkout. The intent stays creating, and a later
		// request finds the checkout through the intent ID in its metadata.
		return nil, grpcerrors.Internal(err, "Could not start subscription checkout.")
	}
	return session, nil
}

// recordFailedCheckoutCreate keeps the intent in the state that the failure
// proves. A definite rejection means no checkout exists. A lost response means
// a checkout may exist, so the intent keeps blocking new checkouts.
func recordFailedCheckoutCreate(db *gorm.DB, intent *billingmodels.OrganizationBillingCheckout, createErr error) {
	var err error
	if polar.IsOutcomeUnknown(createErr) {
		err = intent.MarkAmbiguous(db, createErr.Error())
	} else {
		err = intent.Release(db, createErr.Error())
	}
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"organization_id":    intent.OrganizationID.String(),
			"checkout_intent_id": intent.ID.String(),
		}).Error("could not record the failed Polar checkout create")
	}
}

// reconcileUnresolvedCheckout checks an existing unresolved intent against
// Polar. It returns a checkout to reuse, or nil when the intent is resolved and
// the caller can claim again.
func reconcileUnresolvedCheckout(ctx context.Context, lifecycle *polar.CheckoutLifecycle, intent *billingmodels.OrganizationBillingCheckout, request subscriptionCheckoutRequest) (*polar.CheckoutSession, error) {
	verification, err := lifecycle.Verify(ctx, intent)
	if err != nil {
		return nil, polarBillingError(err, "Could not verify the pending checkout.")
	}
	switch verification.Outcome {
	case polar.CheckoutPayable:
		return reusablePendingCheckout(intent, request, verification.Checkout)
	case polar.CheckoutSubscriptionLive:
		return nil, liveSubscriptionSelectionError()
	case polar.CheckoutUnconfirmed:
		return nil, grpcerrors.FailedPrecondition(nil, "The previous checkout could not be confirmed. Try again in a few minutes.")
	case polar.CheckoutResolved, polar.CheckoutChanged:
		return nil, nil
	default:
		// Another request is creating the checkout now.
		return nil, checkoutInProgressError()
	}
}

// reusablePendingCheckout returns the payable checkout of the intent when the
// request asks for the same selection.
func reusablePendingCheckout(intent *billingmodels.OrganizationBillingCheckout, request subscriptionCheckoutRequest, checkout *polar.CheckoutSession) (*polar.CheckoutSession, error) {
	if !intent.MatchesSelection(request.PlanKey, request.ProductID, request.Seats) {
		return nil, checkoutInProgressError()
	}
	if intent.CheckoutURLValue() == "" {
		return nil, grpcerrors.Internal(errors.New("pending checkout has no url"), "Could not start subscription checkout.")
	}
	return &polar.CheckoutSession{ID: checkout.ID, URL: intent.CheckoutURLValue(), Status: checkout.Status}, nil
}

// reconcileCheckoutBeforeLocalSelection verifies pending checkouts before a
// local plan change. Activation rechecks the guard under the organization lock.
func reconcileCheckoutBeforeLocalSelection(ctx context.Context, orgID uuid.UUID) error {
	lifecycle := polar.NewCheckoutLifecycle(polar.NewClientFromEnv())
	if err := lifecycle.AdoptPreexisting(ctx, orgID); err != nil {
		return polarBillingError(err, "Could not verify existing checkouts.")
	}
	intent, err := billingmodels.FindUnresolvedOrganizationBillingCheckout(database.DB(ctx), orgID)
	if err != nil {
		return grpcerrors.Internal(err, "Could not verify the pending checkout.")
	}
	if intent == nil {
		return nil
	}
	_, err = reconcileUnresolvedCheckout(ctx, lifecycle, intent, subscriptionCheckoutRequest{OrganizationID: orgID})
	return err
}

func liveSubscriptionSelectionError() error {
	return grpcerrors.FailedPrecondition(nil, "Manage your current subscription in the billing portal before selecting another plan.")
}

func checkoutInProgressError() error {
	return grpcerrors.FailedPrecondition(nil, "A checkout is already in progress. Finish it or wait until it expires.")
}
