package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListGroupUsers(ctx context.Context, domainType, domainID, groupName string, authService authorization.Authorization) (*pb.ListGroupUsersResponse, error) {
	if groupName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "group name must be specified")
	}

	db := database.DB(ctx)

	role, err := authService.GetGroupRole(ctx, domainID, domainType, groupName)
	if err != nil {
		log.Errorf("failed to get group role: %v", err)
		return nil, grpcerrors.NotFound(err, "group not found")
	}

	userIDs, err := authService.GetGroupUsers(ctx, domainID, domainType, groupName)
	if err != nil {
		log.Errorf("failed to get group users: %v", err)
		return nil, grpcerrors.Internal(err, "failed to get group users")
	}

	users, err := models.FindUsersByIDs(db, userIDs)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to fetch group users")
	}

	accountProviders, err := models.FindUserAccountProviders(db, users)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to fetch account providers")
	}

	protoUsers := usersToProto(users, accountProviders)
	groupMetadata, err := models.FindGroupMetadata(db, groupName, domainType, domainID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "group not found")
	}

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       groupName,
			DomainType: actions.DomainTypeToProto(domainType),
			DomainId:   domainID,
			CreatedAt:  timestamppb.New(groupMetadata.CreatedAt),
			UpdatedAt:  timestamppb.New(groupMetadata.UpdatedAt),
		},
		Spec: &pb.Group_Spec{
			Description: groupMetadata.Description,
			DisplayName: groupMetadata.DisplayName,
			Role:        role,
		},
		Status: &pb.Group_Status{
			MembersCount: int32(len(userIDs)),
		},
	}

	return &pb.ListGroupUsersResponse{
		Users: protoUsers,
		Group: group,
	}, nil
}
