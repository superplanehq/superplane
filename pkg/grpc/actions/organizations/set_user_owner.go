package organizations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

func SetUserOwner(ctx context.Context, orgID, userID string, isOwner bool) (*pb.SetUserOwnerResponse, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization ID")
	}

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := models.FindActiveUserByIDInTransaction(tx, orgID, userID)
		if err != nil {
			return err
		}

		if user.IsAPIKey() {
			return grpcerrors.InvalidArgument(nil, "API keys cannot be organization owners")
		}

		if !isOwner && user.IsOwner {
			if err := models.RefuseIfLastOrganizationOwner(tx, orgUUID, user.ID); err != nil {
				return err
			}
		}

		return models.SetUserIsOwner(tx, user.ID, isOwner)
	})
	if err != nil {
		return nil, mapOwnerMutationError(orgID, userID, err)
	}

	return &pb.SetUserOwnerResponse{}, nil
}

func mapOwnerMutationError(orgID, userID string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return grpcerrors.NotFound(err, "user not found")
	}

	if errors.Is(err, models.ErrLastOrganizationOwner) {
		return grpcerrors.FailedPrecondition(err, models.ErrLastOrganizationOwner.Error())
	}

	if _, _, ok := grpcerrors.HandlerStatus(err); ok {
		return err
	}

	log.Errorf("Error updating owner flag for %s in org %s: %v", userID, orgID, err)
	return grpcerrors.Internal(err, "failed to update owner flag")
}
