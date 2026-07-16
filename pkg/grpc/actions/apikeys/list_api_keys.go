package apikeys

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/api_keys"
)

func ListAPIKeys(ctx context.Context) (*pb.ListAPIKeysResponse, error) {
	_, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	db := database.DB(ctx)
	users, err := models.FindAPIKeysByOrganization(db, orgID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list API keys")
	}

	creatorsByID, err := creatorsByIDForAPIKeys(db, orgID, users)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list API keys")
	}

	apiKeys := make([]*pb.APIKey, len(users))
	for i := range users {
		var creator *models.User
		if users[i].CreatedBy != nil {
			creator = creatorsByID[users[i].CreatedBy.String()]
		}
		apiKeys[i] = serializeAPIKey(&users[i], creator)
	}

	return &pb.ListAPIKeysResponse{
		ApiKeys: apiKeys,
	}, nil
}
