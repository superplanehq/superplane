package organizations

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

// seatPurchaseAwaitingConfirmation is the API name of the stored unknown status.
const seatPurchaseAwaitingConfirmation = "awaiting_confirmation"

const (
	messageSeatPurchaseNeedsSubscription = "Seat purchases need an active Team subscription."
	messageSeatPurchaseNeedsResume       = "Resume the subscription before buying seats."
	messageSeatPurchaseInFlight          = "A seat purchase is already in progress."
	messageSeatCountChanged              = "Seat count changed. Refresh and try again."
	messageSeatPurchaseKeyMissing        = "Add an operation key to the seat purchase request."
	messageSeatPurchaseKeyInvalid        = "The operation key must have 1 to 128 printable ASCII characters and no spaces."
	messageSeatPurchaseKeyReused         = "This operation key was used for another seat purchase. Refresh and try again."
)

// maxSeatPurchaseOperationKeyLength limits the caller operation key.
const maxSeatPurchaseOperationKeyLength = 128

// seatPurchaseOperation identifies one purchase request of an organization.
// A repeat with the same key and fingerprint is the same purchase.
type seatPurchaseOperation struct {
	key         string
	fingerprint string
}

// seatPurchaseState is the verified subscription state that a seat purchase
// or preview works from.
type seatPurchaseState struct {
	plan           *models.OrganizationBillingPlan
	seats          *entitlements.SeatUsage
	subscriptionID string
	purchased      int
	target         int
	periodStart    time.Time
	periodEnd      time.Time
}

// PreviewSeatPurchase estimates the prorated charge for additional seats.
// It calls Polar only to settle an earlier unconfirmed purchase.
func PreviewSeatPurchase(ctx context.Context, orgID string, req *pb.PreviewSeatPurchaseRequest) (*pb.PreviewSeatPurchaseResponse, error) {
	return previewSeatPurchase(ctx, orgID, req, time.Now)
}

func previewSeatPurchase(ctx context.Context, orgID string, req *pb.PreviewSeatPurchaseRequest, clock seatPurchaseClock) (*pb.PreviewSeatPurchaseResponse, error) {
	organizationID, err := requireSeatPurchaseAvailable(ctx, orgID)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	reconcileSeatPurchaseAndScheduleRenewal(ctx, db, organizationID)

	now := clock()
	state, err := checkSeatPurchase(db, organizationID, req.GetAdditionalSeats(), now)
	if err != nil {
		return nil, err
	}

	priceCents := state.plan.Catalog.PriceCents
	response := &pb.PreviewSeatPurchaseResponse{
		PurchasedSeats:       int32(state.purchased),
		TargetSeats:          int32(state.target),
		SeatPriceCents:       priceCents,
		ProratedChargeCents:  proratedChargeCents(int(req.GetAdditionalSeats()), priceCents, state.periodStart, state.periodEnd, now),
		PeriodStart:          protoTimestamp(&now),
		PeriodEnd:            protoTimestamp(&state.periodEnd),
		NewMonthlyTotalCents: int64(state.target) * priceCents,
		Estimate:             true,
		ExcludesTax:          true,
	}
	if state.seats.Purchasable != nil {
		response.PurchasableSeats = new(int32(*state.seats.Purchasable))
	}
	return response, nil
}

// PurchaseSeats buys additional seats on the Polar subscription. Polar charges
// the prorated amount at once and applies the change only when the payment
// succeeds. Capacity changes only from the Polar snapshot.
//
// The purchase never reaches Polar near the end of the billing period, or after
// the period changed. See seatPurchaseBoundaryWindow.
func PurchaseSeats(ctx context.Context, orgID string, req *pb.PurchaseSeatsRequest, userID string) (*pb.PurchaseSeatsResponse, error) {
	return purchaseSeats(ctx, orgID, req, userID, time.Now)
}

