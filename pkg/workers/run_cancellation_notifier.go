package workers

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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
}

func (n *RunCancellationNotifier) record(workflowID, runID uuid.UUID, drainResult *models.RunCancellationDrainResult) {
	n.Outcomes = append(n.Outcomes, RunCancellationOutcome{
		WorkflowID:  workflowID,
		RunID:       runID,
		DrainResult: drainResult,
	})
}
