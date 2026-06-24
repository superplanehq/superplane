package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpcerrors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func ValidateCanvasVersionChangeset(
	ctx context.Context,
	registry *registry.Registry,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	versionID uuid.UUID,
	changeset *pb.CanvasChangeset,
) (*pb.ValidateCanvasVersionChangesetResponse, error) {
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

	version, err := models.FindCanvasVersion(canvasID, versionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "version not found")
		}

		log.WithError(err).Errorf("failed to find canvas version - canvas=%s, version=%s", canvasID.String(), versionID.String())
		return nil, grpcerrors.Internal(err, "failed to find canvas version")
	}

	if version.OwnerID == nil || *version.OwnerID != user {
		return nil, grpcerrors.PermissionDenied(nil, "version owner mismatch")
	}

	if version.State == models.CanvasVersionStatePublished || version.State == models.CanvasVersionStateSnapshot {
		return nil, grpcerrors.FailedPrecondition(nil, "published versions are immutable")
	}

	patcher := changesets.NewCanvasPatcher(database.Conn(), organizationID, registry, version)
	err = patcher.ApplyChangeset(changeset, nil)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "changeset is invalid")
	}

	return &pb.ValidateCanvasVersionChangesetResponse{
		Version: SerializeCanvasVersion(patcher.GetVersion(), organizationID.String(), nil),
	}, nil
}
