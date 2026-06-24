package organizations

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func AcceptInviteLink(ctx context.Context, authService authorization.Authorization, accountID string, token string) (*structpb.Struct, error) {
	return AcceptInviteLinkWithUsage(ctx, authService, nil, accountID, token)
}

func AcceptInviteLinkWithUsage(
	ctx context.Context,
	authService authorization.Authorization,
	usageService usage.Service,
	accountID string,
	token string,
) (*structpb.Struct, error) {
	if token == "" {
		return nil, grpcerrors.InvalidArgument(nil, "invite link token is required")
	}

	account, err := models.FindAccountByID(accountID)
	if err != nil {
		return nil, grpcerrors.Unauthenticated(nil, "account not found")
	}

	inviteLink, err := models.FindInviteLinkByToken(token)
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
	tx := database.Conn().Begin()
	user, err := models.FindMaybeDeletedUserByEmailInTransaction(tx, org.ID.String(), account.Email)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return nil, grpcerrors.Internal(err, "failed to accept invite")
		}

		userCount, countErr := models.CountActiveHumanUsersByOrganizationInTransaction(tx, org.ID.String())
		if countErr != nil {
			tx.Rollback()
			return nil, grpcerrors.Internal(countErr, "failed to accept invite")
		}

		if err := usage.EnsureOrganizationWithinLimits(ctx, usageService, org.ID.String(), &usagepb.OrganizationState{
			Users: int32(userCount + 1),
		}, nil); err != nil {
			tx.Rollback()
			return nil, err
		}

		user, err = models.CreateUserInTransaction(tx, org.ID, account.ID, account.Email, account.Name)
		if err != nil {
			tx.Rollback()
			return nil, grpcerrors.Internal(err, "failed to accept invite")
		}
	} else if !user.DeletedAt.Valid {
		tx.Rollback()
		statusValue = "already_member"
		return inviteLinkAcceptResponse(org.ID.String(), org.Name, statusValue)
	} else {
		userCount, countErr := models.CountActiveHumanUsersByOrganizationInTransaction(tx, org.ID.String())
		if countErr != nil {
			tx.Rollback()
			return nil, grpcerrors.Internal(countErr, "failed to accept invite")
		}

		if err := usage.EnsureOrganizationWithinLimits(ctx, usageService, org.ID.String(), &usagepb.OrganizationState{
			Users: int32(userCount + 1),
		}, nil); err != nil {
			tx.Rollback()
			return nil, err
		}

		err = user.RestoreInTransaction(tx)
		if err != nil {
			tx.Rollback()
			return nil, grpcerrors.Internal(err, "failed to accept invite")
		}
	}

	err = authService.AssignRole(user.ID.String(), models.RoleOrgViewer, org.ID.String(), models.DomainTypeOrganization)
	if err != nil {
		tx.Rollback()
		return nil, grpcerrors.Internal(err, "failed to accept invite")
	}

	if err := tx.Commit().Error; err != nil {
		return nil, grpcerrors.Internal(err, "failed to accept invite")
	}

	return inviteLinkAcceptResponse(org.ID.String(), org.Name, statusValue)
}

func inviteLinkAcceptResponse(organizationID, organizationName, statusValue string) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]interface{}{
		"organization_id":   organizationID,
		"organization_name": organizationName,
		"status":            statusValue,
	})
}
