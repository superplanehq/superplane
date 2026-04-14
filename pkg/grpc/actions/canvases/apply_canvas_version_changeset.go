package canvases

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ApplyCanvasVersionChangeset(
	ctx context.Context,
	registry *registry.Registry,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	versionID uuid.UUID,
	changeset *pb.CanvasChangeset,
	dryRun bool,
) (*pb.ApplyCanvasVersionChangesetResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	user, err := uuid.Parse(userID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %v", err)
	}

	var newVersion *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		version, err := models.FindCanvasVersionInTransaction(tx, canvasID, versionID)
		if err != nil {
			return status.Errorf(codes.NotFound, "version not found: %v", err)
		}

		if version.OwnerID == nil || *version.OwnerID != user {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		if version.State == models.CanvasVersionStatePublished {
			return status.Error(codes.FailedPrecondition, "published versions are immutable")
		}

		//
		// Apply operations to version.
		//
		patcher := changesets.NewCanvasPatcher(registry, version)
		err = patcher.ApplyChangeset(changeset)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to update canvas version: %v", err)
		}
		newVersion = patcher.GetVersion()

		//
		// If dry run is used, we do not persist the change to the database.
		//
		if dryRun {
			return nil
		}

		now := time.Now()
		newVersion.UpdatedAt = &now
		return tx.Save(newVersion).Error
	})

	if err != nil {
		return nil, err
	}

	//
	// if we didn't persist the change, we don't send the RabbitMQ message.
	//
	if dryRun {
		return &pb.ApplyCanvasVersionChangesetResponse{
			Version: SerializeCanvasVersion(newVersion, organizationID.String()),
		}, nil
	}

	err = messages.NewCanvasVersionUpdatedMessage(canvasID.String(), newVersion.ID.String()).PublishVersionUpdated()
	if err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.ApplyCanvasVersionChangesetResponse{
		Version: SerializeCanvasVersion(newVersion, organizationID.String()),
	}, nil
}
