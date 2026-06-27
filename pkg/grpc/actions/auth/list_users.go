package auth

import (
	"context"
	"slices"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListUsers(
	ctx context.Context,
	domainType string,
	domainID string,
	includeRoles bool,
	authService authorization.Authorization,
) (*pb.ListUsersResponse, error) {
	if domainType != models.DomainTypeOrganization {
		return nil, grpcerrors.InvalidArgument(nil, "domain type must be organization")
	}

	users, err := models.ListAllActiveUsersInOrganization(domainID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to fetch users")
	}

	accountProviders, err := models.FindUserAccountProviders(database.DB(ctx), users)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to fetch account providers")
	}

	protoUsers := usersToProto(users, accountProviders)
	if !includeRoles {
		return &pb.ListUsersResponse{
			Users: protoUsers,
		}, nil
	}

	//
	// Get all role definitions
	//
	roleDefinitions, err := authService.GetAllRoleDefinitions(ctx, domainType, domainID)
	if err != nil {
		return nil, err
	}

	roleNames := make([]string, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		roleNames[i] = roleDef.Name
	}

	roleMetadataMap, err := models.FindRoleMetadataByNames(roleNames, domainType, domainID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "role not found")
	}

	//
	// For each role, get all users for it
	//
	for _, roleDef := range roleDefinitions {
		userIDs, err := authService.GetOrgUsersForRole(ctx, roleDef.Name, domainID)
		if err != nil {
			continue
		}

		roleMetadata := roleMetadataMap[roleDef.Name]
		roleAssignment := &pb.RoleAssignment{
			RoleName:        roleDef.Name,
			RoleDisplayName: roleMetadata.DisplayName,
			RoleDescription: roleMetadata.Description,
			DomainType:      actions.DomainTypeToProto(domainType),
			DomainId:        domainID,
			AssignedAt:      timestamppb.Now(),
		}

		for _, userID := range userIDs {
			i := slices.IndexFunc(protoUsers, func(user *pb.User) bool {
				return user.Metadata.Id == userID
			})

			if i == -1 {
				continue
			}

			protoUsers[i].Status.Roles = append(protoUsers[i].Status.Roles, roleAssignment)
		}
	}

	return &pb.ListUsersResponse{
		Users: protoUsers,
	}, nil
}
