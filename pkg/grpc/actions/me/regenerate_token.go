package me

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RegenerateToken(ctx context.Context) (*pb.RegenerateTokenResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if user.IsServiceAccount() {
		return nil, status.Error(codes.PermissionDenied, "service accounts must use the service account token endpoint")
	}

	plainToken, err := crypto.Base64String(64)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate new token")
	}

	err = user.UpdateTokenHash(crypto.HashToken(plainToken))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update token")
	}

	return &pb.RegenerateTokenResponse{
		Token: plainToken,
	}, nil
}
