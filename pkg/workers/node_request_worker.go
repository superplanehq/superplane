package workers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type NodeRequestWorker struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	encryptor crypto.Encryptor
}

func NewNodeRequestWorker(encryptor crypto.Encryptor, registry *registry.Registry) *NodeRequestWorker {
	return &NodeRequestWorker{
		encryptor: encryptor,
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *NodeRequestWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()

			requests, err := models.ListNodeRequests()
			if err != nil {
				w.log("Error finding workflow nodes ready to be processed: %v", err)
			}

			telemetry.RecordNodeRequestWorkerRequestsCount(context.Background(), len(requests))

			for _, request := range requests {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(request models.CanvasNodeRequest) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessRequest(request); err != nil {
						w.log("Error processing request %s: %v", request.ID, err)
					}

					if request.ExecutionID != nil {
						messages.NewCanvasExecutionMessage(request.WorkflowID.String(), request.ExecutionID.String(), request.NodeID).Publish()
					}
				}(request)
			}

			telemetry.RecordNodeRequestWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *NodeRequestWorker) LockAndProcessRequest(request models.CanvasNodeRequest) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockNodeRequest(tx, request.ID)
		if err != nil {
			w.log("Request %s already being processed - skipping", request.ID)
			return nil
		}

		return w.processRequest(tx, r)
	})
}

func (w *NodeRequestWorker) processRequest(tx *gorm.DB, request *models.CanvasNodeRequest) error {
	switch request.Type {
	case models.NodeRequestTypeInvokeAction:
		return w.invokeAction(tx, request)
	}

	return fmt.Errorf("unsupported node execution request type %s", request.Type)
}

func (w *NodeRequestWorker) invokeAction(tx *gorm.DB, request *models.CanvasNodeRequest) error {
	if request.ExecutionID == nil {
		return w.invokeTriggerAction(tx, request)
	}

	return w.invokeComponentAction(tx, request)
}

func (w *NodeRequestWorker) invokeTriggerAction(tx *gorm.DB, request *models.CanvasNodeRequest) error {
	node, err := models.FindCanvasNode(tx, request.WorkflowID, request.NodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			request.Fail(tx, err.Error())
			return nil
		}

		return fmt.Errorf("failed to find node: %w", err)
	}

	logger := logging.ForNode(*node)

	trigger, err := w.registry.GetTrigger(node.Ref.Data().Trigger.Name)
	if err != nil {
		logger.Errorf("Failure invoking action: failed to find trigger: %v", err)
		request.Fail(tx, err.Error())
		return nil
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		logger.Errorf("Failure invoking action: spec is not specified")
		request.Fail(tx, "spec is not specified")
		return nil
	}

	actionName := spec.InvokeAction.ActionName
	actionDef := findAction(trigger.Actions(), actionName)
	if actionDef == nil {
		logger.Errorf("Failure invoking action: action '%s' not found for trigger '%s'", actionName, trigger.Name())
		request.Fail(tx, fmt.Sprintf("action '%s' not found for trigger '%s'", actionName, trigger.Name()))
		return nil
	}

	actionCtx := core.TriggerActionContext{
		Name:          actionName,
		Parameters:    spec.InvokeAction.Parameters,
		Configuration: node.Configuration.Data(),
		Logger:        logging.ForNode(*node),
		HTTP:          w.registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataContext(tx, node),
		Events:        contexts.NewEventContext(tx, node),
		Requests:      contexts.NewNodeRequestContext(tx, node),
	}

	if node.AppInstallationID != nil {
		instance, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Errorf("Failure invoking action: integration %s not found", *node.AppInstallationID)
				request.Fail(tx, fmt.Sprintf("integration %s not found", *node.AppInstallationID))
				return nil
			}

			return fmt.Errorf("failed to find integration: %v", err)
		}

		actionCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, w.encryptor, w.registry)
	}

	_, err = trigger.HandleAction(actionCtx)
	if err != nil {
		return w.handleRetryStrategy(tx, logger, request, err)
	}

	err = tx.Save(&node).Error
	if err != nil {
		return fmt.Errorf("error saving node after action handler: %v", err)
	}

	return request.Pass(tx)
}

