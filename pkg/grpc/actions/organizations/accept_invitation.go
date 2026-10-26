package organizations

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

// Accept admits the invited account into the organization.
//
// The transaction locks the account row before the organization row, in the
// same order as account deletion, and then reloads both. A deleted, blocked,
// or changed account is rejected, so a deletion that wins leaves no
// membership, grant, or occupied seat. A deletion that loses finds and
// removes the committed membership.
//
// Acceptance never changes the Polar seat count and never charges. A
// rejected acceptance leaves the invitation pending.
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
		// Polar requests run before any lock. A verified checkout expiry or
		// failure releases its restriction here. The locked check below
		// uses the stored checkout state.
		if err := prepareCheckoutAdmission(ctx, invitation.OrganizationID); err != nil {
			return nil, err
		}
	}
	var org *models.Organization
	statusValue := "joined"
	err = db.Transaction(func(tx *gorm.DB) error {
		locked, err := lockAcceptingAccount(tx, account.ID)
		if err != nil {
			return err
		}
		account = locked
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
			if err := s.requireActiveInviter(ctx, tx, invitation); err != nil {
				return err
			}
			if err := requireSeatForNewMember(tx, org.ID); err != nil {
				return err
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
	return inviteLinkAcceptResponse(org.ID.String(), org.Slug, org.Name, statusValue)
}

// lockAcceptingAccount locks and reloads the account. A deletion that
// committed first makes the account disappear, and a block that committed
// first stops the acceptance.
func lockAcceptingAccount(tx *gorm.DB, accountID uuid.UUID) (*models.Account, error) {
	account, err := models.LockAccount(tx, accountID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcerrors.Unauthenticated(err, "Account not found.")
	}
	if err != nil {
		return nil, err
	}
	if account.IsBlocked() {
		return nil, grpcerrors.PermissionDenied(models.ErrAccountBlocked, models.AccountBlockedMessage)
	}
	return account, nil
}

func requireSeatForNewMember(tx *gorm.DB, orgID uuid.UUID) error {
	return billing.ValidateMemberAdmission(tx, orgID)
}

func prepareCheckoutAdmission(ctx context.Context, orgID uuid.UUID) error {
	return billing.PrepareAdmission(ctx, orgID)
}

// Elevated invitations are valid only while the inviter can still grant them.
func (s *InvitationService) requireActiveInviter(ctx context.Context, tx *gorm.DB, invitation *models.OrganizationInvitation) error {
	if invitation.Role == models.RoleOrgOperator {
		return nil
	}
	orgID := invitation.OrganizationID.String()
	inviterID := invitation.InvitedBy.String()
	revoked := grpcerrors.FailedPreconditionWithReason("Ask an organization admin to send this invitation again.", "INVITATION_INVITER_REVOKED")
	inviter, err := models.FindActiveUserByIDInTransaction(tx, orgID, inviterID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && inviter.DeletedAt.Valid) {
		return revoked
	}
	if err != nil {
		return err
	}
	allowed, err := s.AuthService.CheckOrganizationPermission(ctx, inviterID, orgID, "members", "update")
	if err != nil {
		return grpcerrors.Internal(err, "Failed to check the inviter permissions.")
	}
	if !allowed {
		return revoked
	}
	return nil
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
