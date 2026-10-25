package organizations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	// checkoutIntentGraceWindow is how long an intent without a Polar checkout
	// can still become one. Polar documents no default checkout lifetime, so
	// this is a chosen value. A create call that a client abandoned cannot
	// take effect at Polar after this window.
	checkoutIntentGraceWindow = 10 * time.Minute

	// maxCheckoutClaimAttempts bounds the claim and reconcile loop. Each
	// repeat means that another request changed the intent in between.
	maxCheckoutClaimAttempts = 3
)

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
	if err := adoptPreexistingCheckout(ctx, client, request); err != nil {
		return nil, err
	}

	for range maxCheckoutClaimAttempts {
		intent, claimed, err := models.ClaimOrganizationBillingCheckout(
			database.DB(ctx), request.OrganizationID, request.PlanKey, request.ProductID, request.Seats,
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "Organization not found.")
		}
		if err != nil {
			return nil, grpcerrors.Internal(err, "Could not start subscription checkout.")
		}
		if claimed {
			return createClaimedCheckout(ctx, client, intent, request)
		}
		session, err := reconcileUnresolvedCheckout(ctx, client, intent, request)
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
func createClaimedCheckout(ctx context.Context, client *polar.Client, intent *models.OrganizationBillingCheckout, request subscriptionCheckoutRequest) (*polar.CheckoutSession, error) {
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
	if err := intent.MarkOpen(db, checkoutSnapshot(session)); err != nil {
		// Polar holds the checkout. The intent stays creating, and a later
		// request finds the checkout through the intent ID in its metadata.
		return nil, grpcerrors.Internal(err, "Could not start subscription checkout.")
	}
	return session, nil
}

