package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetCanvasUsers(ctx context.Context, req *pb.GetCanvasUsersRequest, authService authorization.Authorization) (*pb.GetCanvasUsersResponse, error) {
	canvasID, err := ConvertCanvasIdOrNameToId(req.CanvasIdOrName)
	if err != nil {
		return nil, err
	}

	users, err := GetUsersWithRolesInDomain(canvasID, models.DomainCanvas, authService)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get canvas users")
	}

	return &pb.GetCanvasUsersResponse{
		Users: users,
	}, nil
}
