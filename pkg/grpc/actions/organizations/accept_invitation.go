package organizations

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func (s *InvitationService) Accept(ctx context.Context, accountID, token string) (*structpb.Struct, error) {
	if token == "" {
		return nil, grpcerrors.InvalidArgument(nil, "An invitation token is required.")
	}
	account, err := models.FindAccountByID(accountID)
	if err != nil {
		return nil, grpcerrors.Unauthenticated(err, "Account not found.")
	}
	if account.IsBlocked() {
		return nil, grpcerrors.PermissionDenied(models.ErrAccountBlocked, models.AccountBlockedMessage)
	}
	db := database.DB(ctx)
	invitation, err := findInvitationForAcceptance(db, token)
	if err != nil {
		return nil, invitationActionError(err)
	}
	if utils.NormalizeEmail(account.Email) != utils.NormalizeEmail(invitation.Email) {
		return nil, invitationEmailMismatch(invitation.Email)
	}
	if invitation.EffectiveState(time.Now()) == models.InvitationStatePending {
		if err := s.validateRole(ctx, invitation.OrganizationID.String(), invitation.Role); err != nil {
			return nil, err
		}
	}
	var org *models.Organization
	statusValue := "joined"
	err = db.Transaction(func(tx *gorm.DB) error {
		var err error
		org, err = models.LockOrganization(tx, invitation.OrganizationID)
		if err != nil {
			return err
		}
		// Resend can rotate the token while this request waits.
		invitation, err = findInvitationForAcceptance(tx, token)
		if err != nil {
			return err
		}
		if utils.NormalizeEmail(account.Email) != utils.NormalizeEmail(invitation.Email) {
			return invitationEmailMismatch(invitation.Email)
		}
		user, findErr := models.FindMaybeDeletedUserByEmailInTransaction(tx, org.ID.String(), account.Email)
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		activeMember := findErr == nil && !user.DeletedAt.Valid && user.AccountID != nil && *user.AccountID == account.ID
		if invitation.State == models.InvitationStateAccepted {
			if !activeMember || invitation.AcceptedBy == nil || *invitation.AcceptedBy != user.ID {
				return grpcerrors.FailedPreconditionWithReason("This invitation was already accepted. Ask an organization admin for a new invitation.", "INVITATION_ALREADY_ACCEPTED")
			}
			statusValue = "already_member"
			return nil
		}
		now := time.Now()
		switch invitation.EffectiveState(now) {
		case models.InvitationStatePending:
		case models.InvitationStateExpired:
			return grpcerrors.FailedPreconditionWithReason("This invitation expired. Ask an organization admin for a new invitation.", "INVITATION_EXPIRED")
		case models.InvitationStateCanceled:
			return grpcerrors.FailedPreconditionWithReason("This invitation was canceled. Ask an organization admin for a new invitation.", "INVITATION_CANCELED")
		default:
			return grpcerrors.FailedPrecondition(nil, "This invitation is unavailable. Ask an organization admin for a new invitation.")
		}
		if activeMember {
			statusValue = "already_member"
		} else {
			seats, err := entitlements.ResolveSeats(tx, org.ID)
			if err != nil {
				return err
			}
			if limit := seats.JoinLimit(); limit != nil && seats.Occupied >= *limit {
				if seats.PerSeat {
					return grpcerrors.FailedPreconditionWithReason("No seat is available. Ask an organization admin to free or buy a seat.", "NO_SEAT_AVAILABLE")
				}
				return grpcerrors.FailedPreconditionWithReason("This organization reached its member limit. Ask an organization admin.", "MEMBER_LIMIT_REACHED")
			}
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				user, err = models.CreateUserInTransaction(tx, org.ID, account.ID, account.Email, account.Name)
			} else {
				if user.AccountID == nil || *user.AccountID != account.ID {
					return grpcerrors.FailedPrecondition(nil, "This email address belongs to another account.")
				}
				err = user.RestoreInTransaction(tx)
			}
			if err != nil {
				return err
			}
			if err := s.AuthService.AssignRoleInTransaction(tx, user.ID.String(), invitation.Role, org.ID.String(), models.DomainTypeOrganization); err != nil {
				return err
			}
		}
		invitation.State = models.InvitationStateAccepted
		invitation.AcceptedAt = &now
		invitation.AcceptedBy = &user.ID
		return invitation.Save(tx)
	})
	if err != nil {
		return nil, invitationActionError(err)
	}
	// A new or restored member occupies a seat, so Polar must renew it.
	if statusValue == "joined" {
		seatrenewal.ScheduleAsync(ctx, org.ID)
	}
	return inviteLinkAcceptResponse(org.ID.String(), org.Slug, org.Name, statusValue)
}

// Only acceptance consults history. A replaced token cannot authorize signup.
func findInvitationForAcceptance(tx *gorm.DB, token string) (*models.OrganizationInvitation, error) {
	invitation, err := models.FindOrganizationInvitationByToken(tx, token)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return invitation, err
	}
	replaced, historyErr := models.HasReplacedOrganizationInvitationToken(tx, token)
	if historyErr != nil {
		return nil, historyErr
	}
	if replaced {
		return nil, grpcerrors.FailedPreconditionWithReason("A new invitation replaced this link. Open the latest invitation email.", "INVITATION_REPLACED")
	}
	return nil, err
}

func invitationEmailMismatch(email string) error {
	local, domain, _ := strings.Cut(email, "@")
	first, _ := utf8.DecodeRuneInString(local)
	masked := string(first) + "***@" + domain
	message := "This invitation is for " + masked + ". Sign out, then sign in with the invited email address."
	return grpcerrors.PermissionDeniedWithReason(message, "INVITATION_EMAIL_MISMATCH", map[string]string{"invited_email": masked})
}
