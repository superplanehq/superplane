package canvases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpcerrors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func ApplyCanvasVersionChangeset(
	ctx context.Context,
	registry *registry.Registry,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	versionID uuid.UUID,
	changeset *pb.CanvasChangeset,
	autoLayout *pb.CanvasAutoLayout,
) (*pb.ApplyCanvasVersionChangesetResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	user, err := uuid.Parse(userID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid user id")
	}

	if changeset == nil || len(changeset.Changes) == 0 {
		return nil, grpcerrors.InvalidArgument(nil, "changeset is required")
	}

	var newVersion *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		version, err := models.FindCanvasVersionForUpdateInTransaction(tx, canvasID, versionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return grpcerrors.NotFound(err, "version not found")
			}

			log.WithError(err).Errorf("failed to find canvas version - canvas=%s, version=%s", canvasID.String(), versionID.String())
			return grpcerrors.Internal(err, "failed to find canvas version")
		}

		if version.OwnerID == nil || *version.OwnerID != user {
			return grpcerrors.PermissionDenied(nil, "version owner mismatch")
		}

		if version.State == models.CanvasVersionStatePublished || version.State == models.CanvasVersionStateSnapshot {
			return grpcerrors.FailedPrecondition(nil, "published versions are immutable")
		}

		//
		// Apply operations to version.
		//
		patcher := changesets.NewCanvasPatcher(tx, organizationID, registry, version)
		err = patcher.ApplyChangeset(changeset, autoLayout)
		if err != nil {
			return grpcerrors.InvalidArgument(err, "failed to update canvas version")
		}

		now := time.Now()
		newVersion = patcher.GetVersion()
		newVersion.UpdatedAt = &now
		err = tx.Save(newVersion).Error
		if err != nil {
			log.WithError(err).Errorf("failed to save canvas version - canvas=%s, version=%s", canvasID.String(), newVersion.ID.String())
			return grpcerrors.Internal(err, "failed to save canvas version")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	err = messages.NewCanvasVersionUpdatedMessage(canvasID.String(), newVersion.ID.String()).PublishVersionUpdated()
	if err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.ApplyCanvasVersionChangesetResponse{
		Version: SerializeCanvasVersion(newVersion, organizationID.String(), nil),
	}, nil
}
