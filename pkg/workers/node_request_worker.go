package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/authorization"
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
	semaphore      *semaphore.Weighted
	registry       *registry.Registry
	encryptor      crypto.Encryptor
	webhookBaseURL string
	authService    authorization.Authorization
}

func NewNodeRequestWorker(encryptor crypto.Encryptor, registry *registry.Registry, webhookBaseURL string, authService authorization.Authorization) *NodeRequestWorker {
	return &NodeRequestWorker{
		encryptor:      encryptor,
		registry:       registry,
		webhookBaseURL: webhookBaseURL,
		semaphore:      semaphore.NewWeighted(25),
		authService:    authService,
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
	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockNodeRequest(tx, request.ID)
		if err != nil {
			w.log("Request %s already being processed - skipping", request.ID)
			return nil
		}

		return w.processRequest(tx, r, onNewEvents)
	})

	if err != nil {
		return err
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	return nil
}

func (w *NodeRequestWorker) processRequest(tx *gorm.DB, request *models.CanvasNodeRequest, onNewEvents func([]models.CanvasEvent)) error {
	switch request.Type {
	case models.NodeRequestTypeInvokeAction:
		return w.invokeHook(tx, request, onNewEvents)
	}

	return fmt.Errorf("unsupported node execution request type %s", request.Type)
}

func (w *NodeRequestWorker) invokeHook(tx *gorm.DB, request *models.CanvasNodeRequest, onNewEvents func([]models.CanvasEvent)) error {
	if request.ExecutionID == nil {
		return w.invokeNodeHook(tx, request, onNewEvents)
	}

	return w.invokeComponentHook(tx, request, onNewEvents)
}

func (w *NodeRequestWorker) invokeNodeHook(tx *gorm.DB, request *models.CanvasNodeRequest, onNewEvents func([]models.CanvasEvent)) error {
	node, err := models.FindCanvasNode(tx, request.WorkflowID, request.NodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	switch node.Type {
	case models.NodeTypeTrigger:
		return w.invokeTriggerHook(tx, request, node, onNewEvents)

	case models.NodeTypeComponent:
		return w.invokeNodeComponentHook(tx, request, node, onNewEvents)
	}

	return fmt.Errorf("unsupported node type %s for node hook", node.Type)
}

func (w *NodeRequestWorker) invokeTriggerHook(tx *gorm.DB, request *models.CanvasNodeRequest, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) error {
	nodeRef := node.Ref.Data()
	if nodeRef.Trigger == nil {
		return fmt.Errorf("node %s is not a trigger", node.NodeID)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	hookProvider, _, err := w.registry.FindTriggerHook(nodeRef.Trigger.Name, spec.InvokeAction.ActionName)
	if err != nil {
		return fmt.Errorf("failed to find hook: %v", err)
	}

	hookCtx := core.TriggerHookContext{
		Name:          spec.InvokeAction.ActionName,
		Parameters:    spec.InvokeAction.Parameters,
		Configuration: node.Configuration.Data(),
		Logger:        logging.ForNode(*node),
		HTTP:          w.registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataContext(tx, node),
		Events:        contexts.NewEventContext(tx, node, onNewEvents),
		Requests:      contexts.NewNodeRequestContext(tx, node),
	}

	if node.WebhookID != nil {
		hookCtx.Webhook = contexts.NewNodeWebhookContext(context.Background(), tx, w.encryptor, node, w.webhookBaseURL)
	}

	if node.AppInstallationID != nil {
		instance, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.log("integration %s not found - completing request", *node.AppInstallationID)
				return request.Complete(tx)
			}

			return fmt.Errorf("failed to find integration: %v", err)
		}

		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, w.encryptor, w.registry, onNewEvents)
	}

	_, err = hookProvider.HandleHook(hookCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	err = tx.Save(&node).Error
	if err != nil {
		return fmt.Errorf("error saving node after action handler: %v", err)
	}

	return request.Complete(tx)
}

func (w *NodeRequestWorker) invokeNodeComponentHook(tx *gorm.DB, request *models.CanvasNodeRequest, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) error {
	nodeRef := node.Ref.Data()
	if nodeRef.Component == nil {
		return fmt.Errorf("node %s is not a component", node.NodeID)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	hookProvider, _, err := w.registry.FindActionHook(nodeRef.Component.Name, spec.InvokeAction.ActionName)
	if err != nil {
		return fmt.Errorf("failed to find hook: %v", err)
	}

	logger := logging.ForNode(*node)
	hookCtx := core.ActionHookContext{
		Name:          spec.InvokeAction.ActionName,
		Configuration: node.Configuration.Data(),
		Parameters:    spec.InvokeAction.Parameters,
		Logger:        logger,
		HTTP:          w.registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataContext(tx, node),
		Requests:      contexts.NewNodeRequestContext(tx, node),
	}

	if node.AppInstallationID != nil {
		instance, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.log("integration %s not found - completing request", *node.AppInstallationID)
				return request.Complete(tx)
			}

			return fmt.Errorf("failed to find integration: %v", err)
		}

		logger = logging.WithIntegration(logger, *instance)
		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, w.encryptor, w.registry, onNewEvents)
		hookCtx.Logger = logger
	}

	err = hookProvider.HandleHook(hookCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	err = tx.Save(&node).Error
	if err != nil {
		return fmt.Errorf("error saving node after action handler: %v", err)
	}

	return request.Complete(tx)
}

