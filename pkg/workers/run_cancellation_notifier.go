package workers

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

// RunCancellationOutcome records a child run cancellation performed during a transaction.
type RunCancellationOutcome struct {
	WorkflowID  uuid.UUID
	RunID       uuid.UUID
	DrainResult *models.RunCancellationDrainResult
}

// RunCancellationNotifier collects child run cancellations during a database transaction
// so callers can publish RabbitMQ messages after the transaction commits.
type RunCancellationNotifier struct {
	Outcomes []RunCancellationOutcome
	deferred []deferredWorkOrderCancellation
}

type deferredWorkOrderCancellation struct {
	orderID  uuid.UUID
	runIDs   []uuid.UUID
	excluded uuid.UUID
}

func (n *RunCancellationNotifier) Bind(ctx *contexts.RunExecutionContext) *contexts.RunExecutionContext {
	if n == nil {
		return ctx
	}

	return ctx.WithRunCancelled(n.record)
}

func (n *RunCancellationNotifier) Publish() {
	if n == nil {
		return
	}

	for _, outcome := range n.Outcomes {
		messages.PublishRunCancellationDrain(outcome.WorkflowID, outcome.DrainResult)

		if err := messages.NewCanvasRunMessage(outcome.WorkflowID.String(), outcome.RunID.String()).Publish(); err != nil {
			log.Errorf("failed to publish run state RabbitMQ message: %v", err)
		}
	}
	for _, item := range n.deferred {
		item := item
		go finishDeferredWorkOrderCancellation(item)
	}
}

func (n *RunCancellationNotifier) RecordDeferred(orderID uuid.UUID, runIDs []uuid.UUID, excluded uuid.UUID) {
	if n == nil || orderID == uuid.Nil || len(runIDs) == 0 {
		return
	}
	n.deferred = append(n.deferred, deferredWorkOrderCancellation{
		orderID:  orderID,
		runIDs:   append([]uuid.UUID(nil), runIDs...),
		excluded: excluded,
	})
}

func finishDeferredWorkOrderCancellation(item deferredWorkOrderCancellation) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Errorf("deferred work order cancellation failed: %v", recovered)
		}
	}()
	order, err := models.FindUnscopedWorkOrder(database.Conn(), item.orderID)
	if err != nil {
		log.WithError(err).Errorf("failed to load work order %s for deferred cancellation", item.orderID)
		return
	}
	messages.PublishDeferredWorkOrderRunCancellations(database.Conn(), nil, item.runIDs, item.excluded)
	messages.PublishWorkOrderRunCancellations(database.Conn(), order)
}

func (n *RunCancellationNotifier) Record(workflowID, runID uuid.UUID, drainResult *models.RunCancellationDrainResult) {
	n.record(workflowID, runID, drainResult)
}

func (n *RunCancellationNotifier) RecordRuns(results []*models.RunCancellationResult) {
	if n == nil {
		return
	}
	for _, result := range results {
		if result == nil || result.Run == nil || result.Drain == nil {
			continue
		}
		n.record(result.Run.WorkflowID, result.Run.ID, result.Drain)
	}
}

func (n *RunCancellationNotifier) record(workflowID, runID uuid.UUID, drainResult *models.RunCancellationDrainResult) {
	n.Outcomes = append(n.Outcomes, RunCancellationOutcome{
		WorkflowID:  workflowID,
		RunID:       runID,
		DrainResult: drainResult,
	})
}
