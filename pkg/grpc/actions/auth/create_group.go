package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateGroup(ctx context.Context, domainType, domainID string, group *pb.Group, authService authorization.Authorization) (*pb.CreateGroupResponse, error) {
	if group == nil {
		return nil, status.Error(codes.InvalidArgument, "group must be specified")
	}

	if group.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "group metadata must be specified")
	}

	if group.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "group spec must be specified")
	}

	if group.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if group.Spec.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "role must be specified")
	}

	var displayName string
	var description string
	if group.Spec.DisplayName != "" || group.Spec.Description != "" {
		displayName = group.Spec.DisplayName
		if displayName == "" {
			displayName = group.Metadata.Name
		}
		description = group.Spec.Description
	}

	err := authService.CreateGroup(domainID, domainType, group.Metadata.Name, group.Spec.Role, displayName, description)
	if err != nil {
		log.Errorf("failed to create group %s with role %s in domain %s: %v", group.Metadata.Name, group.Spec.Role, domainID, err)
		return nil, status.Error(codes.Internal, "failed to create group")
	}

	log.Infof("created group %s with role %s in domain %s (type: %s)", group.Metadata.Name, group.Spec.Role, domainID, domainType)

	groupResponse := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       group.Metadata.Name,
			DomainType: actions.DomainTypeToProto(domainType),
			DomainId:   domainID,
			CreatedAt:  timestamppb.Now(),
			UpdatedAt:  timestamppb.Now(),
		},
		Spec: &pb.Group_Spec{
			DisplayName: displayName,
			Description: description,
		},
	}

	return &pb.CreateGroupResponse{
		Group: groupResponse,
	}, nil
}
