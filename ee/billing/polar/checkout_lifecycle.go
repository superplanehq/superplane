package polar

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

// CheckoutIntentGraceWindow is how long an intent without a Polar checkout
// can still become one. Polar documents no default checkout lifetime, so this
// is a chosen value. A create call that a client abandoned cannot take effect
// at Polar after this window.
const CheckoutIntentGraceWindow = 10 * time.Minute

// ErrCheckoutSeatLimit rejects a membership that would exceed the seats of a
// checkout that can still activate. Payment of that checkout must not leave
// more members than paid seats.
var ErrCheckoutSeatLimit = errors.New("a pending checkout has no seat for another member")

// CheckoutClient is the part of the Polar API that the checkout lifecycle
// uses. It only reads checkouts.
type CheckoutClient interface {
	GetCheckout(ctx context.Context, checkoutID string) (*CheckoutSession, error)
	ListCheckouts(ctx context.Context, externalCustomerID, productID string) ([]CheckoutSession, error)
}

// CheckoutOutcome is what Polar proved about an unresolved checkout intent.
type CheckoutOutcome string

const (
	// CheckoutPayable means that Polar holds a checkout that the customer
	// can still pay. The intent stays open.
	CheckoutPayable CheckoutOutcome = "payable"
	// CheckoutSubscriptionLive means that the checkout was paid and the
	// organization has a live subscription.
	CheckoutSubscriptionLive CheckoutOutcome = "subscription_live"
	// CheckoutResolved means that the checkout can no longer activate. Polar
	// verified its expiry or failure, no checkout appeared in the grace
	// window, or it was paid and no subscription is live. The intent is
	// resolved.
	CheckoutResolved CheckoutOutcome = "resolved"
	// CheckoutUnconfirmed means that the create call may have succeeded and
	// the grace window is still open. The intent stays unresolved.
	CheckoutUnconfirmed CheckoutOutcome = "unconfirmed"
	// CheckoutCreating means that another request is creating the checkout.
	CheckoutCreating CheckoutOutcome = "creating"
	// CheckoutChanged means that another writer changed the intent during
	// the verification. Read the intent again.
	CheckoutChanged CheckoutOutcome = "changed"
)

// CheckoutVerification is the result of Verify. Checkout is the Polar
// checkout for a payable intent.
type CheckoutVerification struct {
	Outcome  CheckoutOutcome
	Checkout *CheckoutSession
}

// CheckoutLifecycle verifies subscription checkout intents against Polar and
// records what Polar proves. Every Polar call runs outside a transaction and
// outside the organization lock. Only a verified Polar state releases an
// intent: a local expiry time or the elapsed time alone proves nothing.
type CheckoutLifecycle struct {
	client CheckoutClient
	now    func() time.Time
}

func NewCheckoutLifecycle(client CheckoutClient) *CheckoutLifecycle {
	return &CheckoutLifecycle{client: client, now: time.Now}
}

// Verify checks an unresolved intent against Polar. It records an open
// checkout, and it resolves the intent only when Polar proves that the
// checkout can no longer activate. An error leaves the intent unchanged, so
// its restrictions stay.
func (l *CheckoutLifecycle) Verify(ctx context.Context, intent *billingmodels.OrganizationBillingCheckout) (CheckoutVerification, error) {
	switch {
	case intent.State == billingmodels.BillingCheckoutStateOpen:
		checkout, err := l.client.GetCheckout(ctx, intent.PolarCheckoutIDValue())
		if err != nil {
			return CheckoutVerification{}, fmt.Errorf("read the pending Polar checkout: %w", err)
		}
		return l.record(ctx, intent, checkout)
	case intent.State == billingmodels.BillingCheckoutStateAmbiguous, l.createIsStale(intent):
		return l.verifyUndelivered(ctx, intent)
	default:
		return CheckoutVerification{Outcome: CheckoutCreating}, nil
	}
}

