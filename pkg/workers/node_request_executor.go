package workers

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type NodeRequestExecutor struct {
	registry       *registry.Registry
	encryptor      crypto.Encryptor
	webhookBaseURL string
	authService    authorization.Authorization
	logger         *log.Entry
}

func NewNodeRequestExecutor(encryptor crypto.Encryptor, registry *registry.Registry, webhookBaseURL string, authService authorization.Authorization) *NodeRequestExecutor {
	return &NodeRequestExecutor{
		encryptor:      encryptor,
		registry:       registry,
		webhookBaseURL: webhookBaseURL,
		authService:    authService,
		logger:         log.WithFields(log.Fields{"executor": "NodeRequestExecutor"}),
	}
}

func (e *NodeRequestExecutor) factoryIntakeDeps() factoryactions.IntakeDependencies {
	return factoryactions.IntakeDependencies{
		Registry:       e.registry,
		Encryptor:      e.encryptor,
		AuthService:    e.authService,
		WebhookBaseURL: e.webhookBaseURL,
	}
}

// RunNow processes a request synchronously, even when its scheduled time is in the future.
// It shares the worker lock, so an in-flight or completed request is skipped.
func (e *NodeRequestExecutor) RunNow(request models.CanvasNodeRequest) error {
	return e.execute(request, models.LockNodeRequestNow)
}

// Execute processes a pending request when its scheduled time has arrived.
func (e *NodeRequestExecutor) Execute(request models.CanvasNodeRequest) error {
	return e.execute(request, models.LockNodeRequest)
}

func (e *NodeRequestExecutor) execute(
	request models.CanvasNodeRequest,
	lockRequest func(*gorm.DB, uuid.UUID) (*models.CanvasNodeRequest, error),
) error {
	defer e.publishExecution(request)

	if request.Type == models.NodeRequestTypePullRequestChecksStart {
		events, err := factoryactions.ProcessPullRequestChecksStartRequest(context.Background(), e.factoryIntakeDeps(), request.ID)
		if err != nil {
			return err
		}
		for i := range events {
			if err := messages.PublishCanvasEventCreatedMessage(&events[i]); err != nil {
				e.logger.WithError(err).Warnf("checks: failed to publish event %s", events[i].ID)
			}
		}
		return nil
	}
	if request.Type == models.NodeRequestTypeVisualEvidenceCapture {
		return e.processVisualEvidenceCapture(request)
	}

	logger := e.logger.WithFields(log.Fields{"request": request.ID})

	logger.Infof("Locking and processing request")

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	type pendingFactoryWorkOrderUpdate struct {
		factoryID string
		orderID   string
		reason    string
	}
	pendingFactoryWorkOrderUpdates := []pendingFactoryWorkOrderUpdate{}
	onFactoryWorkOrderUpdated := func(factoryID, orderID, reason string) {
		pendingFactoryWorkOrderUpdates = append(pendingFactoryWorkOrderUpdates, pendingFactoryWorkOrderUpdate{
			factoryID: factoryID,
			orderID:   orderID,
			reason:    reason,
		})
	}
	type pendingGitHubPullRequest struct {
		organizationID uuid.UUID
		factoryID      uuid.UUID
		pullRequestID  uuid.UUID
	}
	pendingGitHubPullRequests := []pendingGitHubPullRequest{}
	onGitHubPullRequestRecorded := func(organizationID, factoryID, pullRequestID uuid.UUID) {
		pendingGitHubPullRequests = append(pendingGitHubPullRequests, pendingGitHubPullRequest{
			organizationID: organizationID,
			factoryID:      factoryID,
			pullRequestID:  pullRequestID,
		})
	}
	pendingFileBindCleanups := []contexts.FileBindCleanup{}
	onFileBindCleanup := func(job contexts.FileBindCleanup) {
		pendingFileBindCleanups = append(pendingFileBindCleanups, job)
	}
	pendingProductAnalytics := []productanalytics.Event{}
	onProductAnalytics := func(event productanalytics.Event) {
		pendingProductAnalytics = append(pendingProductAnalytics, event)
	}

	runCancellations := &RunCancellationNotifier{}
	runnerChanges := &RunnerTaskChangeNotifier{}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := lockRequest(tx, request.ID)
		if err == nil {
			return e.processRequest(logger, tx, r, onNewEvents, runCancellations, onFactoryWorkOrderUpdated, onGitHubPullRequestRecorded, onFileBindCleanup, onProductAnalytics, runnerChanges)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Infof("Request already processed - skipping")
			return nil
		}

		logger.Errorf("Error locking request: %v", err)
		return err
	})

	if err != nil {
		contexts.ApplyFileBindCleanups(pendingFileBindCleanups, err)
		logger.Errorf("Error locking and processing request: %v", err)
		return err
	}

	contexts.ApplyFileBindCleanups(pendingFileBindCleanups, nil)
	for _, event := range pendingProductAnalytics {
		productanalytics.Capture(event)
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	for _, update := range pendingFactoryWorkOrderUpdates {
		if err := messages.PublishFactoryWorkOrderUpdated(update.factoryID, update.orderID, update.reason); err != nil {
			logger.Errorf("failed to publish factory work order updated RabbitMQ message: %v", err)
		}
	}

	for _, recorded := range pendingGitHubPullRequests {
		factoryactions.ScheduleFactoryPullRequestMergeabilityRefresh(
			context.Background(),
			e.factoryIntakeDeps(),
			recorded.organizationID,
			recorded.factoryID,
			recorded.pullRequestID,
		)
		factoryactions.StartPullRequestChecks(context.Background(), e.factoryIntakeDeps(), recorded.pullRequestID)
		factoryactions.StartVisualEvidenceCapture(
			context.Background(),
			e.factoryIntakeDeps(),
			recorded.organizationID,
			recorded.factoryID,
			recorded.pullRequestID,
		)
	}

	runCancellations.Publish()
	runnerChanges.Publish()

	return nil
}

