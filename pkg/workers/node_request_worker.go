package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
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
	gitProvider    gitprovider.Provider
	logger         *log.Entry
}

func NewNodeRequestWorker(encryptor crypto.Encryptor, registry *registry.Registry, gitProvider gitprovider.Provider, webhookBaseURL string, authService authorization.Authorization) *NodeRequestWorker {
	return &NodeRequestWorker{
		encryptor:      encryptor,
		registry:       registry,
		gitProvider:    gitProvider,
		webhookBaseURL: webhookBaseURL,
		semaphore:      semaphore.NewWeighted(25),
		authService:    authService,
		logger:         log.WithFields(log.Fields{"worker": "NodeRequestWorker"}),
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
				w.logger.Errorf("Error finding workflow nodes ready to be processed: %v", err)
			}

			telemetry.RecordNodeRequestWorkerRequestsCount(context.Background(), len(requests))

			for _, request := range requests {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(request models.CanvasNodeRequest) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessRequest(request); err != nil {
						w.logger.Errorf("Error processing request %s: %v", request.ID, err)
					}

					if request.ExecutionID != nil {
						if err := messages.PublishCanvasExecutionByID(request.WorkflowID, *request.ExecutionID); err != nil {
							w.logger.Errorf("Error publishing execution state: %v", err)
						}
					}
				}(request)
			}

			telemetry.RecordNodeRequestWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *NodeRequestWorker) LockAndProcessRequest(request models.CanvasNodeRequest) error {
	logger := w.logger.WithFields(log.Fields{"request": request.ID})

	logger.Infof("Locking and processing request")

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockNodeRequest(tx, request.ID)
		if err == nil {
			return w.processRequest(logger, tx, r, onNewEvents)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Infof("Request already processed - skipping")
			return nil
		}

		logger.Errorf("Error locking request: %v", err)
		return err
	})

	if err != nil {
		logger.Errorf("Error locking and processing request: %v", err)
		return err
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	return nil
}

func (w *NodeRequestWorker) processRequest(logger *log.Entry, tx *gorm.DB, request *models.CanvasNodeRequest, onNewEvents func([]models.CanvasEvent)) error {
	switch request.Type {
	case models.NodeRequestTypeInvokeAction:
		return w.invokeHook(logger, tx, request, onNewEvents)
	}

	return fmt.Errorf("unsupported node execution request type %s", request.Type)
}

func (w *NodeRequestWorker) invokeHook(logger *log.Entry, tx *gorm.DB, request *models.CanvasNodeRequest, onNewEvents func([]models.CanvasEvent)) error {
	if request.ExecutionID == nil {
		return w.invokeNodeHook(logger, tx, request, onNewEvents)
	}

	return w.invokeComponentHook(logger, tx, request, onNewEvents)
}

func (w *NodeRequestWorker) invokeNodeHook(logger *log.Entry, tx *gorm.DB, request *models.CanvasNodeRequest, onNewEvents func([]models.CanvasEvent)) error {
	node, err := models.FindUnscopedCanvasNode(tx, request.WorkflowID, request.NodeID)
	if err != nil {
		return fmt.Errorf("failed to find node: %w", err)
	}

	if node.DeletedAt.Valid {
		logger.Infof("Node %s deleted - completing request", request.NodeID)
		return request.Complete(tx)
	}

	switch node.Type {
	case models.NodeTypeTrigger:
		return w.invokeTriggerHook(logger, tx, request, node, onNewEvents)

	case models.NodeTypeComponent:
		return w.invokeNodeComponentHook(logger, tx, request, node, onNewEvents)
	}

	return fmt.Errorf("unsupported node type %s for node hook", node.Type)
}

func (w *NodeRequestWorker) invokeTriggerHook(logger *log.Entry, tx *gorm.DB, request *models.CanvasNodeRequest, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) error {
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

	resolvedConfiguration, err := contexts.NewNodeConfigurationBuilder(tx, node.WorkflowID).
		WithNodeID(node.NodeID).
		WithExpressionVariables(map[string]any{
			"parameters": spec.InvokeAction.Parameters,
		}).
		WithConfigurationFields(hookProvider.Configuration()).
		Build(contexts.WithoutRunTitleConfiguration(node.Configuration.Data()))
	if err != nil {
		return fmt.Errorf("failed to resolve trigger configuration: %w", err)
	}

	hookCtx := core.TriggerHookContext{
		Name:          spec.InvokeAction.ActionName,
		Parameters:    spec.InvokeAction.Parameters,
		Configuration: resolvedConfiguration,
		Logger:        logging.WithNode(logger, *node),
		HTTP:          w.registry.HTTPContextInTransaction(tx),
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
				logger.Infof("Integration %s not found - completing request", *node.AppInstallationID)
				return request.Complete(tx)
			}

			return fmt.Errorf("failed to find integration: %v", err)
		}

		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, w.encryptor, w.registry, onNewEvents)
	}

	logger.Infof("Invoking trigger hook")
	_, err = hookProvider.HandleHook(hookCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	logger.Infof("Trigger hook completed")
	err = tx.Save(&node).Error
	if err != nil {
		return fmt.Errorf("error saving node after action handler: %v", err)
	}

	logger.Infof("Request completed")
	return request.Complete(tx)
}

