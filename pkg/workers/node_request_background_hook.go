package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const backgroundHookLeaseDuration = 5 * time.Minute

type backgroundHookInvocation struct {
	request   *models.CanvasNodeRequest
	node      *models.CanvasNode
	canvas    *models.Canvas
	execution *models.CanvasNodeExecution
	input     any
	handler   core.BackgroundHookHandler
}

func (e *NodeRequestExecutor) executeBackgroundHook(request models.CanvasNodeRequest, lockRequest func(*gorm.DB, uuid.UUID) (*models.CanvasNodeRequest, error)) (bool, error) {
	if request.Type != models.NodeRequestTypeInvokeAction || request.Spec.Data().InvokeAction == nil {
		return false, nil
	}
	db := database.DB(context.Background())
	var invocation *backgroundHookInvocation
	handled := false
	err := db.Transaction(func(tx *gorm.DB) error {
		stored, err := lockRequest(tx, request.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handled = true
			return nil
		}
		if err != nil {
			return err
		}
		invocation, err = e.claimBackgroundHook(tx, stored)
		handled = invocation != nil
		return err
	})
	if err != nil || invocation == nil {
		return handled, err
	}
	if invocation.handler == nil {
		return true, nil
	}

	hookContext, err := e.backgroundHookContext(db, invocation)
	result := core.BackgroundHookResult{}
	if err == nil {
		result, err = invocation.handler.HandleBackgroundHook(hookContext)
	}
	if err != nil {
		result.RetryAfter = time.Minute
	}
	finalizeErr := db.Transaction(func(tx *gorm.DB) error {
		if result.RetryAfter > 0 {
			_, finishErr := invocation.request.RescheduleLease(tx, time.Now().Add(result.RetryAfter))
			return finishErr
		}
		_, finishErr := invocation.request.CompleteLease(tx)
		return finishErr
	})
	return true, errors.Join(err, finalizeErr)
}

func (e *NodeRequestExecutor) claimBackgroundHook(tx *gorm.DB, request *models.CanvasNodeRequest) (*backgroundHookInvocation, error) {
	node, err := models.FindUnscopedCanvasNode(tx, request.WorkflowID, request.NodeID)
	if err != nil {
		return nil, err
	}
	component := node.Ref.Data().Component
	if node.Type != models.NodeTypeComponent || component == nil || node.DeletedAt.Valid {
		return nil, nil
	}
	invocation := &backgroundHookInvocation{request: request, node: node}
	componentName := component.Name
	if request.ExecutionID != nil {
		invocation.execution, err = models.FindNodeExecutionInTransaction(tx, request.WorkflowID, *request.ExecutionID)
		if err != nil {
			return nil, err
		}
		if frozen := invocation.execution.FrozenComponentName(); frozen != "" {
			componentName = frozen
		}
	}
	provider, _, err := e.registry.FindActionHook(componentName, request.Spec.Data().InvokeAction.ActionName)
	if err != nil {
		// The ordinary executor retains validation for components without a
		// background handler, including finished executions and deleted nodes.
		return nil, nil
	}
	invocation.handler, _ = provider.(core.BackgroundHookHandler)
	if wrapped, ok := provider.(interface {
		BackgroundHookHandler() core.BackgroundHookHandler
	}); ok {
		invocation.handler = wrapped.BackgroundHookHandler()
	}
	if invocation.handler == nil {
		return nil, nil
	}
	if invocation.execution != nil {
		if invocation.execution.State == models.CanvasNodeExecutionStateFinished || invocation.execution.State == models.CanvasNodeExecutionStateCancelling {
			invocation.handler = nil
			return invocation, request.Complete(tx)
		}
		input, err := models.FindCanvasEventInTransaction(tx, invocation.execution.EventID)
		if err != nil {
			return nil, err
		}
		invocation.input = input.Data.Data()
	}
	invocation.canvas, err = models.FindCanvasWithoutOrgScopeInTransaction(tx, request.WorkflowID)
	if err != nil {
		return nil, err
	}
	return invocation, request.ClaimLease(tx, backgroundHookLeaseDuration)
}

func (e *NodeRequestExecutor) backgroundHookContext(db *gorm.DB, invocation *backgroundHookInvocation) (core.BackgroundHookContext, error) {
	request, node, canvas, execution := invocation.request, invocation.node, invocation.canvas, invocation.execution
	action := request.Spec.Data().InvokeAction
	logger := logging.WithNode(e.logger, *node)
	publishEvents := func(events []models.CanvasEvent) {
		for i := range events {
			if err := messages.PublishCanvasEventCreatedMessage(&events[i]); err != nil {
				logger.WithError(err).Warnf("Failed to publish background hook event %s", events[i].ID)
			}
		}
	}
	hook := core.BackgroundHookContext{
		WorkflowID: request.WorkflowID, NodeID: request.NodeID, OrganizationID: canvas.OrganizationID,
		Data: invocation.input,
		ActionHookContext: core.ActionHookContext{
			Name: action.ActionName, Parameters: action.Parameters, Configuration: node.Configuration.Data(),
			Logger: logger, HTTP: e.registry.HTTPContext(),
			Metadata: contexts.NewNodeMetadataContext(db, node),
			Requests: contexts.NewBackgroundRequestContext(db, request, node, execution),
			Auth:     contexts.NewAuthReader(db, canvas.OrganizationID, e.authService, nil),
			Secrets:  contexts.NewSecretsContext(db, e.registry, canvas.OrganizationID, e.encryptor),
		},
	}
	if execution != nil {
		hook.ExecutionID, hook.RunID = execution.ID, execution.RunID
		hook.Configuration = execution.Configuration.Data()
		hook.Logger = logging.WithExecution(logger, execution)
		hook.Metadata = contexts.NewExecutionMetadataContext(db, execution)
		hook.ExecutionState = contexts.NewBackgroundExecutionStateContext(db, request, execution, publishEvents)
		hook.Runs = contexts.NewRunExecutionContext(db, canvas, node, execution)
		hook.Factory = contexts.NewFactoryContext(db, canvas, execution).
			WithNewEvents(publishEvents).
			WithRemoteImageIngest(e.encryptor, e.registry)
		hook.Usage = contexts.NewUsageContext(canvas.OrganizationID, execution)
		hook.HostedLLM = contexts.NewHostedLLMContext(db, e.encryptor, canvas.OrganizationID, canvas.FactoryID)
		hook.RunnerTasks = contexts.NewRunnerTaskContext(db, e.encryptor, canvas.OrganizationID)
	}
	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(db, *node.AppInstallationID)
		if err != nil {
			return hook, fmt.Errorf("find background hook integration: %w", err)
		}
		hook.Integration = contexts.NewIntegrationContext(db, node, integration, e.encryptor, e.registry, publishEvents)
	}
	return hook, nil
}
