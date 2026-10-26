package polar

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	"github.com/superplanehq/superplane/pkg/billing"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Runtime provides billing checks that require Polar and checkout lifecycle
// state. Core prepares external verification before it takes workflow locks.
type Runtime struct{}

func NewRuntime() billing.Provider { return &Runtime{} }

func (*Runtime) PrepareDeletion(ctx context.Context, orgID uuid.UUID) (billing.DeletionGuard, error) {
	return VerifyOrganizationDeletion(ctx, orgID)
}

func (*Runtime) PrepareAdmission(ctx context.Context, orgID uuid.UUID) error {
	if !billing.Enabled() || !Configured() {
		return nil
	}
	err := NewCheckoutLifecycle(NewClientFromEnv()).PrepareAdmission(ctx, orgID)
	if err != nil {
		log.WithError(err).WithField("organization_id", orgID.String()).Warn("could not verify checkout restrictions before an invitation acceptance")
		return grpcerrors.UnavailableWithReason(billingUnverifiedMessage, billingUnverifiedReason)
	}
	return nil
}

// ValidateMemberAdmission runs under the caller's organization lock and checks
// both the paid capacity and the capacity of every unresolved checkout.
func (*Runtime) ValidateMemberAdmission(tx *gorm.DB, orgID uuid.UUID) error {
	seats, err := entitlements.ResolveSeats(tx, orgID)
	if err != nil {
		return err
	}
	if seats.PaymentBlocked {
		return grpcerrors.FailedPreconditionWithReason("This organization cannot add members until its payment is updated. Ask an organization admin.", "PAYMENT_PAST_DUE")
	}
	if limit := entitlements.JoinLimit(seats); limit != nil && seats.Occupied >= *limit {
		if seats.PerSeat {
			return grpcerrors.FailedPreconditionWithReason("No seat is available. Ask an organization admin to free or buy a seat.", "NO_SEAT_AVAILABLE")
		}
		return grpcerrors.FailedPreconditionWithReason("This organization reached its member limit. Ask an organization admin.", "MEMBER_LIMIT_REACHED")
	}
	if !billing.Enabled() {
		return nil
	}
	err = CheckCheckoutAdmission(tx, orgID, seats.Occupied+1)
	if errors.Is(err, ErrCheckoutSeatLimit) {
		return grpcerrors.FailedPreconditionWithReason(checkoutSeatLimitMessage, checkoutSeatLimitReason)
	}
	return err
}

// ValidateInvitationCapacity preserves pending reservations and the core limit
// for installations whose billing plan does not cap invitations.
func (*Runtime) ValidateInvitationCapacity(tx *gorm.DB, orgID uuid.UUID, alreadyPending bool, pendingLimit int) error {
	seats, err := entitlements.ResolveSeats(tx, orgID)
	if err != nil {
		return err
	}
	if seats.PaymentBlocked {
		return grpcerrors.FailedPreconditionWithReason("The last renewal payment failed. Update the payment method before you invite members.", "PAYMENT_PAST_DUE")
	}
	if seats.PerSeat {
		if alreadyPending || *seats.Invitable > 0 {
			return nil
		}
		if seats.Purchasable != nil && *seats.Purchasable == 0 {
			return grpcerrors.FailedPreconditionWithReason("This organization has reached its plan seat maximum.", "PLAN_MAXIMUM_REACHED")
		}
		return grpcerrors.FailedPreconditionWithReason("Purchase a seat before sending another invitation.", "SEAT_PURCHASE_REQUIRED")
	}
	if seats.MemberLimit != nil && seats.Occupied >= *seats.MemberLimit {
		return grpcerrors.FailedPreconditionWithReason("This organization has reached its member limit.", "MEMBER_LIMIT_REACHED")
	}
	if seats.MemberLimit != nil || alreadyPending {
		return nil
	}
	pending, err := models.CountPendingOrganizationInvitations(tx, orgID)
	if err != nil {
		return err
	}
	if pending >= int64(pendingLimit) {
		return grpcerrors.FailedPreconditionWithReason("This organization has too many pending invitations. Cancel some before you send more.", "PENDING_INVITATION_LIMIT_REACHED")
	}
	return nil
}

func (*Runtime) RecheckSeatCapacity(tx *gorm.DB, orgID uuid.UUID, cause string) (billing.CapacityAlert, error) {
	check, err := RecheckSeatCapacity(tx, orgID, SeatCapacityCause(cause))
	if err != nil {
		return nil, err
	}
	return SeatCapacityAlerts{check}, nil
}

func (*Runtime) AssertOrganizationPaymentMethod(tx *gorm.DB, orgID uuid.UUID) error {
	return AssertOrganizationPaymentMethod(tx, orgID)
}

const (
	checkoutSeatLimitReason  = "CHECKOUT_SEAT_LIMIT"
	checkoutSeatLimitMessage = "This organization is changing its plan. The new plan has no seat for you yet. Ask an organization admin to finish the plan change and buy a seat for you."
	billingUnverifiedReason  = "BILLING_UNVERIFIED"
	billingUnverifiedMessage = "SuperPlane could not check the billing state of this organization. Try again later."
)