func (e *NodeRequestExecutor) processVisualEvidenceCapture(request models.CanvasNodeRequest) error {
	events, err := factoryactions.ProcessVisualEvidenceCaptureRequest(
		context.Background(),
		e.factoryIntakeDeps(),
		request.ID,
	)
	if err != nil {
		return err
	}
	for i := range events {
		if err := messages.PublishCanvasEventCreatedMessage(&events[i]); err != nil {
			e.logger.WithError(err).Warnf("visual evidence: failed to publish event %s", events[i].ID)
		}
	}
	return nil
}

func (e *NodeRequestExecutor) processRequest(
	logger *log.Entry,
	tx *gorm.DB,
	request *models.CanvasNodeRequest,
	onNewEvents func([]models.CanvasEvent),
	runCancellations *RunCancellationNotifier,
	onFactoryWorkOrderUpdated func(factoryID, orderID, reason string),
	onGitHubPullRequestRecorded func(organizationID, factoryID, pullRequestID uuid.UUID),
	onFileBindCleanup func(contexts.FileBindCleanup),
	onProductAnalytics func(productanalytics.Event),
	runnerChanges *RunnerTaskChangeNotifier,
) error {
	switch request.Type {
	case models.NodeRequestTypeInvokeAction:
		return e.invokeHook(logger, tx, request, onNewEvents, runCancellations, onFactoryWorkOrderUpdated, onGitHubPullRequestRecorded, onFileBindCleanup, onProductAnalytics, runnerChanges)
	}

	return fmt.Errorf("unsupported node execution request type %s", request.Type)
}