func purchaseSeats(ctx context.Context, orgID string, req *pb.PurchaseSeatsRequest, userID string, clock seatPurchaseClock) (*pb.PurchaseSeatsResponse, error) {
	organizationID, err := requireSeatPurchaseAvailable(ctx, orgID)
	if err != nil {
		return nil, err
	}
	operation, err := newSeatPurchaseOperation(req)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)

	// A repeat of a known operation returns the stored purchase in any
	// billing period. It never reaches Polar, so it cannot charge again,
	// also after a renewal restored the original seat count.
	stored, err := findSeatPurchaseOperation(db, organizationID, operation)
	if err != nil {
		return nil, err
	}
	if stored != nil {
		return seatPurchaseResponse(ctx, organizationID, stored), nil
	}

	// Settle an unfinished purchase before the local checks. Only a snapshot
	// from the period of that purchase can settle it.
	reconcileSeatPurchaseAndScheduleRenewal(ctx, db, organizationID)

	// The purchase sends an absolute seat count, so it must start from the
	// seats Polar reports now. A stale local count could remove seats that
	// were added elsewhere. Requests that cannot take seats are rejected
	// before the Polar read.
	state, err := checkSeatPurchase(db, organizationID, req.GetAdditionalSeats(), clock())
	if err != nil {
		return replaySeatPurchaseBeforeError(ctx, organizationID, operation, err)
	}
	if err := polar.RefreshSeatPurchaseState(ctx, db, polar.NewClientFromEnv(), organizationID, state.subscriptionID); err != nil {
		return replaySeatPurchaseBeforeError(ctx, organizationID, operation, polarBillingError(err, "Could not confirm the seat count with Polar. Try again."))
	}

	// The refresh can apply a renewal. Such a purchase was previewed for
	// another period, and its absolute seat count can be wrong.
	started, err := beginSeatPurchase(ctx, organizationID, req, operation, userID, state, clock)
	if err != nil {
		return nil, err
	}
	if started.replayed {
		// A concurrent request with the same operation recorded the purchase
		// first. That request owns the Polar call.
		return seatPurchaseResponse(ctx, organizationID, started.purchase), nil
	}
	purchase := started.purchase

	// Last check before the charge. Time passed since the locked check.
	// A purchase that cannot start safely never reaches Polar. Nothing was
	// sent, so the recorded attempt ends as failed and does not block the
	// next purchase.
	if err := checkSeatPurchaseBoundary(started.state.periodEnd, clock()); err != nil {
		settleCtx := context.WithoutCancel(ctx)
		if recordErr := recordSeatPurchaseStatus(settleCtx, purchase, models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorRenewalBoundary); recordErr != nil {
			return nil, recordErr
		}
		return nil, err
	}

	// The outcome must be recorded even when the caller disconnects.
	settleCtx := context.WithoutCancel(ctx)
	snapshot, updateErr := polar.NewClientFromEnv().UpdateSubscriptionSeats(settleCtx, purchase.PolarSubscriptionID, purchase.ToSeats)
	if updateErr == nil {
		if err := recordAppliedSeatPurchase(settleCtx, purchase, snapshot); err != nil {
			return nil, err
		}
		// The invoice update replaced any scheduled renewal seat count.
		seatrenewal.ScheduleAsync(settleCtx, organizationID)
		return seatPurchaseResponse(ctx, organizationID, purchase), nil
	}

	status, errorCode := classifySeatUpdateError(updateErr)
	if err := recordSeatPurchaseStatus(settleCtx, purchase, status, errorCode); err != nil {
		return nil, err
	}
	if status == models.SeatPurchaseStatusUnknown {
		log.WithError(updateErr).WithField("organization_id", organizationID.String()).Warn("seat purchase outcome is unknown")
		return seatPurchaseResponse(ctx, organizationID, purchase), nil
	}
	return nil, seatPurchaseError(purchase, updateErr)
}

