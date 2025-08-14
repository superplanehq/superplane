package organizations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func RemoveUser(ctx context.Context, authService authorization.Authorization, orgID, userID string) (*pb.RemoveUserResponse, error) {
	user, err := models.FindUserByID(orgID, userID)
	if err != nil {
		return nil, err
	}

	roles, err := authService.GetUserRolesForOrg(orgID, user.ID.String())
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		err = authService.RemoveRole(user.ID.String(), role.Name, orgID, models.DomainTypeOrganization)
		if err != nil {
			return nil, err
		}
	}

	err = user.Delete()
	if err != nil {
		return nil, err
	}

	return &pb.RemoveUserResponse{}, nil
}