func (w *NodeRequestWorker) invokeNodeComponentHook(logger *log.Entry, tx *gorm.DB, request *models.CanvasNodeRequest, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) error {
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

	logger = logging.WithNode(logger, *node)
	hookCtx := core.ActionHookContext{
		Name:          spec.InvokeAction.ActionName,
		Configuration: node.Configuration.Data(),
		Parameters:    spec.InvokeAction.Parameters,
		Logger:        logger,
		HTTP:          w.registry.HTTPContextInTransaction(tx),
		Metadata:      contexts.NewNodeMetadataContext(tx, node),
		Requests:      contexts.NewNodeRequestContext(tx, node),
	}

	if node.AppInstallationID != nil {
		instance, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Infof("Integration %s not found - completing request", *node.AppInstallationID)
				return request.Complete(tx)
			}

			return fmt.Errorf("failed to find integration: %v", err)
		}

		logger = logging.WithIntegration(logger, *instance)
		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, w.encryptor, w.registry, onNewEvents)
		hookCtx.Logger = logger
	}

	logger.Infof("Invoking component hook")
	err = hookProvider.HandleHook(hookCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	logger.Infof("Component hook completed")
	err = tx.Save(&node).Error
	if err != nil {
		return fmt.Errorf("error saving node after action handler: %v", err)
	}

	logger.Infof("Request completed")
	return request.Complete(tx)
}

func (w *NodeRequestWorker) invokeComponentHook(logger *log.Entry, tx *gorm.DB, request *models.CanvasNodeRequest, onNewEvents func([]models.CanvasEvent)) error {
	if request.ExecutionID == nil {
		return fmt.Errorf("execution id is required for component hook")
	}

	execution, err := models.FindNodeExecutionInTransaction(tx, request.WorkflowID, *request.ExecutionID)
	if err != nil {
		return fmt.Errorf("execution %s not found: %w", request.ExecutionID, err)
	}

	if execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
		logger.Infof("Execution %s already finished or cancelling - completing request", execution.ID)
		return request.Complete(tx)
	}

	return w.invokeExecutionComponentHook(logger, tx, request, execution, onNewEvents)
}

func (w *NodeRequestWorker) invokeExecutionComponentHook(
	logger *log.Entry,
	tx *gorm.DB,
	request *models.CanvasNodeRequest,
	execution *models.CanvasNodeExecution,
	onNewEvents func([]models.CanvasEvent),
) error {
	node, err := models.FindUnscopedCanvasNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		return fmt.Errorf("failed to find node: %w", err)
	}

	//
	// If node was deleted, we cancel the execution before completing the request.
	//
	if node.DeletedAt.Valid {
		logger.Infof("Node %s deleted - requesting execution cancellation and completing request", execution.NodeID)
		if err := w.cancelExecutionForDeletedNode(logger, tx, execution); err != nil {
			return err
		}

		return request.Complete(tx)
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

	logger = logging.WithExecution(logger, execution)
	hookCtx := core.ActionHookContext{
		Name:           spec.InvokeAction.ActionName,
		Configuration:  execution.Configuration.Data(),
		Parameters:     spec.InvokeAction.Parameters,
		HTTP:           w.registry.HTTPContextInTransaction(tx),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution, onNewEvents),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Auth:           contexts.NewAuthReader(tx, workflow.OrganizationID, w.authService, nil),
		Secrets:        contexts.NewSecretsContext(tx, workflow.OrganizationID, w.encryptor),
		Files:          contexts.NewRepositoryFilesContextInTransaction(w.gitProvider, execution.WorkflowID, tx),
	}

	if node.AppInstallationID != nil {
		instance, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find integration: %v", err)
		}

		logger = logging.WithIntegration(logger, *instance)
		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, w.encryptor, w.registry, onNewEvents)
	}

	logger.Infof("Invoking component execution hook")

	hookCtx.Logger = logger
	err = hookProvider.HandleHook(hookCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	logger.Infof("Component execution hook completed")
	finished, err := execution.IsFinished(tx)
	if err != nil {
		return err
	}

	if finished {
		logger.Infof("Execution %s already finished after hook - completing request", execution.ID)
		return request.Complete(tx)
	}

	err = tx.Save(&execution).Error
	if err != nil {
		return fmt.Errorf("error saving execution after action handler: %v", err)
	}

	logger.Infof("Request completed")
	return request.Complete(tx)
}

func (w *NodeRequestWorker) cancelExecutionForDeletedNode(logger *log.Entry, tx *gorm.DB, execution *models.CanvasNodeExecution) error {
	if execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
		return nil
	}

	logger.Infof("Requesting cancellation for execution %s", execution.ID)
	return execution.RequestCancellation(tx, nil)
}