// newSeatPurchaseOperation validates the operation key of a purchase request
// and computes the fingerprint of the charge that the request asks for.
//
// The key is required. A request without a key could not be replayed safely,
// so no purchase path charges without one. A client from before operation
// keys gets InvalidArgument and no charge until it sends a key.
func newSeatPurchaseOperation(req *pb.PurchaseSeatsRequest) (seatPurchaseOperation, error) {
	key := req.GetOperationKey()
	if key == "" {
		return seatPurchaseOperation{}, grpcerrors.InvalidArgument(nil, messageSeatPurchaseKeyMissing)
	}
	if !validSeatPurchaseOperationKey(key) {
		return seatPurchaseOperation{}, grpcerrors.InvalidArgument(nil, messageSeatPurchaseKeyInvalid)
	}
	return seatPurchaseOperation{
		key:         key,
		fingerprint: models.SeatPurchaseFingerprint(int(req.GetAdditionalSeats()), int(req.GetExpectedPurchasedSeats())),
	}, nil
}

func validSeatPurchaseOperationKey(key string) bool {
	if len(key) > maxSeatPurchaseOperationKeyLength {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] < '!' || key[i] > '~' {
			return false
		}
	}
	return true
}

// findSeatPurchaseOperation returns the purchase that the operation already
// started, or nil when the key is new. A key that started another request
// is rejected, so it cannot buy seats that the caller did not ask for.
func findSeatPurchaseOperation(tx *gorm.DB, orgID uuid.UUID, operation seatPurchaseOperation) (*models.SeatPurchase, error) {
	stored, err := models.FindSeatPurchaseByOperationKey(tx, orgID, operation.key)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not buy seats.")
	}
	if stored == nil {
		return nil, nil
	}
	if !stored.MatchesRequest(operation.fingerprint) {
		return nil, grpcerrors.FailedPrecondition(nil, messageSeatPurchaseKeyReused)
	}
	return stored, nil
}

// A concurrent request can record the operation after the initial lookup.
// Return that purchase instead of a pre-lock error caused by its new state.
func replaySeatPurchaseBeforeError(ctx context.Context, orgID uuid.UUID, operation seatPurchaseOperation, purchaseErr error) (*pb.PurchaseSeatsResponse, error) {
	stored, err := findSeatPurchaseOperation(database.DB(ctx), orgID, operation)
	if err != nil {
		return nil, err
	}
	if stored != nil {
		return seatPurchaseResponse(ctx, orgID, stored), nil
	}
	return nil, purchaseErr
}

func requireSeatPurchaseAvailable(ctx context.Context, orgID string) (uuid.UUID, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return uuid.Nil, err
	}
	if !models.BillingEnabled() || !polar.Configured() {
		return uuid.Nil, grpcerrors.FailedPrecondition(nil, "Billing is not configured.")
	}
	if err := requireNewBilling(ctx, organizationID); err != nil {
		return uuid.Nil, err
	}
	return organizationID, nil
}

