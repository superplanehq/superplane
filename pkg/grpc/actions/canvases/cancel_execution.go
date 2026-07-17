package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func CancelExecution(ctx context.Context, authService authorization.Authorization, encryptor crypto.Encryptor, organizationID string, registry *registry.Registry, workflowID, executionID uuid.UUID) (*pb.CancelExecutionResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.PermissionDenied(nil, "user not authenticated")
	}

	user, err := models.FindActiveUserByID(organizationID, userID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "user not found")
	}

	var cancelled bool
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		execution, err := models.FindNodeExecutionInTransaction(tx, workflowID, executionID)
		if err != nil {
			return grpcerrors.NotFound(err, "execution not found")
		}

		//
		// Execution is already finished or cancelling, do nothing.
		//
		if execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
			return nil
		}

		cancelled = true
		return execution.RequestCancellation(tx, &user.ID)
	})

	if err != nil {
		return nil, err
	}

	if cancelled {
		if err := messages.PublishCanvasExecutionByID(workflowID, executionID); err != nil {
			log.Errorf("failed to publish execution cancelling RabbitMQ message: %v", err)
		}
	}

	return &pb.CancelExecutionResponse{}, nil
}
