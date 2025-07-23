package auth

import (
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GroupRequest struct {
	DomainID   string
	GroupName  string
	DomainType pbAuth.DomainType
}

type GroupUserRequest struct {
	DomainID   string
	GroupName  string
	DomainType pbAuth.DomainType
	UserID     string
	UserEmail  string
}

func ValidateGroupRequest(req *GroupRequest) error {
	if req.GroupName == "" {
		return status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	return nil
}

func ValidateGroupUserRequest(req *GroupUserRequest) error {
	if req.GroupName == "" {
		return status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	return nil
}

func ValidateCreateGroupRequest(req *pbGroups.CreateGroupRequest) error {
	if req.GroupName == "" {
		return status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.Role == "" {
		return status.Error(codes.InvalidArgument, "role must be specified")
	}

	if req.DomainType == pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	return nil
}

func ValidateUpdateGroupRequest(req *pb.UpdateGroupRequest) error {
	if req.GroupName == "" {
		return status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	// At least one field must be provided for update
	if req.Role == "" && req.DisplayName == "" && req.Description == "" {
		return status.Error(codes.InvalidArgument, "at least one field must be provided for update")
	}

	return nil
}
