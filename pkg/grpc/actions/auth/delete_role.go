package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest, authService authorization.Authorization) (*pb.DeleteRoleResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.RoleName == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
	}

	domainType := convertDomainType(req.DomainType)
	if domainType == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid domain type")
	}

	// Check if role exists
	_, err = authService.GetRoleDefinition(req.RoleName, domainType, req.DomainId)
	if err != nil {
		log.Errorf("role %s not found: %v", req.RoleName, err)
		return nil, status.Error(codes.NotFound, "role not found")
	}

	err = authService.DeleteCustomRole(req.DomainId, domainType, req.RoleName)
	if err != nil {
		log.Errorf("failed to delete role %s: %v", req.RoleName, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Infof("deleted custom role %s from domain %s (%s)", req.RoleName, req.DomainId, domainType)

	return &pb.DeleteRoleResponse{}, nil
}