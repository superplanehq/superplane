package workers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/renderedtext/go-tackle"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

var ErrRecordLocked = errors.New("record locked")

type NodeExecutor struct {
	encryptor      crypto.Encryptor
	registry       *registry.Registry
	gitProvider    gitprovider.Provider
	authService    authorization.Authorization
	baseURL        string
	webhookBaseURL string
	semaphore      *semaphore.Weighted
	logger         *logrus.Entry

	rabbitMQURL string
	consumer    *tackle.Consumer
}

func NewNodeExecutor(encryptor crypto.Encryptor, registry *registry.Registry, gitProvider gitprovider.Provider, baseURL string, webhookBaseURL string, rabbitMQURL string, authService authorization.Authorization) *NodeExecutor {
	return &NodeExecutor{
		encryptor:      encryptor,
		registry:       registry,
		gitProvider:    gitProvider,
		baseURL:        baseURL,
		webhookBaseURL: webhookBaseURL,
		semaphore:      semaphore.NewWeighted(25),
		logger:         logrus.WithFields(logrus.Fields{"worker": "NodeExecutor"}),
		rabbitMQURL:    rabbitMQURL,
		authService:    authService,
	}
}

func (w *NodeExecutor) Name() string {
	return "NodeExecutor"
}

func (w *NodeExecutor) Start(ctx context.Context) {
	go w.StartRabbitMQConsumer(ctx)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()

			executions, err := models.ListPendingNodeExecutions()
			if err != nil {
				w.logger.Errorf("Error finding workflow nodes ready to be processed: %v", err)
			}

			telemetry.RecordExecutorWorkerNodesCount(context.Background(), len(executions))

			for _, execution := range executions {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(execution models.CanvasNodeExecution) {
					defer w.semaphore.Release(1)

					err := w.LockAndProcessNodeExecution(execution.ID)
					if err == nil {
						if publishErr := messages.PublishCanvasExecutionByID(execution.WorkflowID, execution.ID); publishErr != nil {
							w.logger.Errorf("Error publishing execution state: %v", publishErr)
						}
						return
					}

					if err == ErrRecordLocked {
						return
					}

					w.logger.Errorf("Error processing node execution - node=%s, execution=%s: %v", execution.NodeID, execution.ID, err)
				}(execution)
			}

			telemetry.RecordExecutorWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *NodeExecutor) StartRabbitMQConsumer(ctx context.Context) {
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name(),
		RemoteExchange: messages.ExecutionsExchange,
		Service:        messages.ExecutionsExchange + "." + messages.ExecutionPendingRoutingKey + "." + w.Name(),
		RoutingKey:     messages.ExecutionPendingRoutingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))
	w.consumer = consumer

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.ExecutionPendingRoutingKey)

		err := w.consumer.Start(&options, w.Consume)
		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", messages.ExecutionPendingRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.ExecutionPendingRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *NodeExecutor) Consume(delivery tackle.Delivery) error {
	data := &pb.CanvasNodeExecutionMessage{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		w.logger.Errorf("Error unmarshaling canvas execution message: %v", err)
		return err
	}

	executionID, err := uuid.Parse(data.Id)
	if err != nil {
		w.logger.Errorf("Error parsing execution id: %v", err)
		return err
	}

	err = w.LockAndProcessNodeExecution(executionID)
	if err == nil {
		workflowID, parseErr := uuid.Parse(data.CanvasId)
		if parseErr != nil {
			w.logger.Errorf("Error parsing canvas id: %v", parseErr)
			return parseErr
		}

		if publishErr := messages.PublishCanvasExecutionByID(workflowID, executionID); publishErr != nil {
			w.logger.Errorf("Error publishing execution state: %v", publishErr)
			return publishErr
		}

		return nil
	}

	if err == ErrRecordLocked {
		return nil
	}

	w.logger.Errorf("Error processing node execution - execution=%s: %v", executionID, err)
	return err
}

