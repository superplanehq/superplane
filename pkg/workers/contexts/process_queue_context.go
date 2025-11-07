package contexts

import (
	"time"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func BuildProcessQueueContext(tx *gorm.DB, node *models.WorkflowNode, queueItem *models.WorkflowNodeQueueItem) (*components.ProcessQueueContext, error) {
	event, err := models.FindWorkflowEventInTransaction(tx, queueItem.EventID)
	if err != nil {
		return nil, err
	}

	config, err := NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
		WithRootEvent(&queueItem.RootEventID).
		WithPreviousExecution(event.ExecutionID).
		WithInput(event.Data.Data()).
		Build(node.Configuration.Data())
	if err != nil {
		return nil, err
	}

	ctx := &components.ProcessQueueContext{
		WorkflowID:    node.WorkflowID.String(),
		NodeID:        node.NodeID,
		Configuration: config,
		RootEventID:   queueItem.RootEventID.String(),
		EventID:       event.ID.String(),
		Input:         event.Data.Data(),
	}

	ctx.CreateExecution = func() error {
		now := time.Now()

		execution := models.WorkflowNodeExecution{
			WorkflowID:          queueItem.WorkflowID,
			NodeID:              node.NodeID,
			RootEventID:         queueItem.RootEventID,
			EventID:             event.ID,
			PreviousExecutionID: event.ExecutionID,
			State:               models.WorkflowNodeExecutionStatePending,
			Configuration:       datatypes.NewJSONType(config),
			CreatedAt:           &now,
			UpdatedAt:           &now,
		}

		if err := tx.Create(&execution).Error; err != nil {
			return err
		}

		messages.NewWorkflowExecutionCreatedMessage(execution.WorkflowID.String(), &execution).PublishWithDelay(1 * time.Second)
		return nil
	}

	ctx.DequeueItem = func() error { return queueItem.Delete(tx) }
	ctx.UpdateNodeState = func(state string) error { return node.UpdateState(tx, state) }
	ctx.DefaultProcessing = func() error {
		if err := ctx.CreateExecution(); err != nil {
			return err
		}
		if err := ctx.DequeueItem(); err != nil {
			return err
		}
		return ctx.UpdateNodeState(models.WorkflowNodeStateProcessing)
	}

	return ctx, nil
}
