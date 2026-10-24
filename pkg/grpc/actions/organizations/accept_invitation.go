package organizations

import (
	"context"
	"errors"
	"time"

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
	invitation, err := models.FindOrganizationInvitationByToken(db, token)
	if err != nil {
		return nil, invitationActionError(err)
	}
	if utils.NormalizeEmail(account.Email) != utils.NormalizeEmail(invitation.Email) {
		return nil, grpcerrors.PermissionDenied(nil, "Sign in with the invited email address.")
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
		// Resend and cancellation can change the token while this request waits.
		invitation, err = models.FindOrganizationInvitationByToken(tx, token)
		if err != nil {
			return err
		}
		if utils.NormalizeEmail(account.Email) != utils.NormalizeEmail(invitation.Email) {
			return grpcerrors.PermissionDenied(nil, "Sign in with the invited email address.")
		}
		user, findErr := models.FindMaybeDeletedUserByEmailInTransaction(tx, org.ID.String(), account.Email)
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		activeMember := findErr == nil && !user.DeletedAt.Valid && user.AccountID != nil && *user.AccountID == account.ID
		if invitation.State == models.InvitationStateAccepted {
			if !activeMember || invitation.AcceptedBy == nil || *invitation.AcceptedBy != user.ID {
				return grpcerrors.FailedPrecondition(nil, "This invitation has already been accepted.")
			}
			statusValue = "already_member"
			return nil
		}
		now := time.Now()
		if invitation.EffectiveState(now) != models.InvitationStatePending {
			return grpcerrors.FailedPrecondition(nil, "This invitation is canceled or expired.")
		}
		if activeMember {
			statusValue = "already_member"
		} else {
			seats, err := entitlements.ResolveSeats(tx, org.ID)
			if err != nil {
				return err
			}
			if limit := seats.JoinLimit(); limit != nil && seats.Occupied >= *limit {
				return grpcerrors.FailedPrecondition(nil, "This organization has no available seat. Contact an organization admin.")
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
