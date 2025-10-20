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

func AssignRole(ctx context.Context, orgID, domainType, domainID, roleName, userID, userEmail string, authService authorization.Authorization) (*pb.AssignRoleResponse, error) {
	if roleName == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user not found")
	}

	if domainType == models.DomainTypeCanvas && domainID == "*" {
		err = authService.AssignRoleWithOrgContext(user.ID.String(), roleName, domainID, domainType, orgID)
	} else {
		err = authService.AssignRole(user.ID.String(), roleName, domainID, domainType)
	}
	if err != nil {
		log.Errorf("Error assigning role %s to %s: %v", roleName, user.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to assign role")
	}

	return &pb.AssignRoleResponse{}, nil
}
