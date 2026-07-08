package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func DeleteCanvasStaging(ctx context.Context, organizationID string, canvasID string, paths []string) (*pb.Staging, error) {
	db := database.DB(ctx)

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	canvas, err := models.FindCanvasInTransaction(db, organizationUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	userUUID := uuid.MustParse(userID)
	if err := models.DiscardStagedFilesForUser(db, canvas.ID, userUUID, paths); err != nil {
		return nil, grpcerrors.Internal(err, "failed to discard staging")
	}

	if err := messages.NewCanvasStagingMessage(canvas.ID.String(), userID).Publish(); err != nil {
		log.Errorf("failed to publish canvas staging updated RabbitMQ message: %v", err)
	}

	rows, err := models.ListStagedFilesForUser(db, canvas.ID, userUUID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	return buildStaging(ctx, canvas, rows)
}
