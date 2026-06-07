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

func resolveConsoleVersionID(tx *gorm.DB, canvas *models.Canvas, requestedVersionID string) (uuid.UUID, error) {
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

func ensureConsoleVersionReadable(
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

	// Drafts are user-private work-in-progress: only the draft owner can
	// read them. Reviewers don't need to see in-flight drafts because the
	// review surface is the change request snapshot, not the draft.
	if models.IsUserOwnedDraftVersion(version, userUUID) && models.IsRegisteredDraftVersion(version) {
		return nil
	}

	// Snapshot versions are exposed through a change request. Change
	// requests themselves are described to any authenticated org member
	// (see DescribeCanvasChangeRequest, which returns the snapshot's full
	// spec), so the matching console must be readable too — otherwise
	// reviewers can see the proposed graph but get a 403 when the UI
	// fetches its console. Drafts are still gated by the check above.
	if _, requestErr := models.FindCanvasChangeRequestByVersionInTransaction(tx, canvas.ID, version.ID); requestErr == nil {
		return nil
	} else if !errors.Is(requestErr, gorm.ErrRecordNotFound) {
		return requestErr
	}

	return status.Error(codes.PermissionDenied, "version is not visible in current flow")
}
