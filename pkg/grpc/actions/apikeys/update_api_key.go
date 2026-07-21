package apikeys

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/api_keys"
	"gorm.io/datatypes"
)

func UpdateAPIKey(ctx context.Context, req *pb.UpdateAPIKeyRequest) (*pb.UpdateAPIKeyResponse, error) {
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

	if req.Name != "" {
		name := strings.TrimSpace(req.Name)
		if name == "" {
			return nil, grpcerrors.InvalidArgument(nil, "name is required")
		}
		user.Name = name
	}

	if req.Description != "" {
		user.Description = &req.Description
	}

	db := database.DB(ctx)
	if req.CanvasIds != nil {
		canvasIDs, err := validateAPIKeyCanvasIDs(db, orgID, req.CanvasIds)
		if err != nil {
			return nil, err
		}
		user.APIKeyCanvasIDs = datatypes.NewJSONSlice(canvasIDs)
	}

	if req.ClearExpiresAt {
		user.APIKeyExpiresAt = nil
	} else if req.ExpiresAt != nil {
		expiresAt := req.ExpiresAt.AsTime()
		if !expiresAt.After(time.Now()) {
			return nil, grpcerrors.InvalidArgument(nil, "expiration must be in the future")
		}
		user.APIKeyExpiresAt = &expiresAt
	}

	user.UpdatedAt = time.Now()
	err = models.UpdateAPIKey(db, user)
	if err != nil {
		if errors.Is(err, models.ErrAPIKeyNameAlreadyExists) {
			return nil, grpcerrors.AlreadyExists(err, fmt.Sprintf("an API key with the name %q already exists in this organization", user.Name))
		}
		return nil, grpcerrors.Internal(err, "failed to update API key")
	}

	creator, err := creatorUserForAPIKey(db, orgID, user)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update API key")
	}

	return &pb.UpdateAPIKeyResponse{
		ApiKey: serializeAPIKey(user, creator),
	}, nil
}
