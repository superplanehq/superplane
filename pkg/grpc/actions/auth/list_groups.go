package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListGroups(ctx context.Context, domainType string, domainID string, req *pb.ListGroupsRequest, authService authorization.Authorization) (*pb.ListGroupsResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	if req.DomainType == pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}


	groupNames, err := authService.GetGroups(req.DomainId, domainType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get groups")
	}

	groups := make([]*pb.Group, len(groupNames))
	for i, groupName := range groupNames {
		role, err := authService.GetGroupRole(req.DomainId, domainType, groupName)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to get group roles")
		}

		groupUsers, err := authService.GetGroupUsers(req.DomainId, domainType, groupName)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to get group members count")
		}

		groupMetadata, err := models.FindGroupMetadata(groupName, domainType, req.DomainId)
		var createdAt, updatedAt *timestamppb.Timestamp
		var displayName, description string
		if err == nil {
			createdAt = timestamppb.New(groupMetadata.CreatedAt)
			updatedAt = timestamppb.New(groupMetadata.UpdatedAt)
			displayName = groupMetadata.DisplayName
			description = groupMetadata.Description
		} else {
			// Use fallback values when metadata is not found
			displayName = groupName
			description = ""
		}

		groups[i] = &pb.Group{
			Metadata: &pb.Group_Metadata{
				Name:       groupName,
				DomainType: req.DomainType,
				DomainId:   req.DomainId,
				CreatedAt:  createdAt,
				UpdatedAt:  updatedAt,
			},
			Spec: &pb.Group_Spec{
				Role:        role,
				DisplayName: displayName,
				Description: description,
			},
			Status: &pb.Group_Status{
				MembersCount: int32(len(groupUsers)),
			},
		}
	}

	return &pb.ListGroupsResponse{
		Groups: groups,
	}, nil
}
