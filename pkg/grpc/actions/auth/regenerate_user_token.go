package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RegenerateUserToken(ctx context.Context, orgID string, userID string) (*pb.RegenerateTokenResponse, error) {
	authenticatedUser, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	//
	// TODO: we are enforcing regeneration only for the user being authenticated,
	// but teoretically, the organization owner/admin should be able to regenerate any user's token.
	//
	if authenticatedUser != userID {
		return nil, status.Error(codes.PermissionDenied, "user not authorized")
	}

	user, err := models.FindUserByID(orgID, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
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
