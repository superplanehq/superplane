// Package seatremoval schedules an Admin seat removal for the next renewal.
// It never reduces seats unless an Admin operation asks for that removal.
// Recovery may clear a removal. It never buys seats and never removes a member.
package seatremoval

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/ee/billing/seatquantity"
	"github.com/superplanehq/superplane/pkg/billing"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	ReasonRenewalBoundary = "renewal_boundary"
	ReasonMemberFloor     = "member_floor"
	ReasonPaymentBlocked  = "payment_blocked"
	ReasonMixedPending    = "mixed_pending"
	ReasonStalePreview    = "stale_preview"
	ReasonOutcomeUnknown  = "outcome_unknown"
	ReasonInvalidTarget   = "invalid_target"
	ReasonKeyReused       = "key_reused"
	ReasonActorRequired   = "actor_required"
	ReasonNotAvailable    = "not_available"
	ReasonRepairRequired  = "repair_required"
	ReasonKeyInvalid      = "key_invalid"
)

// StatusError is an actionable removal result. It does not include a provider secret.
type StatusError struct {
	Reason  string
	Message string
}

func (e *StatusError) Error() string { return e.Message }

func statusError(reason, message string) error {
	return &StatusError{Reason: reason, Message: message}
}

// SubscriptionClient is the provider surface a removal uses.
type SubscriptionClient interface {
	GetSubscription(ctx context.Context, subscriptionID string) (*polar.SubscriptionData, error)
	ScheduleSubscriptionSeats(ctx context.Context, subscriptionID string, seats int) (*polar.SubscriptionData, error)
}

// Request is one Admin confirmation of a preview.
type Request struct {
	OrganizationID     uuid.UUID
	ActorUserID        *uuid.UUID
	ActorAPIKeyID      *uuid.UUID
	OperationKey       string
	Target             int
	SubscriptionID     string
	PeriodStart        time.Time
	PeriodEnd          time.Time
	SourcePaidQuantity int
	SeatChangeVersion  int64
}

// Result is the operation a confirmation produced or replayed.
type Result struct {
	Operation *billingmodels.SeatQuantityOperation
	View      billing.SeatQuantityView
}

// Runner performs removal mutations. Now is replaceable in tests.
type Runner struct {
	DB     *gorm.DB
	Client SubscriptionClient
	Now    func() time.Time
}

func NewRunner(db *gorm.DB, client SubscriptionClient) *Runner {
	return &Runner{DB: db, Client: client, Now: time.Now}
}

func (r *Runner) now() time.Time {
	if r.Now == nil {
		return time.Now()
	}
	return r.Now()
}

// Preview describes a removal and does not write.
func (r *Runner) Preview(ctx context.Context, orgID uuid.UUID, target int) (billing.SeatQuantityPreview, error) {
	db := r.DB.WithContext(ctx)
	plan, err := billingmodels.FindOrganizationBillingPlan(db, orgID)
	if err != nil {
		return billing.SeatQuantityPreview{}, err
	}
	var op *billingmodels.SeatQuantityOperation
	if plan != nil && plan.SeatQuantityOperationID != nil {
		op, err = billingmodels.FindSeatQuantityOperation(db, orgID, *plan.SeatQuantityOperationID)
		if err != nil {
			return billing.SeatQuantityPreview{}, err
		}
	}
	members, err := humanMembers(db, orgID)
	if err != nil {
		return billing.SeatQuantityPreview{}, err
	}
	return preview(plan, op, members, target, r.now()), nil
}

// Schedule registers the first removal for the next renewal.
func (r *Runner) Schedule(ctx context.Context, req Request) (Result, error) {
	return r.mutate(ctx, billingmodels.SeatQuantityActionSchedule, req)
}

// Change replaces a verified removal. A target equal to the paid seats cancels it.
func (r *Runner) Change(ctx context.Context, req Request) (Result, error) {
	return r.mutate(ctx, billingmodels.SeatQuantityActionChange, req)
}

// Cancel clears the verified removal. The paid seats do not change.
func (r *Runner) Cancel(ctx context.Context, req Request) (Result, error) {
	return r.mutate(ctx, billingmodels.SeatQuantityActionCancel, req)
}