// checkSeatPurchase verifies that the subscription can take additional seats.
func checkSeatPurchase(tx *gorm.DB, orgID uuid.UUID, additionalSeats int32, now time.Time) (*seatPurchaseState, error) {
	plan, err := models.ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load the current plan.")
	}
	if !plan.Catalog.PerSeat || plan.PlanSource == models.BillingPlanSourceAdmin ||
		plan.PolarSubscriptionID == nil || *plan.PolarSubscriptionID == "" ||
		plan.PolarSubscriptionStatus != models.PolarSubscriptionStatusActive {
		return nil, grpcerrors.FailedPrecondition(nil, messageSeatPurchaseNeedsSubscription)
	}
	if plan.CancelAtPeriodEnd {
		return nil, grpcerrors.FailedPrecondition(nil, messageSeatPurchaseNeedsResume)
	}
	if plan.CurrentPeriodStart == nil || plan.CurrentPeriodEnd == nil || !plan.CurrentPeriodEnd.After(now) {
		return nil, grpcerrors.FailedPrecondition(nil, "The billing period is not known. Sync billing and try again.")
	}
	if err := checkSeatPurchaseBoundary(*plan.CurrentPeriodEnd, now); err != nil {
		return nil, err
	}
	if additionalSeats < 1 {
		return nil, grpcerrors.InvalidArgument(nil, "Select at least 1 seat to buy.")
	}

	seats, err := entitlements.ResolveSeatsForPlan(tx, orgID, plan)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load seat usage.")
	}
	if !seats.PerSeat || seats.Purchased == nil {
		return nil, grpcerrors.FailedPrecondition(nil, messageSeatPurchaseNeedsSubscription)
	}
	// A nil Purchasable means unlimited. Only the Polar seat tiers limit it then.
	if seats.Purchasable != nil && int(additionalSeats) > *seats.Purchasable {
		return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("The %s plan allows up to %d seats.", plan.Catalog.Name, *seats.PlanMaximum))
	}
	target := int64(*seats.Purchased) + int64(additionalSeats)
	if target > math.MaxInt32 {
		return nil, grpcerrors.InvalidArgument(nil, "Select fewer seats to buy.")
	}

	return &seatPurchaseState{
		plan:           plan,
		seats:          seats,
		subscriptionID: *plan.PolarSubscriptionID,
		purchased:      *seats.Purchased,
		target:         int(target),
		periodStart:    *plan.CurrentPeriodStart,
		periodEnd:      *plan.CurrentPeriodEnd,
	}, nil
}

// proratedChargeCents uses the Polar formula: seat price x added seats x
// remaining seconds / period seconds, rounded to the nearest cent.
func proratedChargeCents(additionalSeats int, seatPriceCents int64, periodStart, periodEnd, now time.Time) int64 {
	periodSeconds := int64(periodEnd.Sub(periodStart) / time.Second)
	remainingSeconds := int64(periodEnd.Sub(now) / time.Second)
	if periodSeconds <= 0 || remainingSeconds <= 0 {
		return 0
	}
	fullCharge := float64(int64(additionalSeats) * seatPriceCents)
	return int64(math.Round(fullCharge * float64(remainingSeconds) / float64(periodSeconds)))
}

// seatPurchaseStart is the purchase that beginSeatPurchase recorded or found.
type seatPurchaseStart struct {
	purchase *models.SeatPurchase
	// state is the locked state that a new purchase works from. It is nil
	// for a replay.
	state *seatPurchaseState
	// replayed marks a purchase that another request with the same
	// operation recorded. Only that request calls Polar.
	replayed bool
}

// beginSeatPurchase checks the request and records a pending purchase under the
// organization lock. The lock is not held during the Polar call. The billing
// period must be the period that the caller saw before the Polar refresh.
//
// The purchase stores the operation, the subscription, and the billing period,
// so a repeat and a later settlement can refer to them. The lock serializes
// requests with the same operation. The unique operation key index catches a
// writer that does not take the lock.
func beginSeatPurchase(ctx context.Context, orgID uuid.UUID, req *pb.PurchaseSeatsRequest, operation seatPurchaseOperation, userID string, seen *seatPurchaseState, clock seatPurchaseClock) (*seatPurchaseStart, error) {
	var started *seatPurchaseStart
	db := database.DB(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, orgID); err != nil {
			return grpcerrors.Internal(err, "Could not buy seats.")
		}
		stored, err := findSeatPurchaseOperation(tx, orgID, operation)
		if err != nil {
			return err
		}
		if stored != nil {
			started = &seatPurchaseStart{purchase: stored, replayed: true}
			return nil
		}

		state, err := checkSeatPurchase(tx, orgID, req.GetAdditionalSeats(), clock())
		if err != nil {
			return err
		}
		if err := checkSeatPurchasePeriodUnchanged(seen, state); err != nil {
			return err
		}
		if int(req.GetExpectedPurchasedSeats()) != state.purchased {
			return grpcerrors.Conflict(nil, messageSeatCountChanged)
		}
		inFlight, err := models.FindInFlightSeatPurchase(tx, orgID)
		if err != nil {
			return grpcerrors.Internal(err, "Could not buy seats.")
		}
		if inFlight != nil {
			return grpcerrors.Conflict(nil, messageSeatPurchaseInFlight)
		}

		purchase := models.NewSeatPurchase(orgID, parseUserID(userID), state.subscriptionID, state.purchased, state.target, &models.SeatPurchaseOperation{
			Key:         operation.key,
			Fingerprint: operation.fingerprint,
			PeriodStart: state.periodStart,
			PeriodEnd:   state.periodEnd,
		})
		if err := purchase.Create(tx); err != nil {
			return grpcerrors.Internal(err, "Could not buy seats.")
		}
		started = &seatPurchaseStart{purchase: purchase, state: state}
		return nil
	})
	if models.IsSeatPurchaseOperationKeyConflict(err) {
		return replaySeatPurchaseOperation(db, orgID, operation)
	}
	if err != nil {
		return nil, err
	}
	return started, nil
}

