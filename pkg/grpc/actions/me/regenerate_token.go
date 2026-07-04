package me

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
)

func RegenerateToken(ctx context.Context) (*pb.RegenerateTokenResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load user")
	}

	if user.IsServiceAccount() {
		return nil, grpcerrors.PermissionDenied(nil, "service accounts must use the service account token endpoint")
	}

	plainToken, err := crypto.Base64String(64)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to generate new token")
	}

	err = user.UpdateTokenHash(crypto.HashToken(plainToken))
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update token")
	}

	return &pb.RegenerateTokenResponse{
		Token: plainToken,
	}, nil
}