// AdoptPreexisting records a payable Polar checkout that SuperPlane opened
// before intents existed. An organization without intent rows can still hold
// such a checkout. A checkout is adopted with its real Polar ID and URL, or
// not at all.
func (l *CheckoutLifecycle) AdoptPreexisting(ctx context.Context, orgID uuid.UUID) error {
	db := database.DB(ctx)
	known, err := billingmodels.OrganizationHasBillingCheckouts(db, orgID)
	if err != nil || known {
		return err
	}
	if err := billingmodels.BindLegacyBillingProduct(db, BusinessProductID()); err != nil {
		return err
	}
	checkouts, err := l.client.ListCheckouts(ctx, orgID.String(), "")
	if err != nil {
		return fmt.Errorf("list Polar checkouts: %w", err)
	}
	for i := range checkouts {
		checkout := &checkouts[i]
		if !checkout.Status.Payable() || checkout.IntentID() != "" {
			continue
		}
		plan, err := billingmodels.FindBillingPlanByProduct(db, checkout.ProductID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue // Credit purchase products are not subscription plans.
		}
		if err != nil {
			return err
		}
		_, err = billingmodels.AdoptPreexistingOrganizationBillingCheckout(
			database.DB(context.WithoutCancel(ctx)), orgID, plan.Key, checkout.ProductID, checkout.Seats, CheckoutSnapshot(checkout),
		)
		return err
	}
	return nil
}

// PrepareAdmission verifies the checkout restrictions of an organization
// before a membership change takes the organization lock. It adopts a
// payable checkout from before intents existed and verifies an unresolved
// intent, so a verified expiry or failure releases its restriction.
//
// Only an organization that has a Polar customer can hold a checkout, so
// other organizations need no Polar request. A Polar failure while the
// intent is known leaves the stored restriction in place. A failure while no
// intent is known returns the error, because an unrecorded checkout could
// still activate.
func (l *CheckoutLifecycle) PrepareAdmission(ctx context.Context, orgID uuid.UUID) error {
	hasCustomer, err := billingmodels.OrganizationHasPolarCustomer(database.DB(ctx), orgID)
	if err != nil || !hasCustomer {
		return err
	}
	if err := l.AdoptPreexisting(ctx, orgID); err != nil {
		return err
	}
	intent, err := billingmodels.FindUnresolvedOrganizationBillingCheckout(database.DB(ctx), orgID)
	if err != nil || intent == nil {
		return err
	}
	// The stored intent keeps its restriction when Polar cannot answer.
	_, _ = l.Verify(ctx, intent)
	return nil
}

// CheckCheckoutAdmission rejects a membership change that would leave more
// members than an unresolved checkout selected. Call it under the
// organization lock, which also serializes checkout claims. An ambiguous or
// creating intent counts as payable: its checkout can still exist.
func CheckCheckoutAdmission(tx *gorm.DB, orgID uuid.UUID, members int) error {
	intent, err := billingmodels.FindUnresolvedOrganizationBillingCheckout(tx, orgID)
	if err != nil || intent == nil {
		return err
	}
	if !intent.AdmitsMembers(members) {
		return ErrCheckoutSeatLimit
	}
	return nil
}

// CheckoutSnapshot is the part of a Polar checkout that an intent stores.
func CheckoutSnapshot(checkout *CheckoutSession) billingmodels.PolarCheckoutSnapshot {
	return billingmodels.PolarCheckoutSnapshot{
		CheckoutID: checkout.ID,
		URL:        checkout.URL,
		Status:     string(checkout.Status),
		ExpiresAt:  checkout.ExpiresAt,
	}
}

// verifyUndelivered handles an intent whose create response never arrived.
// The intent ID in the checkout metadata finds a checkout that Polar created.
// Without a match, the intent is released only after the grace window.
func (l *CheckoutLifecycle) verifyUndelivered(ctx context.Context, intent *billingmodels.OrganizationBillingCheckout) (CheckoutVerification, error) {
	checkouts, err := l.client.ListCheckouts(ctx, intent.OrganizationID.String(), intent.PolarProductID)
	if err != nil {
		return CheckoutVerification{}, fmt.Errorf("list Polar checkouts: %w", err)
	}
	for i := range checkouts {
		if checkouts[i].IntentID() == intent.ID.String() {
			return l.record(ctx, intent, &checkouts[i])
		}
	}
	if l.now().Sub(intent.CreatedAt) < CheckoutIntentGraceWindow {
		return CheckoutVerification{Outcome: CheckoutUnconfirmed}, nil
	}
	err = intent.Release(database.DB(context.WithoutCancel(ctx)), "no matching Polar checkout after the grace window")
	return resolvedOrChanged(err)
}