// replaySeatPurchaseOperation reads the purchase that another request with
// the same operation key recorded first.
func replaySeatPurchaseOperation(tx *gorm.DB, orgID uuid.UUID, operation seatPurchaseOperation) (*seatPurchaseStart, error) {
	stored, err := findSeatPurchaseOperation(tx, orgID, operation)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		return nil, grpcerrors.Internal(errors.New("seat purchase operation conflict without a stored purchase"), "Could not buy seats.")
	}
	return &seatPurchaseStart{purchase: stored, replayed: true}, nil
}

// recordAppliedSeatPurchase applies the snapshot of a confirmed update. Polar has
// already charged, so a local failure leaves the purchase unknown. Later
// reconciliation then applies the Polar snapshot.
func recordAppliedSeatPurchase(ctx context.Context, purchase *models.SeatPurchase, snapshot *polar.SubscriptionData) error {
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, purchase.OrganizationID); err != nil {
			return err
		}
		if err := polar.ApplyOrganizationSubscription(ctx, tx, purchase.OrganizationID, *snapshot); err != nil {
			return err
		}
		return purchase.UpdateStatus(tx, models.SeatPurchaseStatusSucceeded, "")
	})
	if err == nil {
		return nil
	}
	log.WithError(err).WithField("organization_id", purchase.OrganizationID.String()).Error("could not apply the seat purchase snapshot")
	return recordSeatPurchaseStatus(ctx, purchase, models.SeatPurchaseStatusUnknown, "")
}

func recordSeatPurchaseStatus(ctx context.Context, purchase *models.SeatPurchase, status, errorCode string) error {
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, purchase.OrganizationID); err != nil {
			return err
		}
		return purchase.UpdateStatus(tx, status, errorCode)
	})
	if err != nil {
		return grpcerrors.Internal(err, "Could not record the seat purchase.")
	}
	return nil
}

func classifySeatUpdateError(err error) (status, errorCode string) {
	switch {
	case polar.IsPaymentFailed(err):
		return models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorPaymentFailed
	case polar.IsPaymentActionRequired(err):
		return models.SeatPurchaseStatusActionRequired, models.SeatPurchaseErrorPaymentActionRequired
	case polar.IsSubscriptionLocked(err):
		return models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorSubscriptionLocked
	case polar.IsSubscriptionInactive(err):
		return models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorSubscriptionInactive
	case polar.IsOutcomeUnknown(err):
		return models.SeatPurchaseStatusUnknown, ""
	default:
		return models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorRejected
	}
}

// seatPurchaseError maps a rejected Polar update. Capacity is unchanged.
func seatPurchaseError(purchase *models.SeatPurchase, updateErr error) error {
	message := seatPurchaseMessage(purchase)
	switch {
	case purchase.ErrorCode == models.SeatPurchaseErrorSubscriptionLocked:
		return grpcerrors.Conflict(updateErr, message)
	case purchase.ErrorCode != models.SeatPurchaseErrorRejected:
		return grpcerrors.FailedPrecondition(updateErr, message)
	case polar.IsInvalidRequest(updateErr):
		return grpcerrors.InvalidArgument(updateErr, message)
	default:
		return polarBillingError(updateErr, message)
	}
}

