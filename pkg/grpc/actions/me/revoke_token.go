package me

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"gorm.io/gorm"
)

func RevokeToken(ctx context.Context, req *pb.RevokeTokenRequest) (*pb.RevokeTokenResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	if req.GetId() == "" {
		return nil, grpcerrors.InvalidArgument(nil, "id is required")
	}

	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load user")
	}

	if user.IsAPIKey() {
		return nil, grpcerrors.PermissionDenied(nil, "API keys must use the API key token endpoint")
	}

	tokenID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid token ID")
	}

	db := database.DB(ctx)
	token, err := models.FindUserAPIToken(db, user.ID, tokenID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "personal token not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load personal token")
	}

	if err := token.HardDelete(db); err != nil {
		return nil, grpcerrors.Internal(err, "failed to revoke personal token")
	}

	return &pb.RevokeTokenResponse{}, nil
}
