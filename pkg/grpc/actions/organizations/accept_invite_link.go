package organizations

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

type memberJoinedPublisher func(messages.OrganizationMemberJoinedMessage) error
type eligibleOwnerLister func(*gorm.DB, string, []string) ([]models.User, error)

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
	return acceptInviteLinkWithUsage(ctx, authService, usageService, func(message messages.OrganizationMemberJoinedMessage) error {
		return message.Publish()
	}, models.ListActiveHumanUsersByIDs, accountID, token)
}

func acceptInviteLinkWithUsage(
	ctx context.Context,
	authService authorization.Authorization,
	usageService usage.Service,
	publish memberJoinedPublisher,
	listOwners eligibleOwnerLister,
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
	tx := database.DB(ctx).Begin()
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

	notifyOrganizationOwnersOfJoinedMember(ctx, authService, publish, listOwners, org, user)

	return inviteLinkAcceptResponse(org.ID.String(), org.Name, statusValue)
}

func notifyOrganizationOwnersOfJoinedMember(
	ctx context.Context,
	authService authorization.Authorization,
	publish memberJoinedPublisher,
	listOwners eligibleOwnerLister,
	organization *models.Organization,
	member *models.User,
) {
	ownerIDs, err := authService.GetOrgUsersForRole(ctx, models.RoleOrgOwner, organization.ID.String())
	if err != nil {
		log.Errorf("failed to resolve organization owners for member join notification: %v", err)
		return
	}

	owners, err := listOwners(database.DB(ctx), organization.ID.String(), ownerIDs)
	if err != nil {
		log.Errorf("failed to load organization owners for member join notification: %v", err)
		return
	}

	for _, owner := range owners {
		message := messages.OrganizationMemberJoinedMessage{
			ToEmail:          owner.GetEmail(),
			OrganizationID:   organization.ID.String(),
			OrganizationName: organization.Name,
			MemberEmail:      member.GetEmail(),
			MemberName:       member.Name,
		}
		if err := publish(message); err != nil {
			log.Errorf("failed to publish member join notification for organization %s and owner %s: %v", organization.ID, owner.ID, err)
		}
	}
}

func inviteLinkAcceptResponse(organizationID, organizationName, statusValue string) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]interface{}{
		"organization_id":   organizationID,
		"organization_name": organizationName,
		"status":            statusValue,
	})
}
