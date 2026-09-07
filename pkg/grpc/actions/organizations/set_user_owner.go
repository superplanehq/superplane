package organizations

import (
	"context"
	"slices"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func SetUserOwner(ctx context.Context, orgID, userID string, isOwner bool) (*pb.SetUserOwnerResponse, error) {
	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "user not found")
	}

	if user.IsAPIKey() {
		return nil, grpcerrors.InvalidArgument(nil, "API keys cannot be organization owners")
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization ID")
	}

	if !isOwner && user.IsOwner {
		ownerIDs, err := models.ListOrganizationOwnerIDs(database.DB(ctx), orgUUID)
		if err != nil {
			log.Errorf("Error determining owners for org %s: %v", orgID, err)
			return nil, grpcerrors.Internal(err, "error determining organization owners")
		}

		if len(ownerIDs) <= 1 && slices.Contains(ownerIDs, user.ID.String()) {
			return nil, grpcerrors.FailedPrecondition(nil, "cannot remove the last organization owner")
		}
	}

	if err := models.SetUserIsOwner(database.DB(ctx), user.ID, isOwner); err != nil {
		log.Errorf("Error updating owner flag for %s in org %s: %v", userID, orgID, err)
		return nil, grpcerrors.Internal(err, "failed to update owner flag")
	}

	return &pb.SetUserOwnerResponse{}, nil
}