func (r *Runner) mutate(ctx context.Context, action string, req Request) (Result, error) {
	if err := validateActor(req); err != nil {
		return Result{}, err
	}
	if err := validateOperationKey(req.OperationKey); err != nil {
		return Result{}, err
	}
	var result Result
	err := billingmodels.WithSeatQuantityLock(r.DB.WithContext(ctx), req.OrganizationID, func(*gorm.DB) error {
		var err error
		result, err = r.mutateLocked(ctx, action, req)
		return err
	})
	return result, err
}

func (r *Runner) mutateLocked(ctx context.Context, action string, req Request) (Result, error) {
	var result Result
	var prepared mutation
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, req.OrganizationID); err != nil {
			return err
		}
		plan, members, err := loadPlan(tx, req.OrganizationID)
		if err != nil {
			return err
		}
		resolved, err := resolveAction(action, plan, req.Target)
		if err != nil {
			return err
		}
		action = resolved
		if replay, err := replay(tx, req, action); err != nil || replay != nil {
			if replay != nil {
				result.Operation = replay
				result.View, err = Describe(tx, req.OrganizationID, r.now())
			}
			return err
		}
		if err := validateLocalTx(tx, plan, members, action, req, r.now()); err != nil {
			return err
		}
		prepared = mutation{Action: action, Plan: plan, Members: members, Request: req}
		return nil
	})
	if err != nil || result.Operation != nil {
		return result, err
	}

	current, err := r.Client.GetSubscription(ctx, prepared.Request.SubscriptionID)
	if err != nil {
		return Result{}, providerReadError(err)
	}
	paid := *prepared.Plan.PurchasedSeats
	if err := validateProvider(current, paid, prepared.Request); err != nil {
		return Result{}, err
	}

	var staged *billingmodels.SeatQuantityOperation
	var prior seatState
	err = r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, req.OrganizationID); err != nil {
			return err
		}
		plan, members, err := loadPlan(tx, req.OrganizationID)
		if err != nil {
			return err
		}
		if err := validateLocalTx(tx, plan, members, prepared.Action, req, r.now()); err != nil {
			return err
		}
		prior = capture(plan)
		staged, err = register(tx, plan, prepared.Action, req)
		return err
	})
	if err != nil {
		return Result{}, err
	}

	if !seatquantity.AllowsMutation(r.now(), staged.EffectiveAt) {
		if err := r.reject(ctx, staged, prior, ReasonRenewalBoundary); err != nil {
			return Result{}, err
		}
		return Result{}, statusError(ReasonRenewalBoundary, "The billing period renews soon. Confirm the seat removal after the renewal.")
	}
	if err := staged.MarkDispatched(r.DB.WithContext(ctx), r.now()); err != nil {
		return Result{}, err
	}
	providerTarget := staged.TargetQuantity
	updated, updateErr := r.Client.ScheduleSubscriptionSeats(ctx, staged.SubscriptionID, providerTarget)
	if err := r.record(ctx, staged, prior, updated, updateErr); err != nil {
		return Result{}, err
	}
	if err := r.enforceFloorLocked(ctx, req.OrganizationID, updated); err != nil {
		return Result{}, err
	}
	if err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stored, err := billingmodels.FindSeatQuantityOperation(tx, req.OrganizationID, staged.ID)
		if err != nil {
			return err
		}
		result.Operation = stored
		result.View, err = Describe(tx, req.OrganizationID, r.now())
		return err
	}); err != nil {
		return Result{}, err
	}
	if updated == nil && updateErr != nil && !polar.IsOutcomeUnknown(updateErr) {
		return result, providerWriteError(updateErr)
	}
	return result, nil
}

type mutation struct {
	Action  string
	Plan    *billingmodels.OrganizationBillingPlan
	Members int
	Request Request
}

type seatState struct {
	Version     int64
	Admission   *int
	Proved      *int
	OperationID *uuid.UUID
	Bound       *time.Time
}

