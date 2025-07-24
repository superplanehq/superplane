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

func DeleteRole(ctx context.Context, domainType string, domainID string, req *pb.DeleteRoleRequest, authService authorization.Authorization) (*pb.DeleteRoleResponse, error) {
	if req.RoleName == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
	}


	// Check if role exists
	_, err := authService.GetRoleDefinition(req.RoleName, domainType, domainID)
	if err != nil {
		log.Errorf("role %s not found: %v", req.RoleName, err)
		return nil, status.Error(codes.NotFound, "role not found")
	}

	err = authService.DeleteCustomRole(domainID, domainType, req.RoleName)
	if err != nil {
		log.Errorf("failed to delete role %s: %v", req.RoleName, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = models.DeleteRoleMetadata(req.RoleName, domainType, domainID)
	if err != nil {
		log.Errorf("failed to delete role metadata for %s: %v", req.RoleName, err)
		return nil, status.Error(codes.Internal, "failed to delete role metadata")
	}

	log.Infof("deleted custom role %s from domain %s (%s)", req.RoleName, domainID, domainType)

	return &pb.DeleteRoleResponse{}, nil
}
