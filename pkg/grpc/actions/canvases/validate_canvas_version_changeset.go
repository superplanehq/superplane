package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	user, err := uuid.Parse(userID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %v", err)
	}

	if changeset == nil || len(changeset.Changes) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "changeset is required")
	}

	version, err := models.FindCanvasVersion(canvasID, versionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "version not found")
		}

		log.WithError(err).Errorf("failed to find canvas version - canvas=%s, version=%s", canvasID.String(), versionID.String())
		return nil, status.Error(codes.Internal, "failed to find canvas version")
	}

	if version.OwnerID == nil || *version.OwnerID != user {
		return nil, status.Error(codes.PermissionDenied, "version owner mismatch")
	}

	if version.State == models.CanvasVersionStatePublished || version.State == models.CanvasVersionStateSnapshot {
		return nil, status.Error(codes.FailedPrecondition, "published versions are immutable")
	}

	patcher := changesets.NewCanvasPatcher(database.Conn(), organizationID, registry, version)
	err = patcher.ApplyChangeset(changeset, nil)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "changeset is invalid: %v", err)
	}

	return &pb.ValidateCanvasVersionChangesetResponse{
		Version: SerializeCanvasVersion(patcher.GetVersion(), organizationID.String()),
	}, nil
}
