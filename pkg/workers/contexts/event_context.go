package contexts

import (
	"time"

	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/triggers"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type EventContext struct {
	tx           *gorm.DB
	workflowNode *models.WorkflowNode
}

func NewEventContext(tx *gorm.DB, workflowNode *models.WorkflowNode) triggers.EventContext {
	return &EventContext{tx: tx, workflowNode: workflowNode}
}

func (s *EventContext) Emit(data any) error {
	now := time.Now()
	event := models.WorkflowEvent{
		WorkflowID: s.workflowNode.WorkflowID,
		NodeID:     s.workflowNode.NodeID,
		Channel:    "default",
		Data:       datatypes.NewJSONType(data),
		State:      models.WorkflowEventStatePending,
		CreatedAt:  &now,
	}

	if err := s.tx.Create(&event).Error; err != nil {
		return err
	}

	messages.NewWorkflowEventCreatedMessage(s.workflowNode.WorkflowID.String(), &event).PublishWithDelay(100 * time.Millisecond)

	return nil
}
