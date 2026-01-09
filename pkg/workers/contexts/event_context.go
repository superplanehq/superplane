package contexts

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
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

func (s *EventContext) Emit(payloadType string, payload any) error {
	var v any

	structuredPayload := map[string]any{
		"type":      payloadType,
		"timestamp": time.Now(),
		"data":      payload,
	}

	err := mapstructure.Decode(structuredPayload, &v)
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

	wrappedPayload := map[string]any{"data": payload}
	customName, err := s.resolveCustomName(wrappedPayload)
	if err == nil && customName != nil {
		event.CustomName = customName
	}

	return s.tx.Create(&event).Error
}

func (s *EventContext) resolveCustomName(payload any) (*string, error) {
	config := s.workflowNode.Configuration.Data()
	if config == nil {
		return nil, nil
	}

	rawTemplate, ok := config["customName"]
	if !ok || rawTemplate == nil {
		return nil, nil
	}

	template, ok := rawTemplate.(string)
	if !ok {
		return nil, nil
	}

	template = strings.TrimSpace(template)
	if template == "" {
		return nil, nil
	}

	builder := NewNodeConfigurationBuilder(s.tx, s.workflowNode.WorkflowID).WithInput(payload)
	resolved, err := builder.ResolveExpression(template)
	if err != nil {
		return nil, err
	}

	resolvedName := strings.TrimSpace(fmt.Sprintf("%v", resolved))
	if resolvedName == "" {
		return nil, nil
	}

	return &resolvedName, nil
}
