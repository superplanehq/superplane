package canvases

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func CreateCanvasVersion(
	ctx context.Context,
	organizationID string,
	canvasID string,
	displayName string,
) (*pb.CreateCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var createErr error
		version, createErr = models.CreateDraftBranchFromLiveInTransaction(
			tx,
			canvasUUID,
			userUUID,
			strings.TrimSpace(displayName),
			nil,
			nil,
		)
		return createErr
	})
	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, grpcerrors.Internal(err, "failed to create canvas version")
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
	}

	return &pb.CreateCanvasVersionResponse{
		Version: SerializeCanvasVersion(canvas, version, organizationID, nil),
	}, nil
}
