package public

import (
	"fmt"

	"github.com/google/uuid"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/runners"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

func (s *Server) finishRunnerTask(runnerTask *runners.RunnerTask) error {
	if runnerTask == nil {
		return fmt.Errorf("runner task is nil")
	}

	tx := database.Conn()
	execution, err := models.FindNodeExecutionByID(tx, runnerTask.ExecutionID)
	if err != nil {
		return fmt.Errorf("execution not found: %w", err)
	}

	node, err := models.FindCanvasNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	execCtx := &core.ExecutionContext{
		ID:             execution.ID,
		WorkflowID:     execution.WorkflowID.String(),
		NodeID:         execution.NodeID,
		BaseURL:        s.BaseURL,
		Configuration:  execution.Configuration.Data(),
		HTTP:           s.registry.HTTPContext(),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		NodeMetadata:   contexts.NewNodeMetadataContext(tx, node),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution, onNewEvents),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Logger:         logging.ForExecution(execution, nil),
		Notifications:  contexts.NewNotificationContext(tx, uuid.Nil, execution.WorkflowID),
		CanvasMemory:   contexts.NewCanvasMemoryContext(tx, execution.WorkflowID),
	}

	fleetTask := runners.FleetTaskFromRunnerTask(runnerTask)
	if err := runneraction.FinishFleetTask(execCtx.Metadata, execCtx.ExecutionState, fleetTask, runnerTask.ID.String()); err != nil {
		return err
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}
	return nil
}