func (e *NodeRequestExecutor) invokeHook(
	logger *log.Entry,
	tx *gorm.DB,
	request *models.CanvasNodeRequest,
	onNewEvents func([]models.CanvasEvent),
	runCancellations *RunCancellationNotifier,
	onFactoryWorkOrderUpdated func(factoryID, orderID, reason string),
	onGitHubPullRequestRecorded func(organizationID, factoryID, pullRequestID uuid.UUID),
	onFileBindCleanup func(contexts.FileBindCleanup),
	onProductAnalytics func(productanalytics.Event),
	runnerChanges *RunnerTaskChangeNotifier,
) error {
	if request.ExecutionID == nil {
		return e.invokeNodeHook(logger, tx, request, onNewEvents)
	}

	return e.invokeComponentHook(logger, tx, request, onNewEvents, runCancellations, onFactoryWorkOrderUpdated, onGitHubPullRequestRecorded, onFileBindCleanup, onProductAnalytics, runnerChanges)
}

func (e *NodeRequestExecutor) invokeNodeHook(logger *log.Entry, tx *gorm.DB, request *models.CanvasNodeRequest, onNewEvents func([]models.CanvasEvent)) error {
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
		return e.invokeTriggerHook(logger, tx, request, node, onNewEvents)

	case models.NodeTypeComponent:
		return e.invokeNodeComponentHook(logger, tx, request, node, onNewEvents)
	}

	return fmt.Errorf("unsupported node type %s for node hook", node.Type)
}

