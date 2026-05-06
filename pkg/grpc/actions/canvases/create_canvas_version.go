package canvases

import (
	"context"
	"errors"

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

func CreateCanvasVersion(ctx context.Context, organizationID string, canvasID string) (*pb.CreateCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgUUID := uuid.MustParse(organizationID)
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		liveVersion, liveVersionErr := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
		if liveVersionErr != nil {
			if errors.Is(liveVersionErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.FailedPrecondition, "canvas live version not found")
			}
			return liveVersionErr
		}

		log.WithFields(log.Fields{
			"action":           "CreateCanvasVersion",
			"canvas_id":        canvasID,
			"live_version_id":  liveVersion.ID.String(),
			"live_readme_len":  len(liveVersion.Readme),
		}).Info("readme-debug: CreateCanvasVersion cloning from live")

		version, err = models.SaveCanvasDraftWithReadmeInTransaction(
			tx,
			canvas.ID,
			userUUID,
			liveVersion.Nodes,
			liveVersion.Edges,
			liveVersion.Readme,
		)
		if version != nil {
			log.WithFields(log.Fields{
				"action":           "CreateCanvasVersion",
				"canvas_id":        canvasID,
				"new_version_id":   version.ID.String(),
				"new_readme_len":   len(version.Readme),
			}).Info("readme-debug: CreateCanvasVersion produced draft")
		}

		return err
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create canvas version: %v", err)
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.CreateCanvasVersionResponse{
		Version: SerializeCanvasVersion(version, organizationID),
	}, nil
}
