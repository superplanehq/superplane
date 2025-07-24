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

func CreateGroup(ctx context.Context, domainType string, domainID string, req *pb.CreateGroupRequest, authService authorization.Authorization) (*pb.CreateGroupResponse, error) {
	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "role must be specified")
	}

	err := authService.CreateGroup(domainID, domainType, req.GroupName, req.Role)
	if err != nil {
		log.Errorf("failed to create group %s with role %s in domain %s: %v", req.GroupName, req.Role, domainID, err)
		return nil, status.Error(codes.Internal, "failed to create group")
	}

	var displayName string
	var description string
	if req.DisplayName != "" || req.Description != "" {
		displayName = req.DisplayName
		if displayName == "" {
			displayName = req.GroupName
		}
		description = req.Description

		err = models.UpsertGroupMetadata(req.GroupName, domainType, domainID, displayName, description)
		if err != nil {
			log.Errorf("failed to create group metadata for %s: %v", req.GroupName, err)
			return nil, status.Error(codes.Internal, "failed to create group metadata")
		}
	}

	log.Infof("created group %s with role %s in domain %s (type: %s)", req.GroupName, req.Role, domainID, domainType)

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       req.GroupName,
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
		Group: group,
	}, nil
}