func (w *NodeExecutor) LockAndProcessNodeExecution(id uuid.UUID) error {
	//
	// For every execution we process, we track the following metrics:
	// - outcome: success, failed, skipped
	// - reason: none, locked, deadlock, not_found, action_error, internal
	// - component: the component name of the node
	//
	start := time.Now()
	metricOutcome := executorOutcomeSuccess
	metricReason := executorReasonNone
	metricComponent := "unknown"
	defer func() {
		telemetry.RecordExecutorWorkerExecution(
			context.Background(),
			time.Since(start),
			metricOutcome,
			metricReason,
			metricComponent,
		)
	}()

	//
	// We track the events produced by the component execution,
	// so we can publish RabbitMQ messages for them.
	//
	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	//
	// We also track whether memory was modified during the execution so we can
	// broadcast a memory_updated event after the transaction commits.
	//
	var memoryChangedCanvasID uuid.UUID
	onMemoryChanged := func(canvasID uuid.UUID) {
		memoryChangedCanvasID = canvasID
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		//
		// Try to lock the execution record for update.
		// If we can't, it means another worker is already processing it.
		//
		// We also ensure that the execution is still in pending state,
		// to avoid processing already started or finished executions.
		//
		// Why we need to check the state again:
		//
		// Even though we fetch pending executions in the main loop,
		// there is a race condition where multiple workers might pick the same execution
		// before any of them has a chance to lock it.
		//
		// By checking the state again here, we ensure that only one worker
		// can start processing a given execution.
		//
		// Note: We use SKIP LOCKED to avoid waiting on locked records.
		//

		execution, err := models.LockPendingNodeExecutionInActiveCanvas(tx, id)
		if err != nil {
			w.logger.Debugf("Execution %s already being processed - skipping", id.String())
			metricOutcome = executorOutcomeSkipped
			metricReason = executorReasonLocked
			return ErrRecordLocked
		}

		node, err := models.FindCanvasNode(tx, execution.WorkflowID, execution.NodeID)
		if err != nil {
			metricOutcome = executorOutcomeFailed
			metricReason = classifyAttemptFailure(err, nil)
			return err
		}

		metricComponent = node.ComponentName()
		processErr := w.executeActionNode(tx, execution, node, onNewEvents, onMemoryChanged)
		if processErr != nil {
			metricOutcome = executorOutcomeFailed
			metricReason = classifyAttemptFailure(processErr, execution)
			return processErr
		}

		if execution.Result == models.CanvasNodeExecutionResultFailed {
			metricOutcome = executorOutcomeFailed
			metricReason = classifyAttemptFailure(nil, execution)
		}

		return nil
	})

	if err != nil {
		return err
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	if memoryChangedCanvasID != uuid.Nil {
		if err := messages.NewCanvasMemoryUpdatedMessage(memoryChangedCanvasID.String()).PublishMemoryUpdated(); err != nil {
			w.logger.Errorf("failed to publish canvas memory updated RabbitMQ message: %v", err)
		}
	}

	return nil
}