func (e *NodeRequestExecutor) invokeTriggerHook(logger *log.Entry, tx *gorm.DB, request *models.CanvasNodeRequest, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) error {
	nodeRef := node.Ref.Data()
	if nodeRef.Trigger == nil {
		return fmt.Errorf("node %s is not a trigger", node.NodeID)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	hookProvider, _, err := e.registry.FindTriggerHook(nodeRef.Trigger.Name, spec.InvokeAction.ActionName)
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
		HTTP:          e.registry.HTTPContextInTransaction(tx),
		Metadata:      contexts.NewNodeMetadataContext(tx, node),
		Events:        contexts.NewEventContext(tx, node, nil, onNewEvents),
		Requests:      contexts.NewNodeRequestContext(tx, node),
	}

	if node.WebhookID != nil {
		hookCtx.Webhook = contexts.NewNodeWebhookContext(context.Background(), tx, e.encryptor, node, e.webhookBaseURL)
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

		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, e.encryptor, e.registry, onNewEvents)
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

func (e *NodeRequestExecutor) invokeNodeComponentHook(logger *log.Entry, tx *gorm.DB, request *models.CanvasNodeRequest, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) error {
	nodeRef := node.Ref.Data()
	if nodeRef.Component == nil {
		return fmt.Errorf("node %s is not a component", node.NodeID)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	componentName := nodeRef.Component.Name
	configuration := node.Configuration.Data()
	if request.ExecutionID != nil {
		execution, err := models.FindNodeExecutionInTransaction(tx, request.WorkflowID, *request.ExecutionID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if execution != nil {
			if frozen := execution.FrozenComponentName(); frozen != "" {
				componentName = frozen
				configuration = execution.Configuration.Data()
			}
		}
	}

	hookProvider, _, err := e.registry.FindActionHook(componentName, spec.InvokeAction.ActionName)
	if err != nil {
		return fmt.Errorf("failed to find hook: %v", err)
	}

	logger = logging.WithNode(logger, *node)
	hookCtx := core.ActionHookContext{
		Name:          spec.InvokeAction.ActionName,
		Configuration: configuration,
		Parameters:    spec.InvokeAction.Parameters,
		Logger:        logger,
		HTTP:          e.registry.HTTPContextInTransaction(tx),
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
		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, e.encryptor, e.registry, onNewEvents)
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

func (e *NodeRequestExecutor) invokeComponentHook(
	logger *log.Entry,
	tx *gorm.DB,
	request *models.CanvasNodeRequest,
	onNewEvents func([]models.CanvasEvent),
	runCancellations *RunCancellationNotifier,
	onFactoryWorkOrderUpdated func(factoryID, orderID, reason string),
	onGitHubPullRequestRecorded func(organizationID, factoryID, pullRequestID uuid.UUID),
	onFileBindCleanup func(contexts.FileBindCleanup),
	onProductAnalytics func(productanalytics.Event),
	runnerChanges *RunnerTaskChangeNotifier,
) error {
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

	return e.invokeExecutionComponentHook(logger, tx, request, execution, onNewEvents, runCancellations, onFactoryWorkOrderUpdated, onGitHubPullRequestRecorded, onFileBindCleanup, onProductAnalytics, runnerChanges)
}

func (e *NodeRequestExecutor) invokeExecutionComponentHook(
	logger *log.Entry,
	tx *gorm.DB,
	request *models.CanvasNodeRequest,
	execution *models.CanvasNodeExecution,
	onNewEvents func([]models.CanvasEvent),
	runCancellations *RunCancellationNotifier,
	onFactoryWorkOrderUpdated func(factoryID, orderID, reason string),
	onGitHubPullRequestRecorded func(organizationID, factoryID, pullRequestID uuid.UUID),
	onFileBindCleanup func(contexts.FileBindCleanup),
	onProductAnalytics func(productanalytics.Event),
	runnerChanges *RunnerTaskChangeNotifier,
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
		if err := e.cancelExecutionForDeletedNode(logger, tx, execution); err != nil {
			return err
		}

		return request.Complete(tx)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	componentName := ""
	if component := node.Ref.Data().Component; component != nil {
		componentName = component.Name
	}
	if frozen := execution.FrozenComponentName(); frozen != "" {
		componentName = frozen
	}
	hookProvider, _, err := e.registry.FindActionHook(componentName, spec.InvokeAction.ActionName)
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
		HTTP:           e.registry.HTTPContextInTransaction(tx),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution, onNewEvents),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Auth:           contexts.NewAuthReader(tx, workflow.OrganizationID, e.authService, nil),
		Secrets:        contexts.NewSecretsContext(tx, e.registry, workflow.OrganizationID, e.encryptor),
		Runs:           runCancellations.Bind(contexts.NewRunExecutionContext(tx, workflow, node, execution)),
		Factory: contexts.NewFactoryContext(tx, workflow, execution).
			WithWorkOrderUpdated(onFactoryWorkOrderUpdated).
			WithRunsCancelled(runCancellations.RecordRuns).
			WithGitHubPullRequestRecorded(onGitHubPullRequestRecorded).
			WithFileBindCleanup(onFileBindCleanup).
			WithProductAnalytics(onProductAnalytics).
			WithRemoteImageIngest(e.encryptor, e.registry),
		Usage:     contexts.NewUsageContext(workflow.OrganizationID, execution),
		HostedLLM: contexts.NewHostedLLMContext(tx, e.encryptor, workflow.OrganizationID, workflow.FactoryID),
		RunnerTasks: runnerChanges.Bind(contexts.NewRunnerTaskContext(
			tx,
			e.encryptor,
			workflow.OrganizationID,
		)),
	}

	hookCtx.RunnerTasks.(*contexts.RunnerTaskContext).
		WithRunScope(execution.RunID).
		WithSecrets(hookCtx.Secrets.(*contexts.SecretsContext))
	if node.AppInstallationID != nil {
		instance, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find integration: %v", err)
		}

		logger = logging.WithIntegration(logger, *instance)
		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, instance, e.encryptor, e.registry, onNewEvents)
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

func (e *NodeRequestExecutor) cancelExecutionForDeletedNode(logger *log.Entry, tx *gorm.DB, execution *models.CanvasNodeExecution) error {
	if execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
		return nil
	}

	logger.Infof("Requesting cancellation for execution %s", execution.ID)
	return execution.RequestCancellation(tx, nil)
}

func (e *NodeRequestExecutor) publishExecution(request models.CanvasNodeRequest) {
	if request.ExecutionID == nil {
		return
	}
	if err := messages.PublishCanvasExecutionByID(request.WorkflowID, *request.ExecutionID); err != nil {
		e.logger.WithError(err).Error("Error publishing execution state")
	}
}
