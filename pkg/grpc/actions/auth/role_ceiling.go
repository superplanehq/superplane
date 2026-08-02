package auth

import (
	"context"
	"slices"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
)

func ensureCanGrantRole(
	ctx context.Context,
	authService authorization.Authorization,
	requesterID, domainType, domainID, roleName string,
) error {
	permissions, err := authService.GetRolePermissions(ctx, roleName, domainType, domainID)
	if err != nil {
		return grpcerrors.InvalidArgument(err, "role not found")
	}

	return ensureCanGrantPermissions(ctx, authService, requesterID, domainID, permissions)
}

func ensureCanGrantPermissions(
	ctx context.Context,
	authService authorization.Authorization,
	requesterID, domainID string,
	permissions []*authorization.Permission,
) error {
	for _, permission := range permissions {
		if permission == nil {
			continue
		}

		allowed, err := authService.CheckOrganizationPermission(
			ctx,
			requesterID,
			domainID,
			permission.Resource,
			permission.Action,
		)
		if err != nil {
			return grpcerrors.Internal(err, "failed to check permissions")
		}
		if !allowed {
			return grpcerrors.PermissionDenied(nil, "cannot assign a role with permissions you do not have")
		}
	}

	return nil
}

func ensureNotDemotingLastOwner(
	ctx context.Context,
	authService authorization.Authorization,
	orgID, targetUserID, newRoleName string,
) error {
	if newRoleName == models.RoleOrgOwner {
		return nil
	}

	ownerIDs, err := authService.GetOrgUsersForRole(ctx, models.RoleOrgOwner, orgID)
	if err != nil {
		return grpcerrors.Internal(err, "error determining organization owners")
	}

	if len(ownerIDs) <= 1 && slices.Contains(ownerIDs, targetUserID) {
		return grpcerrors.FailedPrecondition(nil, "cannot demote the last organization owner")
	}

	return nil
}