func capture(plan *billingmodels.OrganizationBillingPlan) seatState {
	return seatState{
		Version: plan.SeatChangeVersion, Admission: cloneInt(plan.SeatAdmissionLimit),
		Proved: cloneInt(plan.SeatProvedTarget), OperationID: cloneUUID(plan.SeatQuantityOperationID),
		Bound: cloneTime(plan.SeatBoundPeriodEnd),
	}
}

func register(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, action string, req Request) (*billingmodels.SeatQuantityOperation, error) {
	paid := *plan.PurchasedSeats
	target := req.Target
	if action == billingmodels.SeatQuantityActionCancel {
		target = paid
	}
	predecessor, err := billingmodels.FindScheduledSeatQuantityOperation(tx, plan.OrganizationID)
	if err != nil {
		return nil, err
	}
	currentLimit := paid
	if plan.SeatAdmissionLimit != nil && *plan.SeatAdmissionLimit < currentLimit {
		currentLimit = *plan.SeatAdmissionLimit
	}
	plan.SeatChangeVersion++
	if action != billingmodels.SeatQuantityActionCancel && target < currentLimit {
		plan.SeatAdmissionLimit = &target
	}
	op := &billingmodels.SeatQuantityOperation{
		OrganizationID: plan.OrganizationID, ActorUserID: req.ActorUserID, ActorAPIKeyID: req.ActorAPIKeyID,
		Origin: billingmodels.SeatQuantityOriginAdmin, Action: action, SubscriptionID: req.SubscriptionID,
		PeriodStart: req.PeriodStart.UTC(), PeriodEnd: req.PeriodEnd.UTC(), SourcePaidQuantity: paid,
		TargetQuantity: target, EffectiveAt: req.PeriodEnd.UTC(), OperationKey: req.OperationKey,
		RequestFingerprint: billingmodels.SeatQuantityFingerprint(action, target, req.SubscriptionID, req.PeriodEnd, req.SeatChangeVersion, req.SourcePaidQuantity),
		Revision:           1, Status: billingmodels.SeatQuantityStatusSubmitting, SeatChangeVersion: plan.SeatChangeVersion,
	}
	if predecessor != nil {
		op.PredecessorID = &predecessor.ID
	}
	if err := op.Create(tx); err != nil {
		return nil, err
	}
	plan.SeatQuantityOperationID = &op.ID
	bound := req.PeriodEnd.UTC()
	plan.SeatBoundPeriodEnd = &bound
	if err := billingmodels.SaveSeatChange(tx, plan); err != nil {
		return nil, err
	}
	return op, nil
}

func (r *Runner) record(ctx context.Context, staged *billingmodels.SeatQuantityOperation, prior seatState, updated *polar.SubscriptionData, updateErr error) error {
	if updateErr != nil && !acceptedError(updateErr) {
		return r.markUnknown(ctx, staged)
	}
	if updateErr != nil {
		return r.reject(ctx, staged, prior, ReasonNotAvailable)
	}
	if !acceptedSnapshot(staged, updated) {
		return r.markUnknown(ctx, staged)
	}
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, staged.OrganizationID); err != nil {
			return err
		}
		plan, err := billingmodels.FindOrganizationBillingPlan(tx, staged.OrganizationID)
		if err != nil {
			return err
		}
		if err := supersedeOpenPredecessor(tx, staged); err != nil {
			return err
		}
		status := billingmodels.SeatQuantityStatusScheduled
		if staged.Action == billingmodels.SeatQuantityActionCancel || staged.Action == billingmodels.SeatQuantityActionProtectiveCancel {
			status = billingmodels.SeatQuantityStatusCanceled
		}
		if err := staged.Transition(tx, status, "", nil); err != nil {
			return err
		}
		if status == billingmodels.SeatQuantityStatusCanceled {
			billingmodels.ClearSeatRestriction(plan)
		} else {
			target := staged.TargetQuantity
			plan.SeatProvedTarget = &target
			plan.SeatAdmissionLimit = &target
			plan.SeatQuantityOperationID = &staged.ID
			end := staged.PeriodEnd.UTC()
			plan.SeatBoundPeriodEnd = &end
		}
		return billingmodels.SaveSeatChange(tx, plan)
	})
}

