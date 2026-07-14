package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func CancelExecution(ctx context.Context, authService authorization.Authorization, encryptor crypto.Encryptor, organizationID string, registry *registry.Registry, workflowID, executionID uuid.UUID) (*pb.CancelExecutionResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	var user *models.User
	if userIsSet {
		var err error
		user, err = models.FindActiveUserByID(organizationID, userID)
		if err != nil {
			return nil, grpcerrors.NotFound(err, "user not found")
		}
	}
	// If user is not set (like in tests), user will be nil and that's fine

	execution, err := models.FindNodeExecution(workflowID, executionID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "execution not found")
	}

	memoryChanged := false
	onMemoryChanged := func() {
		memoryChanged = true
	}
	var finishedRunIDs []uuid.UUID

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		node, err := models.FindCanvasNode(tx, workflowID, execution.NodeID)

		if err != nil {
			return grpcerrors.NotFound(err, "Node not found for execution")
		}

		err = cancelExecutionInTransaction(tx, authService, encryptor, organizationID, registry, execution, node, user, onMemoryChanged)

		if err != nil {
			return grpcerrors.Internal(err, "failed to cancel execution")
		}

		finishedRunIDs, err = models.FinishCanvasRunsWithNoOpenWork(tx, workflowID, []uuid.UUID{execution.RunID})
		if err != nil {
			return grpcerrors.Internal(err, "failed to finish run")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := messages.PublishCanvasExecutionByID(workflowID, execution.ID); err != nil {
		log.Errorf("failed to publish execution state RabbitMQ message: %v", err)
	}

	publishFinishedRunMessages(workflowID, finishedRunIDs)

	if memoryChanged {
		if err := messages.NewCanvasMemoryUpdatedMessage(workflowID.String()).PublishMemoryUpdated(); err != nil {
			log.Errorf("failed to publish canvas memory updated RabbitMQ message: %v", err)
		}
	}

	return &pb.CancelExecutionResponse{}, nil
}

func cancelExecutionInTransaction(tx *gorm.DB, authService authorization.Authorization, encryptor crypto.Encryptor, organizationID string, registry *registry.Registry, execution *models.CanvasNodeExecution, node *models.CanvasNode, user *models.User, onMemoryChanged func()) error {
	if node.Type != models.NodeTypeComponent {
		return nil
	}

	ref := node.Ref.Data()
	if ref.Component != nil {
		action, err := registry.GetAction(ref.Component.Name)
		if err != nil {
			log.Errorf("action %s not found: %v", ref.Component.Name, err)
			return err
		}

		logger := logging.ForExecution(execution)
		orgUUID := uuid.MustParse(organizationID)
		canvasName := ""
		if workflow, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID); err == nil && workflow != nil {
			canvasName = workflow.Name
		}
		ctx := core.ExecutionContext{
			ID:             execution.ID,
			WorkflowID:     execution.WorkflowID.String(),
			OrganizationID: organizationID,
			CanvasName:     canvasName,
			NodeID:         execution.NodeID,
			NodeName:       node.Name,
			Configuration:  execution.Configuration.Data(),
			HTTP:           registry.HTTPContextInTransaction(tx),
			Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
			ExecutionState: contexts.NewExecutionStateContext(tx, execution, nil),
			Requests:       contexts.NewExecutionRequestContext(tx, execution),
			Auth:           contexts.NewAuthReader(tx, orgUUID, authService, user),
			CanvasMemory:   contexts.NewCanvasMemoryContext(tx, execution.WorkflowID).WithChangeCallback(onMemoryChanged),
		}

		if node.AppInstallationID != nil {
			integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
			if err != nil {
				logger.Errorf("error finding app installation: %v", err)
				return grpcerrors.Internal(err, "error building context")
			}

			logger = logging.WithIntegration(logger, *integration)
			ctx.Integration = contexts.NewIntegrationContext(tx, node, integration, encryptor, registry, nil)
		}

		ctx.Logger = logger
		if err := action.Cancel(ctx); err != nil {
			log.Errorf("failed to cancel component execution %s: %v", execution.ID.String(), err)
		}
	}

	var cancelledBy *uuid.UUID
	if user != nil {
		cancelledBy = &user.ID
	}

	return execution.CancelInTransaction(tx, cancelledBy)
}
