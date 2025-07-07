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

func CreateGroup(ctx context.Context, req *CreateGroupRequest, authService authorization.Authorization) (*CreateGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	err = ValidateCreateGroupRequest(req)
	if err != nil {
		return nil, err
	}

	domainType, err := ConvertDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}

	// TODO: once orgs/canvases are implemented, check if the domain exists

	err = authService.CreateGroup(req.DomainId, domainType, req.GroupName, req.Role)
	if err != nil {
		log.Errorf("failed to create group %s with role %s in domain %s: %v", req.GroupName, req.Role, req.DomainId, err)
		return nil, status.Error(codes.Internal, "failed to create group")
	}

	log.Infof("created group %s with role %s in domain %s (type: %s)", req.GroupName, req.Role, req.DomainId, req.DomainType.String())

	// Create the group object for response
	group := &pb.Group{
		Name:       req.GroupName,
		DomainType: req.DomainType,
		DomainId:   req.DomainId,
		Role:       req.Role,
	}

	return &CreateGroupResponse{
		Group: group,
	}, nil
}