func (r *Runner) reject(ctx context.Context, staged *billingmodels.SeatQuantityOperation, prior seatState, cause string) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, staged.OrganizationID); err != nil {
			return err
		}
		plan, err := billingmodels.FindOrganizationBillingPlan(tx, staged.OrganizationID)
		if err != nil {
			return err
		}
		if err := staged.Transition(tx, billingmodels.SeatQuantityStatusRejected, cause, nil); err != nil {
			return err
		}
		plan.SeatAdmissionLimit = prior.Admission
		plan.SeatProvedTarget = prior.Proved
		plan.SeatQuantityOperationID = prior.OperationID
		plan.SeatBoundPeriodEnd = prior.Bound
		return billingmodels.SaveSeatChange(tx, plan)
	})
}

func (r *Runner) markUnknown(ctx context.Context, staged *billingmodels.SeatQuantityOperation) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, staged.OrganizationID); err != nil {
			return err
		}
		return staged.Transition(tx, billingmodels.SeatQuantityStatusUnknown, "", nil)
	})
}

func replay(tx *gorm.DB, req Request, action string) (*billingmodels.SeatQuantityOperation, error) {
	stored, err := billingmodels.FindSeatQuantityOperationByKey(tx, req.OrganizationID, req.OperationKey)
	if err != nil || stored == nil {
		return nil, err
	}
	target := req.Target
	if action == billingmodels.SeatQuantityActionCancel {
		target = req.SourcePaidQuantity
	}
	fingerprint := billingmodels.SeatQuantityFingerprint(action, target, req.SubscriptionID, req.PeriodEnd, req.SeatChangeVersion, req.SourcePaidQuantity)
	if stored.RequestFingerprint != fingerprint {
		return nil, statusError(ReasonKeyReused, "This operation key was used for another seat removal. Refresh and try again.")
	}
	return stored, nil
}

func resolveAction(action string, plan *billingmodels.OrganizationBillingPlan, target int) (string, error) {
	if plan == nil || plan.PurchasedSeats == nil {
		return "", statusError(ReasonNotAvailable, "Seat removal needs an active Team subscription.")
	}
	if action == billingmodels.SeatQuantityActionChange && target == *plan.PurchasedSeats {
		return billingmodels.SeatQuantityActionCancel, nil
	}
	return action, nil
}

func validateLocalTx(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, members int, action string, req Request, now time.Time) error {
	if err := validateSubscription(plan, req, now); err != nil {
		return err
	}
	inFlight, err := billingmodels.FindInFlightSeatQuantityOperation(tx, plan.OrganizationID)
	if err != nil {
		return err
	}
	if inFlight != nil {
		return statusError(ReasonOutcomeUnknown, "A seat change is still unconfirmed. Wait for billing to finish checking it.")
	}
	repair, err := findRepair(tx, plan.OrganizationID)
	if err != nil {
		return err
	}
	if repair != nil {
		return statusError(ReasonRepairRequired, "Seat billing needs an admin to repair it. Remove members or buy seats. SuperPlane did not remove a member.")
	}
	return validateTargets(plan, members, action, req.Target)
}

