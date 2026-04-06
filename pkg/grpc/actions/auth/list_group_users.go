package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListGroupUsers(ctx context.Context, domainType, domainID, groupName string, authService authorization.Authorization) (*pb.ListGroupUsersResponse, error) {
	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	role, err := authService.GetGroupRole(domainID, domainType, groupName)
	if err != nil {
		log.Errorf("failed to get group role: %v", err)
		return nil, status.Error(codes.NotFound, "group not found")
	}

	userIDs, err := authService.GetGroupUsers(domainID, domainType, groupName)
	if err != nil {
		log.Errorf("failed to get group users: %v", err)
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	users, err := models.FindUsersByIDs(userIDs)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch group users")
	}

	accountProviders, err := getAccountProviders(users)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch account providers")
	}

	protoUsers := usersToProto(users, accountProviders)
	groupMetadata, err := models.FindGroupMetadata(groupName, domainType, domainID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "group not found")
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
			Description: groupMetadata.DisplayName,
			DisplayName: groupMetadata.Description,
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
