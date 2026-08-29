package me

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
)

func ListTokens(ctx context.Context) (*pb.ListTokensResponse, error) {
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

	if user.IsAPIKey() {
		return nil, grpcerrors.PermissionDenied(nil, "API keys must use the API key token endpoint")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid user ID")
	}

	tokens, err := models.ListUserAPITokens(database.DB(ctx), userUUID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list personal tokens")
	}

	out := make([]*pb.UserAPIToken, len(tokens))
	for i := range tokens {
		out[i] = serializeUserAPIToken(&tokens[i])
	}

	return &pb.ListTokensResponse{
		Tokens: out,
	}, nil
}