func validateSubscription(plan *billingmodels.OrganizationBillingPlan, req Request, now time.Time) error {
	if plan == nil || !plan.Catalog.PerSeat || plan.PlanSource != billingmodels.BillingPlanSourcePolar ||
		plan.PolarSubscriptionID == nil || *plan.PolarSubscriptionID == "" || plan.PurchasedSeats == nil || *plan.PurchasedSeats < 1 {
		return statusError(ReasonNotAvailable, "Seat removal needs an active Team subscription.")
	}
	if plan.PolarSubscriptionStatus != billingmodels.PolarSubscriptionStatusActive {
		if billingmodels.PolarSubscriptionIsPaymentBlocked(plan.PolarSubscriptionStatus) {
			return statusError(ReasonPaymentBlocked, "Update the payment method before you change seats.")
		}
		return statusError(ReasonNotAvailable, "Seat removal needs an active Team subscription.")
	}
	if plan.CancelAtPeriodEnd {
		return statusError(ReasonNotAvailable, "Resume the subscription before you change seats.")
	}
	if plan.CurrentPeriodStart == nil || plan.CurrentPeriodEnd == nil {
		return statusError(ReasonNotAvailable, "The billing period is not known. Sync billing and try again.")
	}
	if !seatquantity.AllowsMutation(now, *plan.CurrentPeriodEnd) {
		return statusError(ReasonRenewalBoundary, "The billing period renews soon. Confirm the seat removal after the renewal.")
	}
	if req.SubscriptionID != *plan.PolarSubscriptionID || !req.PeriodStart.Equal(*plan.CurrentPeriodStart) ||
		!req.PeriodEnd.Equal(*plan.CurrentPeriodEnd) || req.SourcePaidQuantity != *plan.PurchasedSeats ||
		req.SeatChangeVersion != plan.SeatChangeVersion {
		return statusError(ReasonStalePreview, "The subscription or the seat removal changed. Review it and confirm again.")
	}
	return nil
}

func validateTargets(plan *billingmodels.OrganizationBillingPlan, members int, action string, target int) error {
	paid := *plan.PurchasedSeats
	floor := seatquantity.MemberFloor(members)
	scheduled := plan.SeatProvedTarget != nil
	switch action {
	case billingmodels.SeatQuantityActionSchedule:
		if scheduled {
			return statusError(ReasonStalePreview, "A seat removal is already scheduled. Change or cancel it.")
		}
		if target < floor || target >= paid {
			return statusError(ReasonInvalidTarget, "Choose a seat count from the current members up to, but not including, the paid seats.")
		}
	case billingmodels.SeatQuantityActionChange:
		if !scheduled {
			return statusError(ReasonStalePreview, "There is no seat removal to change.")
		}
		if target < floor || target >= paid {
			return statusError(ReasonInvalidTarget, "Choose a seat count from the current members up to, but not including, the paid seats.")
		}
		if *plan.SeatProvedTarget == target {
			return statusError(ReasonInvalidTarget, "That seat count is already scheduled for the next renewal.")
		}
	case billingmodels.SeatQuantityActionCancel:
		if !scheduled {
			return statusError(ReasonStalePreview, "There is no seat removal to cancel.")
		}
	default:
		return statusError(ReasonInvalidTarget, "This seat change is not available.")
	}
	return nil
}

func validateProvider(snap *polar.SubscriptionData, paid int, req Request) error {
	if snap == nil || snap.ID != req.SubscriptionID {
		return statusError(ReasonNotAvailable, "SuperPlane could not confirm the subscription with Polar. Try again.")
	}
	if snap.PendingUpdate != nil && snap.PendingUpdate.HasUnrelatedChanges() {
		return statusError(ReasonMixedPending, "Polar has another scheduled change. Resolve that change before you change seats.")
	}
	if billingmodels.PolarSubscriptionIsPaymentBlocked(snap.Status) {
		return statusError(ReasonPaymentBlocked, "Update the payment method before you change seats.")
	}
	if snap.CancelAtPeriodEnd || !strings.EqualFold(snap.Status, billingmodels.PolarSubscriptionStatusActive) {
		return statusError(ReasonNotAvailable, "Seat removal needs an active Team subscription.")
	}
	if snap.Seats == nil || *snap.Seats != paid {
		return statusError(ReasonStalePreview, "The seat count changed. Review the removal and confirm again.")
	}
	if snap.CurrentPeriodEnd.Time.IsZero() || !snap.CurrentPeriodEnd.Time.Equal(req.PeriodEnd) {
		return statusError(ReasonStalePreview, "The billing period changed. Review the removal and confirm again.")
	}
	return nil
}

