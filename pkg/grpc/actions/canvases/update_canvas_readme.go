package canvases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// UpdateCanvasReadme sets the readme on the caller's draft version.
//
// If version_id is empty, the caller's existing draft is used; when no draft
// exists, one is cloned from the current live version so that updating the
// readme never requires a prior explicit draft creation.
//
// If version_id is provided, it must reference the caller's draft.
func UpdateCanvasReadme(ctx context.Context, organizationID, canvasID, versionID, content string) (*pb.UpdateCanvasReadmeResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid user id in context")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	requestedVersionID := strings.TrimSpace(versionID)
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if requestedVersionID != "" {
			versionUUID, parseErr := uuid.Parse(requestedVersionID)
			if parseErr != nil {
				return status.Errorf(codes.InvalidArgument, "invalid version_id: %v", parseErr)
			}

			v, findErr := models.FindCanvasVersionInTransaction(tx, canvasUUID, versionUUID)
			if findErr != nil {
				if errors.Is(findErr, gorm.ErrRecordNotFound) {
					return status.Error(codes.NotFound, "version not found")
				}
				return findErr
			}

			if v.State == models.CanvasVersionStatePublished {
				return status.Error(codes.FailedPrecondition, "published versions are immutable")
			}
			if v.OwnerID == nil || *v.OwnerID != userUUID {
				return status.Error(codes.PermissionDenied, "version owner mismatch")
			}
			if _, draftErr := models.FindCanvasDraftByVersionInTransaction(tx, canvasUUID, userUUID, v.ID); draftErr != nil {
				if errors.Is(draftErr, gorm.ErrRecordNotFound) {
					return status.Error(codes.FailedPrecondition, "version is not your current edit version")
				}
				return draftErr
			}
			version = v
		} else {
			// Clone a fresh draft from the live version if none exists.
			liveVersion, liveErr := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
			if liveErr != nil {
				if errors.Is(liveErr, gorm.ErrRecordNotFound) {
					return status.Error(codes.FailedPrecondition, "canvas has no live version")
				}
				return liveErr
			}

			draft, draftErr := models.SaveCanvasDraftWithReadmeInTransaction(
				tx,
				canvas.ID,
				userUUID,
				liveVersion.Nodes,
				liveVersion.Edges,
				liveVersion.Readme,
			)
			if draftErr != nil {
				return draftErr
			}
			version = draft
		}

		return models.UpdateCanvasVersionReadmeInTransaction(tx, version, content)
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		log.WithError(err).Error("failed to update canvas readme")
		return nil, status.Error(codes.Internal, "failed to update canvas readme")
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	resp := &pb.UpdateCanvasReadmeResponse{
		CanvasId:  canvasUUID.String(),
		VersionId: version.ID.String(),
		Content:   version.Readme,
	}
	if version.UpdatedAt != nil {
		resp.UpdatedAt = timestamppb.New(*version.UpdatedAt)
	} else {
		resp.UpdatedAt = timestamppb.New(time.Now())
	}
	return resp, nil
}
