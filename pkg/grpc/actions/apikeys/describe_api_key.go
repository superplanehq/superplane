package apikeys

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/api_keys"
)

func DescribeAPIKey(ctx context.Context, req *pb.DescribeAPIKeyRequest) (*pb.DescribeAPIKeyResponse, error) {
	_, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	if req.Id == "" {
		return nil, grpcerrors.InvalidArgument(nil, "id is required")
	}

	user, err := models.FindActiveUserByID(orgID, req.Id)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "API key not found")
	}

	if !user.IsAPIKey() {
		return nil, grpcerrors.NotFound(err, "API key not found")
	}

	db := database.DB(ctx)
	creator, err := creatorUserForAPIKey(db, orgID, user)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe API key")
	}

	return &pb.DescribeAPIKeyResponse{
		ApiKey: serializeAPIKey(user, creator),
	}, nil
}
