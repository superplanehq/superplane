package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListGroups(ctx context.Context, domainType string, domainID string, authService authorization.Authorization) (*pb.ListGroupsResponse, error) {
	groupDetails, err := authService.GetGroupsWithDetails(ctx, domainID, domainType)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get groups")
	}

	groupNames := make([]string, len(groupDetails))
	for i, detail := range groupDetails {
		groupNames[i] = detail.Name
	}

	metadataByGroup, err := models.FindGroupMetadataByNames(database.DB(ctx), groupNames, domainType, domainID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get group metadata")
	}

	groups := make([]*pb.Group, len(groupDetails))
	for i, detail := range groupDetails {
		var createdAt, updatedAt *timestamppb.Timestamp
		displayName, description := detail.Name, ""
		if metadata := metadataByGroup[detail.Name]; metadata != nil {
			createdAt = timestamppb.New(metadata.CreatedAt)
			updatedAt = timestamppb.New(metadata.UpdatedAt)
			displayName = metadata.DisplayName
			description = metadata.Description
		}

		groups[i] = &pb.Group{
			Metadata: &pb.Group_Metadata{
				Name:       detail.Name,
				DomainType: actions.DomainTypeToProto(domainType),
				DomainId:   domainID,
				CreatedAt:  createdAt,
				UpdatedAt:  updatedAt,
			},
			Spec: &pb.Group_Spec{
				Role:        detail.Role,
				DisplayName: displayName,
				Description: description,
			},
			Status: &pb.Group_Status{
				MembersCount: int32(len(detail.Members)),
			},
		}
	}

	return &pb.ListGroupsResponse{
		Groups: groups,
	}, nil
}
