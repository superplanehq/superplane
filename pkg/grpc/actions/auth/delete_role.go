package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteRole(ctx context.Context, domainType, domainID, roleName string, authService authorization.Authorization) (*pb.DeleteRoleResponse, error) {
	if roleName == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
	}

	_, err := authService.GetRoleDefinition(roleName, domainType, domainID)
	if err != nil {
		log.Errorf("role %s not found: %v", roleName, err)
		return nil, status.Error(codes.NotFound, "role not found")
	}

	if domainType == models.DomainTypeOrganization {
		if err := reassignGroupsForDeletedRole(authService, domainID, domainType, roleName); err != nil {
			return nil, err
		}
		if err := removeUsersForDeletedRole(ctx, authService, domainID, domainType, roleName); err != nil {
			return nil, err
		}
	}

	err = authService.DeleteCustomRole(domainID, domainType, roleName)
	if err != nil {
		log.Errorf("failed to delete role %s: %v", roleName, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	log.Infof("deleted custom role %s from domain %s (%s)", roleName, domainID, domainType)

	return &pb.DeleteRoleResponse{}, nil
}

func reassignGroupsForDeletedRole(authService authorization.Authorization, domainID, domainType, roleName string) error {
	groups, err := authService.GetGroups(domainID, domainType)
	if err != nil {
		log.Errorf("failed to get groups for org %s: %v", domainID, err)
		return status.Error(codes.Internal, "failed to get groups for org")
	}

	for _, groupName := range groups {
		groupRole, err := authService.GetGroupRole(domainID, domainType, groupName)
		if err != nil {
			log.Errorf("failed to get role for group %s: %v", groupName, err)
			return status.Error(codes.Internal, "failed to get group role")
		}

		if groupRole != roleName {
			continue
		}

		err = authService.UpdateGroup(domainID, domainType, groupName, models.RoleOrgViewer, "", "")
		if err != nil {
			log.Errorf("failed to reassign group %s to member: %v", groupName, err)
			return status.Error(codes.Internal, "failed to reassign group")
		}
	}

	return nil
}

func removeUsersForDeletedRole(ctx context.Context, authService authorization.Authorization, domainID, domainType, roleName string) error {
	userIDs, err := authService.GetOrgUsersForRole(roleName, domainID)
	if err != nil {
		log.Errorf("failed to get users for role %s: %v", roleName, err)
		return status.Error(codes.Internal, "failed to get users for role")
	}

	for _, userID := range userIDs {
		userRoles, err := authService.GetUserRolesForOrg(userID, domainID)
		if err != nil {
			log.Errorf("failed to get roles for user %s: %v", userID, err)
			return status.Error(codes.Internal, "failed to get user roles")
		}

		if len(userRoles) == 1 && userRoles[0].Name == roleName {
			_, err := organizations.RemoveUser(ctx, authService, domainID, userID)
			if err == nil {
				continue
			}
			if status.Code(err) == codes.NotFound {
				err = authService.RemoveRole(userID, roleName, domainID, domainType)
			}
			if err != nil {
				log.Errorf("failed to remove user %s for role %s: %v", userID, roleName, err)
				return status.Error(codes.Internal, "failed to remove user for role")
			}
			continue
		}

		if err := authService.RemoveRole(userID, roleName, domainID, domainType); err != nil {
			log.Errorf("failed to remove role %s for user %s: %v", roleName, userID, err)
			return status.Error(codes.Internal, "failed to remove role for user")
		}
	}

	return nil
}