func (w *NodeExecutor) executeActionNode(tx *gorm.DB, execution *models.CanvasNodeExecution, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent), onMemoryChanged func(uuid.UUID)) error {
	logger := logging.WithExecution(
		logging.WithNode(w.logger, *node),
		execution,
	)

	err := execution.StartInTransaction(tx)
	if err != nil {
		logger.Errorf("failed to start execution: %v", err)
		return fmt.Errorf("failed to start execution: %w", err)
	}

	ref := node.Ref.Data()
	action, err := w.registry.GetAction(ref.Component.Name)
	if err != nil {
		logger.Errorf("action %s not found: %v", ref.Component.Name, err)
		return fmt.Errorf("action %s not found: %w", ref.Component.Name, err)
	}

	inputEvent, err := models.FindCanvasEventInTransaction(tx, execution.EventID)
	if err != nil {
		logger.Errorf("failed to find input event: %v", err)
		return fmt.Errorf("failed to find input event: %w", err)
	}

	input := inputEvent.Data.Data()

	workflow, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, node.WorkflowID)
	if err != nil {
		logger.Errorf("failed to find workflow: %v", err)
		return fmt.Errorf("failed to find workflow: %v", err)
	}

	builder := contexts.NewNodeConfigurationBuilder(tx, execution.WorkflowID).
		WithNodeID(node.NodeID).
		WithRootEvent(&execution.RootEventID).
		WithIncomingEventID(&execution.EventID).
		WithInput(map[string]any{inputEvent.NodeID: input})
	if execution.PreviousExecutionID != nil {
		builder = builder.WithPreviousExecution(execution.PreviousExecutionID)
	}

	ctx := core.ExecutionContext{
		ID:             execution.ID,
		WorkflowID:     execution.WorkflowID.String(),
		OrganizationID: workflow.OrganizationID.String(),
		CanvasName:     workflow.Name,
		NodeID:         execution.NodeID,
		NodeName:       node.Name,
		SourceNodeID:   inputEvent.NodeID,
		BaseURL:        w.baseURL,
		Configuration:  execution.Configuration.Data(),
		Data:           input,
		HTTP:           w.registry.HTTPContextInTransaction(tx),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		NodeMetadata:   contexts.NewNodeMetadataContext(tx, node),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution, onNewEvents),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Auth:           contexts.NewAuthReader(tx, workflow.OrganizationID, w.authService, nil),
		Secrets:        contexts.NewSecretsContext(tx, workflow.OrganizationID, w.encryptor),
		CanvasMemory: contexts.NewCanvasMemoryContext(tx, execution.WorkflowID).
			WithChangeCallback(func() { onMemoryChanged(execution.WorkflowID) }),
		Files:       contexts.NewRepositoryFilesContext(w.gitProvider, execution.WorkflowID),
		Webhook:     contexts.NewNodeWebhookContext(context.Background(), tx, w.encryptor, node, w.webhookBaseURL),
		Expressions: contexts.NewExpressionContext(builder),
	}

	if node.AppInstallationID != nil {
		instance, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Errorf("integration %s not found", *node.AppInstallationID)
				return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, "integration not found")
			}

			logger.Errorf("failed to find integration: %v", err)
			return fmt.Errorf("failed to find integration: %v", err)
		}

		logger = logging.WithIntegration(logger, *instance)
		ctx.Integration = contexts.NewIntegrationContext(tx, node, instance, w.encryptor, w.registry, onNewEvents)
	}

	ctx.Logger = logger
	if err := action.Execute(ctx); err != nil {
		logger.Errorf("failed to execute action: %v", err)
		return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, err.Error())
	}

	logger.Info("Action executed successfully")

	return tx.Save(execution).Error
}

const (
	executorOutcomeSuccess = "success"
	executorOutcomeFailed  = "failed"
	executorOutcomeSkipped = "skipped"

	executorReasonNone     = "none"
	executorReasonLocked   = "locked"
	executorReasonDeadlock = "deadlock"
	executorReasonNotFound = "not_found"
	executorReasonInternal = "internal"
)

func classifyProcessError(err error) string {
	if err == nil {
		return executorReasonNone
	}

	if errors.Is(err, ErrRecordLocked) {
		return executorReasonLocked
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return executorReasonNotFound
	}

	if isDeadlockError(err) {
		return executorReasonDeadlock
	}

	return executorReasonInternal
}

func classifyAttemptFailure(err error, execution *models.CanvasNodeExecution) string {
	if err == nil {
		if execution == nil {
			return executorReasonNone
		}
		return classifyExecutionFailure(execution)
	}

	if reason := classifyProcessError(err); reason != executorReasonInternal {
		return reason
	}

	if execution != nil {
		if isDeadlockMessage(execution.ResultMessage) {
			return executorReasonDeadlock
		}
		if execution.Result == models.CanvasNodeExecutionResultFailed {
			return classifyExecutionFailure(execution)
		}
	}

	return executorReasonInternal
}

func classifyExecutionFailure(execution *models.CanvasNodeExecution) string {
	if execution.Result != models.CanvasNodeExecutionResultFailed {
		return executorReasonNone
	}

	if isDeadlockMessage(execution.ResultMessage) {
		return executorReasonDeadlock
	}

	return execution.ResultReason
}

func isDeadlockError(err error) bool {
	for err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "40P01" {
			return true
		}
		if isDeadlockMessage(err.Error()) {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

func isDeadlockMessage(message string) bool {
	return strings.Contains(message, "deadlock detected") || strings.Contains(message, "40P01")
}
