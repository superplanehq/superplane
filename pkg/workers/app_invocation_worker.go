package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type AppInvocationWorker struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	logger    *log.Entry
}

func NewAppInvocationWorker(registry *registry.Registry) *AppInvocationWorker {
	return &AppInvocationWorker{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
		logger:    log.WithFields(log.Fields{"worker": "AppInvocationWorker"}),
	}
}

func (w *AppInvocationWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			invaocations, err := models.ListInvocations(database.Conn())
			if err != nil {
				w.logger.Errorf("Error listing pending app messages: %v", err)
				continue
			}

			for _, invocation := range invaocations {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(invocation models.AppInvocation) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessInvocation(invocation); err != nil {
						w.logger.Errorf("Error processing app invocation %s: %v", invocation.ID, err)
					}
				}(invocation)
			}
		}
	}
}

func (w *AppInvocationWorker) LockAndProcessInvocation(invocation models.AppInvocation) error {
	logger := w.logger.WithFields(log.Fields{"invocation": invocation.ID})
	logger.Infof("Locking and processing app invocation")

	var sourceExecution *models.CanvasNodeExecution
	var newEvents []models.CanvasEvent

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockInvocation(tx, invocation.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Infof("Invocation already processed - skipping")
				return nil
			}

			return err
		}

		callerApp, err := models.FindUnscopedCanvasInTransaction(tx, locked.CallerAppID)
		if err != nil {
			return fmt.Errorf("find source canvas: %w", err)
		}

		events, execution, err := w.invokeApp(tx, callerApp, locked)
		if err != nil {
			return err
		}

		newEvents = events
		sourceExecution = execution
		return nil
	})

	if err != nil {
		return err
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	if sourceExecution != nil {
		if err := messages.PublishCanvasExecutionByID(sourceExecution.WorkflowID, sourceExecution.ID); err != nil {
			logger.Errorf("Error publishing execution state: %v", err)
		}
	}

	return nil
}

