package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListGroups(ctx context.Context, domainType string, domainID string, authService authorization.Authorization) (*pb.ListGroupsResponse, error) {
	groupNames, err := authService.GetGroups(domainID, domainType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get groups")
	}

	groups := make([]*pb.Group, len(groupNames))
	for i, groupName := range groupNames {
		role, err := authService.GetGroupRole(domainID, domainType, groupName)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to get group roles")
		}

		groupUsers, err := authService.GetGroupUsers(domainID, domainType, groupName)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to get group members count")
		}

		groupMetadata, err := models.FindGroupMetadata(groupName, domainType, domainID)
		var createdAt, updatedAt *timestamppb.Timestamp
		var displayName, description string
		if err == nil {
			createdAt = timestamppb.New(groupMetadata.CreatedAt)
			updatedAt = timestamppb.New(groupMetadata.UpdatedAt)
			displayName = groupMetadata.DisplayName
			description = groupMetadata.Description
		} else {
			displayName = groupName
			description = ""
		}

		groups[i] = &pb.Group{
			Metadata: &pb.Group_Metadata{
				Name:       groupName,
				DomainType: actions.DomainTypeToProto(domainType),
				DomainId:   domainID,
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
