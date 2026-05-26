package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func resolveDashboardVersionID(tx *gorm.DB, canvas *models.Canvas, requestedVersionID string) (uuid.UUID, error) {
	if requestedVersionID != "" {
		versionUUID, err := uuid.Parse(requestedVersionID)
		if err != nil {
			return uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
		}
		return versionUUID, nil
	}

	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, status.Error(codes.NotFound, "canvas live version not found")
		}
		return uuid.Nil, err
	}

	return liveVersion.ID, nil
}

func ensureDashboardVersionReadable(
	ctx context.Context,
	tx *gorm.DB,
	canvas *models.Canvas,
	version *models.CanvasVersion,
) error {
	if version.State == models.CanvasVersionStatePublished {
		return nil
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "user not authenticated")
	}
	userUUID := uuid.MustParse(userID)

	if _, draftErr := models.FindCanvasDraftByVersionInTransaction(tx, canvas.ID, userUUID, version.ID); draftErr == nil {
		return nil
	} else if !errors.Is(draftErr, gorm.ErrRecordNotFound) {
		return draftErr
	}

	request, requestErr := models.FindCanvasChangeRequestByVersionInTransaction(tx, canvas.ID, version.ID)
	if requestErr != nil {
		if errors.Is(requestErr, gorm.ErrRecordNotFound) {
			return status.Error(codes.PermissionDenied, "version is not visible in current flow")
		}
		return requestErr
	}
	if request.OwnerID != nil && *request.OwnerID == userUUID {
		return nil
	}

	return status.Error(codes.PermissionDenied, "version is not visible in current flow")
}
