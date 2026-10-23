package organizations

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func AcceptInviteLink(ctx context.Context, authService authorization.Authorization, accountID string, token string) (*structpb.Struct, error) {
	if token == "" {
		return nil, grpcerrors.InvalidArgument(nil, "invite link token is required")
	}

	account, err := models.FindAccountByID(accountID)
	if err != nil {
		return nil, grpcerrors.Unauthenticated(nil, "account not found")
	}

	inviteLink, err := models.FindInviteLinkByToken(database.DB(ctx), token)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "invite link not found")
	}

	if !inviteLink.Enabled {
		return nil, grpcerrors.PermissionDenied(nil, "invite link disabled")
	}

	org, err := models.FindOrganizationByID(inviteLink.OrganizationID.String())
	if err != nil {
		return nil, grpcerrors.NotFound(err, "organization not found")
	}

	statusValue := "joined"
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, org.ID); err != nil {
			return err
		}
		user, findErr := models.FindMaybeDeletedUserByEmailInTransaction(tx, org.ID.String(), account.Email)
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		if findErr == nil && !user.DeletedAt.Valid {
			statusValue = "already_member"
			return nil
		}
		seats, err := entitlements.ResolveSeats(tx, org.ID)
		if err != nil {
			return err
		}
		if limit := seats.JoinLimit(); limit != nil && seats.Occupied >= *limit {
			return grpcerrors.FailedPrecondition(nil, "This organization has reached its seat limit. Contact an organization admin.")
		}
		if user == nil || errors.Is(findErr, gorm.ErrRecordNotFound) {
			user, err = models.CreateUserInTransaction(tx, org.ID, account.ID, account.Email, account.Name)
		} else {
			err = user.RestoreInTransaction(tx)
		}
		if err != nil {
			return err
		}
		return authService.AssignRole(user.ID.String(), models.RoleOrgOperator, org.ID.String(), models.DomainTypeOrganization)
	})
	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return nil, err
		}
		return nil, grpcerrors.Internal(err, "failed to accept invite")
	}

	return inviteLinkAcceptResponse(org.ID.String(), org.Slug, org.Name, statusValue)
}

func inviteLinkAcceptResponse(organizationID, organizationSlug, organizationName, statusValue string) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]interface{}{
		"organization_id":   organizationID,
		"organization_slug": organizationSlug,
		"organization_name": organizationName,
		"status":            statusValue,
	})
}
