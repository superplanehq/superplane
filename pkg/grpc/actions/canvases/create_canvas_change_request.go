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
	"gorm.io/gorm"
)

func CreateCanvasChangeRequest(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
) (*pb.CreateCanvasChangeRequestResponse, error) {
	return CreateCanvasChangeRequestWithMetadata(ctx, organizationID, canvasID, versionID, "", "")
}

func CreateCanvasChangeRequestWithMetadata(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	title string,
	description string,
) (*pb.CreateCanvasChangeRequestResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	requestedVersionID := strings.TrimSpace(versionID)
	requestedTitle := strings.TrimSpace(title)
	requestedDescription := description
	var requestedVersionUUID *uuid.UUID
	if requestedVersionID != "" {
		versionUUID, parseErr := uuid.Parse(requestedVersionID)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", parseErr)
		}
		requestedVersionUUID = &versionUUID
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	versioningEnabled, modeErr := isCanvasVersioningEnabledForCanvas(canvas)
	if modeErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to load canvas versioning: %v", modeErr)
	}
	if !versioningEnabled {
		return nil, status.Error(codes.FailedPrecondition, "canvas versioning is disabled for this canvas")
	}

	userUUID := uuid.MustParse(userID)
	var request *models.CanvasChangeRequest
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasInTx, findCanvasErr := models.FindCanvasInTransaction(tx, uuid.MustParse(organizationID), canvasUUID)
		if findCanvasErr != nil {
			return findCanvasErr
		}
		if canvasInTx.LiveVersionID == nil {
			return status.Error(codes.FailedPrecondition, "canvas live version not found")
		}

		draft, findDraftErr := models.FindCanvasDraftInTransaction(tx, canvasUUID, userUUID)
		if findDraftErr != nil {
			if errors.Is(findDraftErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.FailedPrecondition, "no edit version found for this user")
			}
			return findDraftErr
		}

		if requestedVersionUUID != nil && draft.VersionID != *requestedVersionUUID {
			return status.Error(codes.FailedPrecondition, "version is not your current edit version")
		}

		draftVersion, findVersionErr := models.FindCanvasVersionInTransaction(tx, canvasUUID, draft.VersionID)
		if findVersionErr != nil {
			if errors.Is(findVersionErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "edit version not found")
			}
			return findVersionErr
		}
		if draftVersion.OwnerID == nil || *draftVersion.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}
		if draftVersion.IsPublished {
			return status.Error(codes.FailedPrecondition, "published versions cannot create change requests")
		}

		version, err = models.CreateCanvasSnapshotVersionInTransaction(
			tx,
			canvasUUID,
			userUUID,
			draftVersion.Nodes,
			draftVersion.Edges,
		)
		if err != nil {
			return err
		}

		now := time.Now()
		request = &models.CanvasChangeRequest{
			ID:               uuid.New(),
			WorkflowID:       canvasUUID,
			VersionID:        version.ID,
			OwnerID:          &userUUID,
			BasedOnVersionID: canvasInTx.LiveVersionID,
			Title:            requestedTitle,
			Description:      requestedDescription,
			Status:           models.CanvasChangeRequestStatusOpen,
			CreatedAt:        &now,
			UpdatedAt:        &now,
		}
		if request.Title == "" {
			request.Title = "Update " + canvasInTx.Name
		}

		if createErr := tx.Create(request).Error; createErr != nil {
			return createErr
		}

		return refreshCanvasChangeRequestDiffInTransaction(tx, canvasInTx, version, request)
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to create canvas change request: %v", err)
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.CreateCanvasChangeRequestResponse{
		ChangeRequest: SerializeCanvasChangeRequest(request, version, organizationID),
	}, nil
}