func acceptedSnapshot(staged *billingmodels.SeatQuantityOperation, snap *polar.SubscriptionData) bool {
	if snap == nil || snap.ID != staged.SubscriptionID || snap.Seats == nil || *snap.Seats != staged.SourcePaidQuantity {
		return false
	}
	if snap.PendingUpdate != nil && snap.PendingUpdate.HasUnrelatedChanges() {
		return false
	}
	if !snap.CurrentPeriodEnd.Time.Equal(staged.PeriodEnd) {
		return false
	}
	if staged.Action == billingmodels.SeatQuantityActionCancel || staged.Action == billingmodels.SeatQuantityActionProtectiveCancel {
		return snap.PendingUpdate == nil || snap.PendingUpdate.Seats == nil || *snap.PendingUpdate.Seats >= staged.SourcePaidQuantity
	}
	return snap.PendingUpdate != nil && snap.PendingUpdate.Seats != nil && *snap.PendingUpdate.Seats == staged.TargetQuantity
}

func acceptedError(err error) bool {
	return polar.IsPaymentFailed(err) || polar.IsPaymentActionRequired(err) || polar.IsSubscriptionLocked(err) ||
		polar.IsSubscriptionInactive(err) || polar.IsInvalidRequest(err)
}

func providerReadError(err error) error {
	if polar.IsOutcomeUnknown(err) || polar.IsRateLimited(err) {
		return statusError(ReasonOutcomeUnknown, "SuperPlane could not confirm the subscription with Polar. Try again.")
	}
	return statusError(ReasonNotAvailable, "SuperPlane could not confirm the subscription with Polar. Try again.")
}

func providerWriteError(err error) error {
	if polar.IsPaymentFailed(err) || polar.IsPaymentActionRequired(err) {
		return statusError(ReasonPaymentBlocked, "Polar did not accept the seat change. No seats were removed and nothing was charged.")
	}
	return statusError(ReasonNotAvailable, "Polar did not accept the seat change. No seats were removed and nothing was charged.")
}

func validateActor(req Request) error {
	if req.ActorUserID == nil && req.ActorAPIKeyID == nil {
		return statusError(ReasonActorRequired, "An organization admin must confirm the seat removal.")
	}
	if req.ActorUserID != nil && *req.ActorUserID == uuid.Nil && (req.ActorAPIKeyID == nil || *req.ActorAPIKeyID == uuid.Nil) {
		return statusError(ReasonActorRequired, "An organization admin must confirm the seat removal.")
	}
	return nil
}

func validateOperationKey(key string) error {
	if len(key) < 1 || len(key) > 128 {
		return statusError(ReasonKeyInvalid, "The operation key must have 1 to 128 printable ASCII characters and no spaces.")
	}
	for i := 0; i < len(key); i++ {
		if key[i] < '!' || key[i] > '~' {
			return statusError(ReasonKeyInvalid, "The operation key must have 1 to 128 printable ASCII characters and no spaces.")
		}
	}
	return nil
}

func loadPlan(tx *gorm.DB, orgID uuid.UUID) (*billingmodels.OrganizationBillingPlan, int, error) {
	plan, err := billingmodels.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, 0, err
	}
	members, err := humanMembers(tx, orgID)
	return plan, members, err
}

func humanMembers(tx *gorm.DB, orgID uuid.UUID) (int, error) {
	count, err := coremodels.CountActiveHumanUsersByOrganization(tx, orgID.String())
	return int(count), err
}

func findRepair(tx *gorm.DB, orgID uuid.UUID) (*billingmodels.SeatQuantityOperation, error) {
	var op billingmodels.SeatQuantityOperation
	err := tx.Where("organization_id = ? AND status = ?", orgID, billingmodels.SeatQuantityStatusRepairRequired).First(&op).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &op, nil
}

func supersedeOpenPredecessor(tx *gorm.DB, op *billingmodels.SeatQuantityOperation) error {
	if op.PredecessorID == nil {
		return nil
	}
	predecessor, err := billingmodels.FindSeatQuantityOperation(tx, op.OrganizationID, *op.PredecessorID)
	if err != nil || predecessor == nil || !predecessor.Open() {
		return err
	}
	if err := predecessor.SetSuperseder(tx, op.ID); err != nil {
		return err
	}
	return predecessor.Transition(tx, billingmodels.SeatQuantityStatusSuperseded, "", nil)
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}
