package canvases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
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
	authService authorization.Authorization,
	webhookBaseURL string,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	versionID uuid.UUID,
	changeset *pb.CanvasChangeset,
	autoLayout *pb.CanvasAutoLayout,
) (*pb.ApplyCanvasVersionChangesetResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	activeUser, err := models.FindActiveUserByID(organizationID.String(), userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get active user: %v", err)
	}

	if changeset == nil || len(changeset.Changes) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "changeset is required")
	}

	var newVersion *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		version, err := models.FindCanvasVersionForUpdateInTransaction(tx, canvasID, versionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}

			log.WithError(err).Errorf("failed to find canvas version - canvas=%s, version=%s", canvasID.String(), versionID.String())
			return status.Error(codes.Internal, "failed to find canvas version")
		}

		if version.OwnerID == nil || version.OwnerID.String() != userID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		if version.State == models.CanvasVersionStatePublished || version.State == models.CanvasVersionStateSnapshot {
			return status.Error(codes.FailedPrecondition, "published versions are immutable")
		}

		//
		// Apply operations to version.
		//
		patcher, err := changesets.NewCanvasPatcher(tx, version, &changesets.CanvasPatcherOptions{
			OrgID:             organizationID,
			Registry:          registry,
			BaseURL:           webhookBaseURL,
			AuthService:       authService,
			AuthenticatedUser: activeUser,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to create canvas patcher: %v", err)
		}
		err = patcher.ApplyChangeset(changeset, autoLayout)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to update canvas version: %v", err)
		}

		now := time.Now()
		newVersion = patcher.GetVersion()
		newVersion.UpdatedAt = &now
		err = tx.Save(newVersion).Error
		if err != nil {
			log.WithError(err).Errorf("failed to save canvas version - canvas=%s, version=%s", canvasID.String(), newVersion.ID.String())
			return status.Error(codes.Internal, "failed to save canvas version")
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
		Version: SerializeCanvasVersion(newVersion, organizationID.String()),
	}, nil
}