func (w *NodeRequestWorker) invokeComponentHook(tx *gorm.DB, request *models.CanvasNodeRequest, onNewEvents func([]models.CanvasEvent)) error {
	execution, err := models.FindNodeExecutionInTransaction(tx, request.WorkflowID, *request.ExecutionID)
	if err != nil {
		return fmt.Errorf("execution %s not found: %w", request.ExecutionID, err)
	}

	if execution.ParentExecutionID == nil {
		return w.invokeParentNodeComponentAction(tx, request, execution, onNewEvents)
	}

	return w.invokeChildNodeComponentAction(tx, request, execution, onNewEvents)
}

func (w *NodeRequestWorker) invokeParentNodeComponentAction(
	tx *gorm.DB,
	request *models.CanvasNodeRequest,
	execution *models.CanvasNodeExecution,
	onNewEvents func([]models.CanvasEvent),
) error {
	node, err := models.FindCanvasNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	hookProvider, _, err := w.registry.FindActionHook(node.Ref.Data().Component.Name, spec.InvokeAction.ActionName)
	if err != nil {
		return fmt.Errorf("component not found: %w", err)
	}

	workflow, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	logger := logging.ForExecution(execution, nil)
	hookCtx := core.ActionHookContext{
		Name:           spec.InvokeAction.ActionName,
		Configuration:  node.Configuration.Data(),
		Parameters:     spec.InvokeAction.Parameters,
		HTTP:           w.registry.HTTPContext(),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, hookProvider, execution, onNewEvents),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Notifications:  contexts.NewNotificationContext(tx, uuid.Nil, node.WorkflowID),
		Auth:           contexts.NewAuthReader(tx, workflow.OrganizationID, w.authService, nil),
		Secrets:        contexts.NewSecretsContext(tx, workflow.OrganizationID, w.encryptor),
	}

	if node.AppInstallationID != nil {
		instance, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find integration: %v", err)
		}

		logger = logging.WithIntegration(logger, *instance)
		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, w.encryptor, w.registry, onNewEvents)
	}

	hookCtx.Logger = logger
	err = hookProvider.HandleHook(hookCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	err = tx.Save(&execution).Error
	if err != nil {
		return fmt.Errorf("error saving execution after action handler: %v", err)
	}

	return request.Complete(tx)
}

func (w *NodeRequestWorker) invokeChildNodeComponentAction(
	tx *gorm.DB,
	request *models.CanvasNodeRequest,
	execution *models.CanvasNodeExecution,
	onNewEvents func([]models.CanvasEvent),
) error {
	parentExecution, err := models.FindNodeExecutionInTransaction(tx, execution.WorkflowID, *execution.ParentExecutionID)
	if err != nil {
		return fmt.Errorf("parent execution %s not found: %w", execution.ParentExecutionID, err)
	}

	parentNode, err := models.FindCanvasNode(tx, execution.WorkflowID, parentExecution.NodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	blueprint, err := models.FindUnscopedBlueprintInTransaction(tx, parentNode.Ref.Data().Blueprint.ID)
	if err != nil {
		return fmt.Errorf("blueprint not found: %w", err)
	}

	childNodeID := strings.Split(execution.NodeID, ":")[1]
	childNode, err := blueprint.FindNode(childNodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	hookProvider, _, err := w.registry.FindActionHook(childNode.Ref.Component.Name, spec.InvokeAction.ActionName)
	if err != nil {
		return fmt.Errorf("component not found: %w", err)
	}

	workflow, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	hookCtx := core.ActionHookContext{
		Name:           spec.InvokeAction.ActionName,
		Configuration:  execution.Configuration.Data(),
		Parameters:     spec.InvokeAction.Parameters,
		Logger:         logging.ForExecution(execution, parentExecution),
		HTTP:           w.registry.HTTPContext(),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, hookProvider, execution, onNewEvents),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Notifications:  contexts.NewNotificationContext(tx, uuid.Nil, execution.WorkflowID),
		Auth:           contexts.NewAuthReader(tx, workflow.OrganizationID, nil, nil),
		Secrets:        contexts.NewSecretsContext(tx, workflow.OrganizationID, w.encryptor),
	}

	err = hookProvider.HandleHook(hookCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	err = tx.Save(&execution).Error
	if err != nil {
		return fmt.Errorf("error saving execution after action handler: %v", err)
	}

	return request.Complete(tx)
}

func (w *NodeRequestWorker) log(format string, v ...any) {
	log.Printf("[NodeRequestWorker] "+format, v...)
}
