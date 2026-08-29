package me

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"gorm.io/gorm"
)

func CreateToken(ctx context.Context, req *pb.CreateTokenRequest) (*pb.CreateTokenResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, grpcerrors.InvalidArgument(nil, "name is required")
	}

	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load user")
	}

	if user.IsAPIKey() {
		return nil, grpcerrors.PermissionDenied(nil, "API keys must use the API key token endpoint")
	}

	plainToken, err := crypto.Base64String(64)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to generate token")
	}

	token := models.NewUserAPIToken(user.ID, name, crypto.HashToken(plainToken))

	db := database.DB(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		return models.CreateUserAPIToken(tx, token)
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create personal token")
	}

	return &pb.CreateTokenResponse{
		Token:     serializeUserAPIToken(token),
		Plaintext: plainToken,
	}, nil
}