func (w *NodeRequestWorker) handleRetryStrategy(tx *gorm.DB, logger *log.Entry, request *models.CanvasNodeRequest, err error) error {
	retryStrategy := request.RetryStrategy.Data()
	if retryStrategy.Type == "" {
		logger.Errorf("Retry strategy type is not specified: %v", err)
		request.Fail(tx, err.Error())
		return nil
	}

	if retryStrategy.Type == models.RetryStrategyTypeConstant {
		return w.handleConstantRetryStrategy(tx, request, retryStrategy.Constant, logger, err)
	}

	logger.Errorf("Unknown retry strategy type: %s", retryStrategy.Type)
	return fmt.Errorf("unknown retry strategy type: %s", retryStrategy.Type)
}

func (w *NodeRequestWorker) handleConstantRetryStrategy(tx *gorm.DB, request *models.CanvasNodeRequest, retryStrategy *models.ConstantRetryStrategy, logger *log.Entry, err error) error {
	attempts := request.Attempts + 1

	logger.Infof("Attempt %d / %d failed for request %s: %v", attempts, retryStrategy.MaxAttempts, request.ID, err)

	if attempts > retryStrategy.MaxAttempts-1 {
		logger.Errorf("Max attempts reached for request %s - completing: %v", request.ID, err)
		return request.Fail(tx, fmt.Sprintf("max attempts reached: %v", err))
	}

	nextRunAt, err := retryStrategy.NextRunAt()
	if err != nil {
		logger.Errorf("Failed to get next run at for request %s: %v", request.ID, err)
		return request.Fail(tx, fmt.Sprintf("failed to get next run at: %v", err))
	}

	request.RunAt = *nextRunAt
	request.Attempts = attempts
	err = tx.Save(request).Error
	if err != nil {
		return fmt.Errorf("failed to save request: %w", err)
	}

	logger.Infof("Next run at %s for request %s", nextRunAt.Format(time.RFC3339), request.ID)
	return nil
}

func (w *NodeRequestWorker) invokeComponentAction(tx *gorm.DB, request *models.CanvasNodeRequest) error {
	execution, err := models.FindNodeExecutionInTransaction(tx, request.WorkflowID, *request.ExecutionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			request.Fail(tx, fmt.Sprintf("execution %s not found", request.ExecutionID))
			return nil
		}

		return fmt.Errorf("failed to find execution: %w", err)
	}

	logger := logging.ForExecution(execution, nil)
	if execution.ParentExecutionID == nil {
		return w.invokeParentNodeComponentAction(tx, logger, request, execution)
	}

	return w.invokeChildNodeComponentAction(tx, logger, request, execution)
}

func (w *NodeRequestWorker) invokeParentNodeComponentAction(tx *gorm.DB, logger *log.Entry, request *models.CanvasNodeRequest, execution *models.CanvasNodeExecution) error {
	node, err := models.FindCanvasNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			request.Fail(tx, fmt.Sprintf("node %s not found", execution.NodeID))
			return nil
		}

		logger.Errorf("Failure invoking action: failed to find node: %v", err)
		return fmt.Errorf("failed to find node: %w", err)
	}

	component, err := w.registry.GetComponent(node.Ref.Data().Component.Name)
	if err != nil {
		logger.Errorf("Failure invoking action: failed to find component: %v", err)
		request.Fail(tx, fmt.Sprintf("component %s not found", node.Ref.Data().Component.Name))
		return nil
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		logger.Errorf("Failure invoking action: spec is not specified")
		request.Fail(tx, "spec is not specified")
		return nil
	}

	actionName := spec.InvokeAction.ActionName
	actionDef := findAction(component.Actions(), actionName)
	if actionDef == nil {
		logger.Errorf("Failure invoking action: action '%s' not found for component '%s'", actionName, component.Name())
		request.Fail(tx, fmt.Sprintf("action '%s' not found for component '%s'", actionName, component.Name()))
		return nil
	}

	actionCtx := core.ActionContext{
		Name:           actionName,
		Configuration:  node.Configuration.Data(),
		Parameters:     spec.InvokeAction.Parameters,
		HTTP:           w.registry.HTTPContext(),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Notifications:  contexts.NewNotificationContext(tx, uuid.Nil, node.WorkflowID),
	}

	if node.AppInstallationID != nil {
		instance, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Errorf("Failure invoking action: integration %s not found", *node.AppInstallationID)
				request.Fail(tx, fmt.Sprintf("integration %s not found", *node.AppInstallationID))
				return nil
			}

			logger.Errorf("Failure invoking action: failed to find integration: %v", err)
			return fmt.Errorf("failed to find integration: %v", err)
		}

		logger = logging.WithIntegration(logger, *instance)
		actionCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, w.encryptor, w.registry)
	}

	actionCtx.Logger = logger
	err = component.HandleAction(actionCtx)
	if err != nil {
		return w.handleRetryStrategy(tx, logger, request, err)
	}

	err = tx.Save(&execution).Error
	if err != nil {
		return fmt.Errorf("error saving execution after action handler: %v", err)
	}

	return request.Pass(tx)
}

