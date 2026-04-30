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
	onNewEvents    func([]models.CanvasEvent)
}

func NewEventContext(tx *gorm.DB, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) *EventContext {
	return &EventContext{tx: tx, node: node, maxPayloadSize: DefaultMaxPayloadSize, onNewEvents: onNewEvents}
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
	runTitle, err := s.resolveRunTitle(wrappedPayload, structuredPayload)
	if err != nil {
		failed := fmt.Sprintf("Failed to resolve run title: %s", err.Error())
		event.RunTitle = &failed
	} else if runTitle != nil {
		event.RunTitle = runTitle
	}

	err = s.tx.Create(&event).Error
	if err != nil {
		return err
	}

	if s.onNewEvents != nil {
		s.onNewEvents([]models.CanvasEvent{event})
	}

	return nil
}

func (s *EventContext) resolveRunTitle(payload any, rootPayload any) (*string, error) {
	template := strings.TrimSpace(valueOrEmpty(s.node.RunTitleTemplate))
	if template == "" {
		return nil, nil
	}

	builder := NewNodeConfigurationBuilder(s.tx, s.node.WorkflowID).
		WithNodeID(s.node.NodeID).
		WithInput(map[string]any{s.node.NodeID: payload}).
		WithRootPayload(rootPayload)
	resolved, err := builder.ResolveTemplateExpressions(template)
	if err != nil {
		return nil, err
	}

	resolvedTitle := strings.TrimSpace(fmt.Sprintf("%v", resolved))
	if resolvedTitle == "" {
		return nil, nil
	}

	return &resolvedTitle, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
