package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
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

	err = authService.DeleteCustomRole(domainID, domainType, roleName)
	if err != nil {
		log.Errorf("failed to delete role %s: %v", roleName, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	log.Infof("deleted custom role %s from domain %s (%s)", roleName, domainID, domainType)

	return &pb.DeleteRoleResponse{}, nil
}
