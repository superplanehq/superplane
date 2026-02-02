package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
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
		userIDs, err := authService.GetOrgUsersForRole(roleName, domainID)
		if err != nil {
			log.Errorf("failed to get users for role %s: %v", roleName, err)
			return nil, status.Error(codes.Internal, "failed to get users for role")
		}

		for _, userID := range userIDs {
			if err := authService.RemoveRole(userID, roleName, domainID, domainType); err != nil {
				log.Errorf("failed to remove role %s for user %s: %v", roleName, userID, err)
				return nil, status.Error(codes.Internal, "failed to remove role for user")
			}
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
