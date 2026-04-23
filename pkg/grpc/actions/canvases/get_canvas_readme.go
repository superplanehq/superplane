package canvases

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// GetCanvasReadme returns the markdown readme for a canvas version.
//
// Version selection:
//   - empty version_id: returns the live version's readme
//   - "draft": returns the caller's draft readme
//   - any uuid: returns that specific version's readme
func GetCanvasReadme(ctx context.Context, organizationID, canvasID, versionID string) (*pb.GetCanvasReadmeResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	version, err := resolveReadmeVersion(ctx, canvas, versionID)
	if err != nil {
		return nil, err
	}

	resp := &pb.GetCanvasReadmeResponse{
		CanvasId:     canvasUUID.String(),
		VersionId:    version.ID.String(),
		VersionState: version.State,
		Content:      version.Readme,
	}

	if version.UpdatedAt != nil {
		resp.UpdatedAt = timestamppb.New(*version.UpdatedAt)
	}

	return resp, nil
}

func resolveReadmeVersion(ctx context.Context, canvas *models.Canvas, versionID string) (*models.CanvasVersion, error) {
	requested := strings.TrimSpace(versionID)

	if requested == "" {
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), canvas)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, status.Error(codes.FailedPrecondition, "canvas has no live version")
			}
			return nil, status.Error(codes.Internal, "failed to load live version")
		}
		return version, nil
	}

	if strings.EqualFold(requested, "draft") {
		userID, ok := authentication.GetUserIdFromMetadata(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "user not authenticated")
		}
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "invalid user id in context")
		}
		version, err := models.FindCanvasDraftInTransaction(database.Conn(), canvas.ID, userUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, status.Error(codes.NotFound, "no draft version for user")
			}
			return nil, status.Error(codes.Internal, "failed to load draft version")
		}
		return version, nil
	}

	versionUUID, err := uuid.Parse(requested)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version_id: %v", err)
	}

	version, err := models.FindCanvasVersion(canvas.ID, versionUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "version not found")
		}
		return nil, status.Error(codes.Internal, "failed to load version")
	}
	return version, nil
}
