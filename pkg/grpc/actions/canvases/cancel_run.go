package canvases

import (
	"context"
	goerrors "errors"

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

// CancelRun cancels an entire run in a single atomic operation: it cancels every
// active execution, deletes all pending queue items, completes pending requests,
// and finalizes the run with result=cancelled. External component Cancel hooks
// are invoked best-effort after the run transaction commits, so the run lock is
// not held for the duration of remote calls.
func CancelRun(ctx context.Context, authService authorization.Authorization, encryptor crypto.Encryptor, organizationID string, registry *registry.Registry, workflowID, runID uuid.UUID) (*pb.CancelRunResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	var user *models.User
	if userIsSet {
		var err error
		user, err = models.FindActiveUserByID(organizationID, userID)
		if err != nil {
			return nil, grpcerrors.NotFound(err, "user not found")
		}
	}
	// If user is not set (like in tests), user will be nil and that's fine.

	var cancelledBy *uuid.UUID
	if user != nil {
		cancelledBy = &user.ID
	}

	var result *models.CancelRunResult
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = models.CancelRun(tx, workflowID, runID, cancelledBy)
		return err
	})

	if err != nil {
		if goerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "run not found")
		}

		return nil, grpcerrors.Internal(err, "failed to cancel run")
	}

	//
	// Cancelling an already-finished run is an idempotent no-op.
	//
	if result.AlreadyFinished {
		return &pb.CancelRunResponse{}, nil
	}

	//
	// The run is already cancelled and finalized in the database. Announce the
	// state changes over RabbitMQ first so the UI updates promptly, then run the
	// best-effort external Cancel hooks for the executions that were active.
	//
	if err := messages.NewCanvasRunMessage(workflowID.String(), runID.String()).Publish(); err != nil {
		log.Errorf("failed to publish run state RabbitMQ message: %v", err)
	}

	for _, execution := range result.CancelledExecutions {
		if err := messages.PublishCanvasExecutionByID(workflowID, execution.ID); err != nil {
			log.Errorf("failed to publish execution state RabbitMQ message: %v", err)
		}
	}

	memoryChanged := invokeRunExecutionCancelHooks(ctx, authService, encryptor, organizationID, registry, workflowID, result.CancelledExecutions, user)

	if memoryChanged {
		if err := messages.NewCanvasMemoryUpdatedMessage(workflowID.String()).PublishMemoryUpdated(); err != nil {
			log.Errorf("failed to publish canvas memory updated RabbitMQ message: %v", err)
		}
	}

	return &pb.CancelRunResponse{}, nil
}

// invokeRunExecutionCancelHooks runs the best-effort external Cancel hook for
// each cancelled execution, outside the run lock. Failures are logged and do not
// abort the operation — the run is already correctly finalized as cancelled.
func invokeRunExecutionCancelHooks(ctx context.Context, authService authorization.Authorization, encryptor crypto.Encryptor, organizationID string, registry *registry.Registry, workflowID uuid.UUID, executions []models.CanvasNodeExecution, user *models.User) bool {
	memoryChanged := false
	onMemoryChanged := func() {
		memoryChanged = true
	}

	for i := range executions {
		execution := executions[i]
		err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
			node, err := models.FindCanvasNode(tx, workflowID, execution.NodeID)
			if err != nil {
				return err
			}

			return invokeExecutionCancelHook(tx, authService, encryptor, organizationID, registry, &execution, node, user, onMemoryChanged)
		})
		if err != nil {
			log.Errorf("failed to run cancel hook for execution %s: %v", execution.ID, err)
		}
	}

	return memoryChanged
}
