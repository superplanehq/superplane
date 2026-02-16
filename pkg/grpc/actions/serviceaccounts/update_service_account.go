package serviceaccounts

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateServiceAccount(ctx context.Context, req *pb.UpdateServiceAccountRequest) (*pb.UpdateServiceAccountResponse, error) {
	_, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	user, err := models.FindActiveUserByID(orgID, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "service account not found")
	}

	if !user.IsServiceAccount() {
		return nil, status.Error(codes.NotFound, "service account not found")
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	if req.Description != "" {
		user.Description = &req.Description
	}

	user.UpdatedAt = time.Now()
	err = database.Conn().Save(user).Error
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update service account")
	}

	return &pb.UpdateServiceAccountResponse{
		ServiceAccount: serializeServiceAccount(user),
	}, nil
}
