package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func DeleteCanvasStaging(ctx context.Context, db *gorm.DB, canvas *models.Canvas, paths []string) (*pb.StagingSummary, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	if err := models.DiscardStagedFilesForUser(db, canvas.ID, uuid.MustParse(userID), paths); err != nil {
		return nil, grpcerrors.Internal(err, "failed to discard staging")
	}

	if err := messages.NewCanvasStagingMessage(canvas.ID.String(), userID).Publish(); err != nil {
		log.Errorf("failed to publish canvas staging updated RabbitMQ message: %v", err)
	}

	return buildStagingSummary(canvas, []models.WorkflowStagedFile{}), nil
}