// seatPurchaseMessage is the user-facing text for a purchase. It is empty
// after a success.
func seatPurchaseMessage(purchase *models.SeatPurchase) string {
	switch purchase.ErrorCode {
	case models.SeatPurchaseErrorPaymentFailed:
		return "The payment failed. No seats were added."
	case models.SeatPurchaseErrorPaymentActionRequired:
		return "Your bank needs to confirm this payment. Update the payment method and try again."
	case models.SeatPurchaseErrorSubscriptionLocked:
		return "Try again shortly."
	case models.SeatPurchaseErrorSubscriptionInactive:
		return "The subscription cannot add seats. Check that it is active and not set to cancel."
	case models.SeatPurchaseErrorRejected:
		return "Polar did not accept the seat purchase. No seats were added."
	case models.SeatPurchaseErrorNotApplied:
		return "Polar did not apply the seat purchase. No seats were added."
	case models.SeatPurchaseErrorRenewalBoundary:
		return "The billing period renews soon. No seats were added."
	}
	switch purchase.Status {
	case models.SeatPurchaseStatusPending:
		return "The seat purchase is in progress."
	case models.SeatPurchaseStatusUnknown:
		return "Polar has not confirmed the payment yet. Seats are added after Polar confirms it."
	}
	return ""
}

func seatPurchaseToProto(purchase *models.SeatPurchase) *pb.SeatPurchase {
	if purchase == nil {
		return nil
	}
	status := purchase.Status
	if status == models.SeatPurchaseStatusUnknown {
		status = seatPurchaseAwaitingConfirmation
	}
	return &pb.SeatPurchase{
		Id:        purchase.ID.String(),
		Status:    status,
		FromSeats: int32(purchase.FromSeats),
		ToSeats:   int32(purchase.ToSeats),
		Message:   seatPurchaseMessage(purchase),
		CreatedAt: protoTimestamp(&purchase.CreatedAt),
	}
}

// seatPurchaseResponse describes the purchase with fresh billing. A failure to
// describe billing must not hide a purchase that Polar already charged.
func seatPurchaseResponse(ctx context.Context, orgID uuid.UUID, purchase *models.SeatPurchase) *pb.PurchaseSeatsResponse {
	response := &pb.PurchaseSeatsResponse{Purchase: seatPurchaseToProto(purchase)}
	billing, err := DescribeOrganizationBilling(ctx, orgID.String(), &pb.DescribeOrganizationBillingRequest{Id: orgID.String()})
	if err != nil {
		log.WithError(err).WithField("organization_id", orgID.String()).Warn("could not describe billing after a seat purchase")
		return response
	}
	response.Billing = billing
	return response
}

// reconcileSeatPurchase settles an earlier unconfirmed purchase from Polar.
// When Polar cannot confirm it, the purchase stays in flight and blocks a new one.
func reconcileSeatPurchase(ctx context.Context, db *gorm.DB, orgID uuid.UUID) error {
	if err := polar.ReconcileSeatPurchase(ctx, db, polar.NewClientFromEnv(), orgID); err != nil {
		log.WithError(err).WithField("organization_id", orgID.String()).Warn("could not reconcile the seat purchase")
		return err
	}
	return nil
}

func reconcileSeatPurchaseAndScheduleRenewal(ctx context.Context, db *gorm.DB, orgID uuid.UUID) {
	if err := reconcileSeatPurchase(ctx, db, orgID); err != nil {
		return
	}
	seatrenewal.ScheduleAfterSeatPurchaseAsync(ctx, orgID)
}

func parseUserID(userID string) *uuid.UUID {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil
	}
	return &id
}