func (w *NodeRequestWorker) invokeChildNodeComponentAction(tx *gorm.DB, logger *log.Entry, request *models.CanvasNodeRequest, execution *models.CanvasNodeExecution) error {
	parentExecution, err := models.FindNodeExecutionInTransaction(tx, execution.WorkflowID, *execution.ParentExecutionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			request.Fail(tx, fmt.Sprintf("parent execution %s not found", execution.ParentExecutionID))
			return nil
		}

		logger.Errorf("Failure invoking action: failed to find parent execution: %v", err)
		return fmt.Errorf("failed to find parent execution: %w", err)
	}

	parentNode, err := models.FindCanvasNode(tx, execution.WorkflowID, parentExecution.NodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			request.Fail(tx, fmt.Sprintf("node %s not found", execution.NodeID))
			return nil
		}

		logger.Errorf("Failure invoking action: failed to find node: %v", err)
		return fmt.Errorf("failed to find node: %w", err)
	}

	blueprint, err := models.FindUnscopedBlueprintInTransaction(tx, parentNode.Ref.Data().Blueprint.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			request.Fail(tx, fmt.Sprintf("blueprint %s not found", parentNode.Ref.Data().Blueprint.ID))
			return nil
		}

		logger.Errorf("Failure invoking action: failed to find blueprint: %v", err)
		return fmt.Errorf("failed to find blueprint: %w", err)
	}

	childNodeID := strings.Split(execution.NodeID, ":")[1]
	childNode, err := blueprint.FindNode(childNodeID)
	if err != nil {
		request.Fail(tx, fmt.Sprintf("node %s not found in blueprint", childNodeID))
		return nil
	}

	component, err := w.registry.GetComponent(childNode.Ref.Component.Name)
	if err != nil {
		logger.Errorf("Failure invoking action: failed to find component: %v", err)
		request.Fail(tx, fmt.Sprintf("component %s not found", childNode.Ref.Component.Name))
		return nil
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		logger.Errorf("Failure invoking action: spec is not specified")
		request.Fail(tx, "spec is not specified")
		return nil
	}

	actionName := spec.InvokeAction.ActionName
	actionDef := findAction(component.Actions(), actionName)
	if actionDef == nil {
		logger.Errorf("Failure invoking action: action '%s' not found for component '%s'", actionName, component.Name())
		request.Fail(tx, fmt.Sprintf("action '%s' not found for component '%s'", actionName, component.Name()))
		return nil
	}

	actionCtx := core.ActionContext{
		Name:           actionName,
		Configuration:  childNode.Configuration,
		Parameters:     spec.InvokeAction.Parameters,
		Logger:         logging.ForExecution(execution, parentExecution),
		HTTP:           w.registry.HTTPContext(),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Notifications:  contexts.NewNotificationContext(tx, uuid.Nil, execution.WorkflowID),
	}

	err = component.HandleAction(actionCtx)
	if err != nil {
		return w.handleRetryStrategy(tx, logger, request, err)
	}

	err = tx.Save(&execution).Error
	if err != nil {
		return fmt.Errorf("error saving execution after action handler: %v", err)
	}

	return request.Pass(tx)
}

func (w *NodeRequestWorker) log(format string, v ...any) {
	log.Printf("[NodeRequestWorker] "+format, v...)
}

func findAction(actions []core.Action, actionName string) *core.Action {
	for _, action := range actions {
		if action.Name == actionName {
			return &action
		}
	}

	return nil
}