// recordFailedCheckoutCreate keeps the intent in the state that the failure
// proves. A definite rejection means no checkout exists. A lost response means
// a checkout may exist, so the intent keeps blocking new checkouts.
func recordFailedCheckoutCreate(db *gorm.DB, intent *models.OrganizationBillingCheckout, createErr error) {
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
func reconcileUnresolvedCheckout(ctx context.Context, client *polar.Client, intent *models.OrganizationBillingCheckout, request subscriptionCheckoutRequest) (*polar.CheckoutSession, error) {
	switch {
	case intent.State == models.BillingCheckoutStateOpen:
		return reconcileOpenCheckout(ctx, client, intent, request)
	case intent.State == models.BillingCheckoutStateAmbiguous, checkoutCreateIsStale(intent):
		return reconcileUndeliveredCheckout(ctx, client, intent, request)
	default:
		// Another request is creating the checkout now.
		return nil, checkoutInProgressError()
	}
}

// reconcileOpenCheckout reads the checkout from Polar. A past local expiry
// proves nothing. Only the Polar status resolves the intent.
func reconcileOpenCheckout(ctx context.Context, client *polar.Client, intent *models.OrganizationBillingCheckout, request subscriptionCheckoutRequest) (*polar.CheckoutSession, error) {
	checkout, err := client.GetCheckout(ctx, intent.PolarCheckoutIDValue())
	if err != nil {
		return nil, polarBillingError(err, "Could not verify the pending checkout.")
	}
	return applyPolarCheckout(ctx, intent, request, checkout)
}

// reconcileUndeliveredCheckout handles an intent whose create response never
// arrived. The intent ID in the checkout metadata finds a checkout that Polar
// created. Without a match, the intent can only be released after the grace
// window.
func reconcileUndeliveredCheckout(ctx context.Context, client *polar.Client, intent *models.OrganizationBillingCheckout, request subscriptionCheckoutRequest) (*polar.CheckoutSession, error) {
	checkouts, err := client.ListCheckouts(ctx, request.OrganizationID.String(), intent.PolarProductID)
	if err != nil {
		return nil, polarBillingError(err, "Could not verify the pending checkout.")
	}
	for i := range checkouts {
		if checkouts[i].IntentID() == intent.ID.String() {
			return applyPolarCheckout(ctx, intent, request, &checkouts[i])
		}
	}
	if time.Since(intent.CreatedAt) < checkoutIntentGraceWindow {
		return nil, grpcerrors.FailedPrecondition(nil, "The previous checkout could not be confirmed. Try again in a few minutes.")
	}
	db := database.DB(context.WithoutCancel(ctx))
	err = intent.Release(db, "no matching Polar checkout after the grace window")
	return nil, ignoreCheckoutConflict(err)
}

// applyPolarCheckout records what Polar reports and decides the request.
func applyPolarCheckout(ctx context.Context, intent *models.OrganizationBillingCheckout, request subscriptionCheckoutRequest, checkout *polar.CheckoutSession) (*polar.CheckoutSession, error) {
	db := database.DB(context.WithoutCancel(ctx))
	snapshot := checkoutSnapshot(checkout)

	if checkout.Status.Payable() {
		if err := intent.MarkOpen(db, snapshot); err != nil {
			return nil, ignoreCheckoutConflict(err)
		}
		if !intent.MatchesSelection(request.PlanKey, request.ProductID, request.Seats) {
			return nil, checkoutInProgressError()
		}
		if intent.CheckoutURLValue() == "" {
			return nil, grpcerrors.Internal(errors.New("pending checkout has no url"), "Could not start subscription checkout.")
		}
		return &polar.CheckoutSession{ID: checkout.ID, URL: intent.CheckoutURLValue(), Status: checkout.Status}, nil
	}

	state, _ := checkout.Status.IntentState()
	if err := intent.Resolve(db, state, &snapshot); err != nil {
		return nil, ignoreCheckoutConflict(err)
	}
	if checkout.Status == polar.CheckoutStatusSucceeded {
		return nil, requireNoLiveSubscription(ctx, request.OrganizationID)
	}
	return nil, nil
}

// requireNoLiveSubscription guards the time between a paid checkout and the
// webhook that applies its subscription. It reads the subscription from Polar
// before the caller opens another checkout.
func requireNoLiveSubscription(ctx context.Context, orgID uuid.UUID) error {
	db := database.DB(ctx)
	if err := polar.SyncOrganizationSubscription(ctx, db, orgID); err != nil {
		log.WithError(err).WithField("organization_id", orgID.String()).Warn("failed to sync Polar subscription after a paid checkout")
	}
	current, err := models.FindOrganizationBillingPlan(db, orgID)
	if err != nil {
		return grpcerrors.Internal(err, "Could not load the current plan.")
	}
	if current.HasLivePolarSubscription(time.Now()) {
		return grpcerrors.FailedPrecondition(nil, "Manage your current subscription in the billing portal before selecting another plan.")
	}
	return nil
}

// adoptPreexistingCheckout covers Polar checkouts that were opened before
// intents existed. An organization without intent rows can still hold a
// payable checkout, so the first request records it. A request never fabricates
// an intent: a checkout is adopted with its real Polar ID and URL, or not at all.
func adoptPreexistingCheckout(ctx context.Context, client *polar.Client, request subscriptionCheckoutRequest) error {
	db := database.DB(ctx)
	known, err := models.OrganizationHasBillingCheckouts(db, request.OrganizationID)
	if err != nil {
		return grpcerrors.Internal(err, "Could not start subscription checkout.")
	}
	if known {
		return nil
	}
	checkouts, err := client.ListCheckouts(ctx, request.OrganizationID.String(), request.ProductID)
	if err != nil {
		return polarBillingError(err, "Could not verify existing checkouts.")
	}
	for i := range checkouts {
		checkout := &checkouts[i]
		if !checkout.Status.Payable() || checkout.IntentID() != "" {
			continue
		}
		_, err := models.AdoptPreexistingOrganizationBillingCheckout(
			database.DB(context.WithoutCancel(ctx)), request.OrganizationID, request.PlanKey, request.ProductID, checkout.Seats, checkoutSnapshot(checkout),
		)
		if err != nil {
			return grpcerrors.Internal(err, "Could not start subscription checkout.")
		}
		return nil
	}
	return nil
}

// checkoutCreateIsStale reports a creating intent whose request is gone. The
// server can stop between the claim and the end of the Polar call.
func checkoutCreateIsStale(intent *models.OrganizationBillingCheckout) bool {
	return intent.State == models.BillingCheckoutStateCreating && time.Since(intent.CreatedAt) >= checkoutIntentGraceWindow
}

func checkoutSnapshot(checkout *polar.CheckoutSession) models.PolarCheckoutSnapshot {
	return models.PolarCheckoutSnapshot{
		CheckoutID: checkout.ID,
		URL:        checkout.URL,
		Status:     string(checkout.Status),
		ExpiresAt:  checkout.ExpiresAt,
	}
}

// ignoreCheckoutConflict treats a lost race as success. The winner changed the
// intent, and the next claim reads the new state.
func ignoreCheckoutConflict(err error) error {
	if err == nil || errors.Is(err, models.ErrBillingCheckoutStateConflict) {
		return nil
	}
	return grpcerrors.Internal(err, "Could not start subscription checkout.")
}

func checkoutInProgressError() error {
	return grpcerrors.FailedPrecondition(nil, "A checkout is already in progress. Finish it or wait until it expires.")
}
