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

func DescribeGroup(ctx context.Context, domainType, domainID, groupName string, authService authorization.Authorization) (*pb.DescribeGroupResponse, error) {
	if groupName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "group name must be specified")
	}

	role, err := authService.GetGroupRole(ctx, domainID, domainType, groupName)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "group not found")
	}

	groupMetadata, err := models.FindGroupMetadata(database.DB(ctx), groupName, domainType, domainID)
	var displayName, description string
	var createdAt, updatedAt *timestamppb.Timestamp
	if err == nil {
		displayName = groupMetadata.DisplayName
		description = groupMetadata.Description
		createdAt = timestamppb.New(groupMetadata.CreatedAt)
		updatedAt = timestamppb.New(groupMetadata.UpdatedAt)
	} else {
		return nil, grpcerrors.NotFound(err, "group not found")
	}

	groupUsers, err := authService.GetGroupUsers(ctx, domainID, domainType, groupName)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get group members count")
	}

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       groupName,
			DomainType: actions.DomainTypeToProto(domainType),
			DomainId:   domainID,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
		},
		Spec: &pb.Group_Spec{
			Description: description,
			Role:        role,
			DisplayName: displayName,
		},
		Status: &pb.Group_Status{
			MembersCount: int32(len(groupUsers)),
		},
	}

	return &pb.DescribeGroupResponse{
		Group: group,
	}, nil
}
