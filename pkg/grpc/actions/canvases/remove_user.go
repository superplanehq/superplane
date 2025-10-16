package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveUser(ctx context.Context, authService authorization.Authorization, orgID string, canvasID string, userID string) (*pb.RemoveUserResponse, error) {
	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	roles, err := authService.GetUserRolesForCanvas(userID, canvasID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to determine user roles")
	}

	//
	// TODO: this should be in transaction
	//
	for _, role := range roles {
		err = authService.RemoveRole(user.ID.String(), role.Name, canvasID, models.DomainTypeCanvas)
		if err != nil {
			return nil, status.Error(codes.Internal, "error removing user")
		}
	}

	return &pb.RemoveUserResponse{}, nil
}
