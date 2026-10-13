package canvases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func InvokeNodeExecutionHook(
	ctx context.Context,
	authService authorization.Authorization,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	db *gorm.DB,
	canvas *models.Canvas,
	executionID uuid.UUID,
	hookName string,
	parameters map[string]any,
) (*pb.InvokeNodeExecutionHookResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	var execution *models.CanvasNodeExecution
	var logger *log.Entry
	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		//
		// Hooks read the execution metadata, change it and write the whole
		// document back, so the row has to stay locked from the read until the
		// commit. Otherwise two people invoking a hook at the same time start
		// from the same metadata and the second write drops the first decision.
		//
		var err error
		execution, err = models.LockNodeExecution(tx, canvas.ID, executionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return grpcerrors.NotFound(err, "execution not found")
			}

			return fmt.Errorf("failed to lock execution: %w", err)
		}

		//
		// Components decide this on the snapshot they are handed, or do not
		// check it at all, so the state is verified here, under the lock.
		//
		if execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
			return grpcerrors.FailedPrecondition(nil, fmt.Sprintf("execution is %s", execution.State))
		}

		node, err := models.FindCanvasNode(tx, canvas.ID, execution.NodeID)
		if err != nil {
			return fmt.Errorf("node not found: %w", err)
		}

		if node.Type != models.NodeTypeComponent || node.Ref.Data().Component == nil {
			return grpcerrors.InvalidArgument(nil, "node is not a component node")
		}

		hookProvider, hookDef, err := registry.FindActionHook(node.Ref.Data().Component.Name, hookName)
		if err != nil {
			return fmt.Errorf("hook not found: %w", err)
		}

		if hookDef.Type != core.HookTypeUser {
			return fmt.Errorf("hook '%s' cannot be invoked", hookName)
		}

		if err := configuration.ValidateConfiguration(hookDef.Parameters, parameters); err != nil {
			return fmt.Errorf("hook parameters validation failed: %w", err)
		}

		user, err := models.FindActiveUserByIDInTransaction(tx, canvas.OrganizationID.String(), userID)
		if err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		logger = logging.ForExecution(execution)
		actionCtx := core.ActionHookContext{
			Name:           hookName,
			Parameters:     parameters,
			Configuration:  node.Configuration.Data(),
			HTTP:           registry.HTTPContextInTransaction(tx),
			Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
			ExecutionState: contexts.NewExecutionStateContext(tx, execution, onNewEvents),
			Auth:           contexts.NewAuthReader(tx, canvas.OrganizationID, authService, user),
			Requests:       contexts.NewExecutionRequestContext(tx, execution),
			Runs:           contexts.NewRunExecutionContext(tx, canvas, node, execution),
		}

		if node.AppInstallationID != nil {
			integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
			if err != nil {
				logger.Errorf("error finding app installation: %v", err)
				return grpcerrors.Internal(err, "error building context")
			}

			logger = logging.WithIntegration(logger, *integration)
			actionCtx.Integration = contexts.NewIntegrationContext(tx, node, integration, encryptor, registry, onNewEvents)
		}

		actionCtx.Logger = logger
		if err := hookProvider.HandleHook(actionCtx); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				// A database conflict (deadlock, timeout) is not a client error.
				return fmt.Errorf("action execution failed: %w", err)
			}

			return grpcerrors.InvalidArgument(err, "action execution failed")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	//
	// Announce the new state only once it is committed.
	//
	if err := messages.PublishCanvasExecutionByID(execution.WorkflowID, execution.ID); err != nil {
		logger.Errorf("failed to publish execution state RabbitMQ message: %v", err)
	}

	for _, event := range newEvents {
		if err := messages.PublishCanvasEventCreatedMessage(&event); err != nil {
			logger.Errorf("failed to publish canvas event created RabbitMQ message: %v", err)
		}
	}

	return &pb.InvokeNodeExecutionHookResponse{}, nil
}
