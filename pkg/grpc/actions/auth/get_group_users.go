package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetGroupUsers(ctx context.Context, req *GetGroupUsersRequest, authService authorization.Authorization) (*GetGroupUsersResponse, error) {
	err := actions.ValidateUUIDs(req.DomainID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	groupReq := &GroupRequest{
		DomainID:   req.DomainID,
		GroupName:  req.GroupName,
		DomainType: req.DomainType,
	}

	err = ValidateGroupRequest(groupReq)
	if err != nil {
		return nil, err
	}

	domainType, err := ConvertDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}

	userIDs, err := authService.GetGroupUsers(req.DomainID, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	role, err := authService.GetGroupRole(req.DomainID, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group roles")
	}

	group := &pb.Group{
		Name:        req.GroupName,
		DomainType:  req.DomainType,
		DomainId:    req.DomainID,
		Role:        role,
		DisplayName: models.GetGroupDisplayName(req.GroupName, domainType, req.DomainID),
		Description: models.GetGroupDescription(req.GroupName, domainType, req.DomainID),
	}

	return &GetGroupUsersResponse{
		UserIDs: userIDs,
		Group:   group,
	}, nil
}