func (w *AppInvocationWorker) invokeApp(tx *gorm.DB, callerApp *models.Canvas, invocation *models.AppInvocation) ([]models.CanvasEvent, *models.CanvasNodeExecution, error) {
	if invocation.TargetCanvasID == nil {
		return nil, nil, fmt.Errorf("invocation missing target canvas id")
	}

	if invocation.CallerExecutionID == nil {
		return nil, nil, fmt.Errorf("invocation missing caller execution id")
	}

	execution, err := models.FindNodeExecutionInTransaction(tx, invocation.CallerAppID, *invocation.CallerExecutionID)
	if err != nil {
		return nil, nil, fmt.Errorf("find caller execution: %w", err)
	}

	var payload any
	if err := json.Unmarshal(invocation.Payload, &payload); err != nil {
		return nil, nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	targetNode, err := invocation.FindTargetNode(tx)
	if err != nil {
		w.logger.WithError(err).Errorf("Error finding target node %s", invocation.TargetNodeID)
		return nil, nil, fmt.Errorf("find target node: %w", err)
	}

	//
	// Call OnAppMessage() on the target node.
	//
	err = w.sendMessageToNode(tx, callerApp, targetNode, payload, onNewEvents)
	if err != nil {
		execution, hookErr := w.failAppInvocation(tx, invocation, onNewEvents, err.Error())
		if hookErr != nil {
			return nil, nil, hookErr
		}

		return nil, execution, invocation.Delete(tx)
	}

	if err != nil {
		w.logger.WithError(err).Errorf("Error sending message to node %s", targetNode.NodeID)
		return nil, nil, fmt.Errorf("send message to node: %w", err)
	}

	//
	// If no events were generated, something went wrong
	//
	if len(newEvents) != 1 {
		w.logger.Infof("Target app %s did not generate an event", invocation.TargetCanvasID)
		execution, hookErr := w.failAppInvocation(tx, invocation, onNewEvents, "target app did not generate an event")
		if hookErr != nil {
			return nil, nil, hookErr
		}

		return nil, execution, invocation.Delete(tx)
	}

	//
	// Attach run to the invocation record,
	// and call the invocation started hook
	//
	runID := newEvents[0].RunID
	err = invocation.AttachRun(tx, runID)
	if err != nil {
		return nil, nil, fmt.Errorf("start invocation: %w", err)
	}

	err = w.callExecutionHook(tx, execution, "invocationStarted", map[string]any{"run_id": invocation.RunID.String()}, onNewEvents)
	if err != nil {
		return nil, nil, fmt.Errorf("call invocation started hook: %w", err)
	}

	return newEvents, nil, nil
}

func (w *AppInvocationWorker) sendMessageToNode(tx *gorm.DB, callerApp *models.Canvas, targetNode *models.CanvasNode, payload any, onNewEvents func([]models.CanvasEvent)) error {
	ref := targetNode.Ref.Data()
	if targetNode.Type != models.NodeTypeTrigger || ref.Trigger == nil {
		return nil
	}

	triggerName := ref.Trigger.Name
	trigger, err := w.registry.GetTrigger(triggerName)
	if err != nil {
		return fmt.Errorf("trigger %s not found", triggerName)
	}

	message := map[string]any{
		"payload": payload,
		"app": map[string]any{
			"id":   callerApp.ID.String(),
			"name": callerApp.Name,
		},
	}

	appTrigger, ok := trigger.(core.AppTrigger)
	if !ok {
		return nil
	}

	return appTrigger.OnAppMessage(core.AppMessageContext{
		HTTP:          w.registry.HTTPContextInTransaction(tx),
		Configuration: targetNode.Configuration.Data(),
		NodeMetadata:  contexts.NewNodeMetadataContext(tx, targetNode),
		Message:       message,
		Events:        contexts.NewEventContext(tx, targetNode, onNewEvents),
		Logger:        logging.ForNode(*targetNode),
	})
}

func (w *AppInvocationWorker) failAppInvocation(tx *gorm.DB, invocation *models.AppInvocation, onNewEvents func([]models.CanvasEvent), message string) (*models.CanvasNodeExecution, error) {
	execution, err := models.FindNodeExecutionInTransaction(tx, invocation.CallerAppID, *invocation.CallerExecutionID)
	if err != nil {
		return nil, fmt.Errorf("find caller execution: %w", err)
	}

	err = w.callExecutionHook(tx, execution, "invocationFailed", map[string]any{"message": message}, onNewEvents)
	if err != nil {
		return nil, fmt.Errorf("call invocation failed hook: %w", err)
	}

	err = tx.Save(execution).Error
	if err != nil {
		return nil, fmt.Errorf("save execution: %w", err)
	}

	return execution, nil
}

func (w *AppInvocationWorker) callExecutionHook(tx *gorm.DB, execution *models.CanvasNodeExecution, hookName string, parameters map[string]any, onNewEvents func([]models.CanvasEvent)) error {
	node, err := models.FindUnscopedCanvasNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		return fmt.Errorf("find caller node: %w", err)
	}

	ref := node.Ref.Data()
	if ref.Component == nil {
		return fmt.Errorf("node %s is not a component", execution.NodeID)
	}

	hookProvider, hook, err := w.registry.FindActionHook(ref.Component.Name, hookName)
	if err != nil {
		return fmt.Errorf("find invokeApp hook: %w", err)
	}

	logger := logging.ForExecution(execution)
	hookCtx := core.ActionHookContext{
		Name:           hook.Name,
		Configuration:  execution.Configuration.Data(),
		Parameters:     parameters,
		Logger:         logger,
		HTTP:           w.registry.HTTPContextInTransaction(tx),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution, onNewEvents),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
	}

	if err := hookProvider.HandleHook(hookCtx); err != nil {
		return fmt.Errorf("invokeApp hook failed: %w", err)
	}

	return nil
}
