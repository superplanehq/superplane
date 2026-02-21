package contexts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type EventContext struct {
	tx             *gorm.DB
	node           *models.CanvasNode
	maxPayloadSize int
}

func NewEventContext(tx *gorm.DB, node *models.CanvasNode) *EventContext {
	return &EventContext{tx: tx, node: node, maxPayloadSize: DefaultMaxPayloadSize}
}

func (s *EventContext) Emit(payloadType string, payload any) error {
	structuredPayload := map[string]any{
		"type":      payloadType,
		"timestamp": time.Now(),
		"data":      payload,
	}

	data, err := json.Marshal(structuredPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	if len(data) > s.maxPayloadSize {
		return fmt.Errorf("event payload too large: %d bytes (max %d)", len(data), s.maxPayloadSize)
	}

	now := time.Now()

	//
	// We use RawMessage here to avoid a second marshal when GORM persists the JSONType.
	//
	event := models.CanvasEvent{
		WorkflowID: s.node.WorkflowID,
		NodeID:     s.node.NodeID,
		Channel:    "default",
		Data:       datatypes.NewJSONType[any](json.RawMessage(data)),
		State:      models.CanvasEventStatePending,
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
	config := s.node.Configuration.Data()
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

	builder := NewNodeConfigurationBuilder(s.tx, s.node.WorkflowID).
		WithNodeID(s.node.NodeID).
		WithInput(map[string]any{s.node.NodeID: payload})
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
