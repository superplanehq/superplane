package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListGroups(ctx context.Context, req *pb.ListGroupsRequest, authService authorization.Authorization) (*pb.ListGroupsResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var domainType string
	switch req.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		domainType = "org"
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		domainType = "canvas"
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	groupNames, err := authService.GetGroups(req.DomainId, domainType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get groups")
	}

	// Convert group names to Group objects
	groups := make([]*pb.Group, len(groupNames))
	for i, groupName := range groupNames {
		// Get the role for this group - for now we'll leave it empty as the interface doesn't provide it
		// TODO: Update authorization service to return group details including roles
		groups[i] = &pb.Group{
			Name:       groupName,
			DomainType: req.DomainType,
			DomainId:   req.DomainId,
			Role:       "", // TODO: get actual role from service
		}
	}

	return &pb.ListGroupsResponse{
		Groups: groups,
	}, nil
}