// record stores what Polar reports about the checkout of an intent.
func (l *CheckoutLifecycle) record(ctx context.Context, intent *billingmodels.OrganizationBillingCheckout, checkout *CheckoutSession) (CheckoutVerification, error) {
	db := database.DB(context.WithoutCancel(ctx))
	snapshot := CheckoutSnapshot(checkout)
	if checkout.Status.Payable() {
		if err := intent.MarkOpen(db, snapshot); err != nil {
			return changedOrError(err)
		}
		return CheckoutVerification{Outcome: CheckoutPayable, Checkout: checkout}, nil
	}
	if checkout.Status == CheckoutStatusSucceeded {
		return l.recordPaid(ctx, intent, snapshot)
	}
	state, _ := checkout.Status.IntentState()
	return resolvedOrChanged(intent.Resolve(db, state, &snapshot))
}

// recordPaid handles a paid checkout. Its subscription may not be applied
// yet: a webhook can be lost or arrive late. The subscription is read from
// Polar and applied first, and the intent is resolved only when the stored
// plan holds the live subscription or Polar reports that none is live.
func (l *CheckoutLifecycle) recordPaid(ctx context.Context, intent *billingmodels.OrganizationBillingCheckout, snapshot billingmodels.PolarCheckoutSnapshot) (CheckoutVerification, error) {
	orgID := intent.OrganizationID
	status, err := VerifyOrganizationSubscription(ctx, orgID)
	if err != nil {
		return CheckoutVerification{}, fmt.Errorf("verify the Polar subscription of a paid checkout: %w", err)
	}
	if status == "active" {
		if err := SyncOrganizationSubscription(ctx, database.DB(ctx), orgID); err != nil {
			return CheckoutVerification{}, fmt.Errorf("apply the Polar subscription of a paid checkout: %w", err)
		}
	}
	plan, err := billingmodels.FindOrganizationBillingPlan(database.DB(ctx), orgID)
	if err != nil {
		return CheckoutVerification{}, err
	}
	live := plan.HasLivePolarSubscription(l.now())
	if status == "active" && !live {
		// Polar has a live subscription that is not stored yet. The intent
		// keeps its restriction until the subscription is applied.
		return CheckoutVerification{Outcome: CheckoutSubscriptionLive}, nil
	}
	if status != "active" && live {
		// The stored plan still claims a live subscription. A later
		// webhook or sync decides it.
		return CheckoutVerification{Outcome: CheckoutSubscriptionLive}, nil
	}
	verification, err := resolvedOrChanged(intent.Resolve(database.DB(context.WithoutCancel(ctx)), billingmodels.BillingCheckoutStateSucceeded, &snapshot))
	if err != nil || !live {
		return verification, err
	}
	return CheckoutVerification{Outcome: CheckoutSubscriptionLive}, nil
}

// createIsStale reports a creating intent whose request is gone. The server
// can stop between the claim and the end of the Polar call.
func (l *CheckoutLifecycle) createIsStale(intent *billingmodels.OrganizationBillingCheckout) bool {
	return intent.State == billingmodels.BillingCheckoutStateCreating && l.now().Sub(intent.CreatedAt) >= CheckoutIntentGraceWindow
}

// resolvedOrChanged treats a lost race as a change. The winner changed the
// intent, and the caller reads the new state.
func resolvedOrChanged(err error) (CheckoutVerification, error) {
	if err == nil {
		return CheckoutVerification{Outcome: CheckoutResolved}, nil
	}
	return changedOrError(err)
}

func changedOrError(err error) (CheckoutVerification, error) {
	if errors.Is(err, billingmodels.ErrBillingCheckoutStateConflict) {
		return CheckoutVerification{Outcome: CheckoutChanged}, nil
	}
	return CheckoutVerification{}, err
}
