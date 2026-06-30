package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func resolveConsoleVersionID(tx *gorm.DB, canvas *models.Canvas, requestedVersionID string) (uuid.UUID, error) {
	if requestedVersionID != "" {
		versionUUID, err := uuid.Parse(requestedVersionID)
		if err != nil {
			return uuid.Nil, grpcerrors.InvalidArgument(err, "invalid version id")
		}
		return versionUUID, nil
	}

	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, grpcerrors.NotFound(err, "canvas live version not found")
		}
		return uuid.Nil, err
	}

	return liveVersion.ID, nil
}

func ensureConsoleVersionReadable(
	ctx context.Context,
	tx *gorm.DB,
	canvas *models.Canvas,
	version *models.CanvasVersion,
) error {
	_ = tx
	_ = canvas
	_ = version
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	return nil
}
