package contexts

import (
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type EventContext struct {
	tx           *gorm.DB
	workflowNode *models.WorkflowNode
}

func NewEventContext(tx *gorm.DB, workflowNode *models.WorkflowNode) *EventContext {
	return &EventContext{tx: tx, workflowNode: workflowNode}
}

func (s *EventContext) Emit(data core.Payload) error {
	var v any
	err := mapstructure.Decode(data, &v)
	if err != nil {
		return err
	}

	now := time.Now()
	event := models.WorkflowEvent{
		WorkflowID: s.workflowNode.WorkflowID,
		NodeID:     s.workflowNode.NodeID,
		Channel:    "default",
		Data:       datatypes.NewJSONType(v),
		State:      models.WorkflowEventStatePending,
		CreatedAt:  &now,
	}

	return s.tx.Create(&event).Error
}
