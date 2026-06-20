package contexts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type EventContext struct {
	tx             *gorm.DB
	node           *models.CanvasNode
	maxPayloadSize int
	onNewEvents    func([]models.CanvasEvent)
}

func NewEventContext(tx *gorm.DB, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) *EventContext {
	return &EventContext{tx: tx, node: node, maxPayloadSize: config.MaxPayloadSize(), onNewEvents: onNewEvents}
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
	// We use RawMessage here to avoid a second marshal when GORM persists the JSON value.
	//
	event := models.CanvasEvent{
		WorkflowID: s.node.WorkflowID,
		NodeID:     s.node.NodeID,
		Channel:    "default",
		Data:       models.NewJSONValue(json.RawMessage(data)),
		State:      models.CanvasEventStatePending,
		CreatedAt:  &now,
	}

	wrappedPayload := map[string]any{"data": payload}
	customName, err := s.resolveCustomName(wrappedPayload, structuredPayload)
	if err != nil {
		failed := fmt.Sprintf("Failed to resolve run title: %s", err.Error())
		event.CustomName = &failed
	} else if customName != nil {
		event.CustomName = customName
	}

	parentRunID, err := resolveParentRunID(payload)
	if err != nil {
		return fmt.Errorf("invalid _superplane.parentRunId: %w", err)
	}

	if parentRunID != nil {
		linkedRun, err := models.CreateLinkedCanvasRunInTransaction(s.tx, s.node.WorkflowID, *parentRunID, nil)
		if err != nil {
			return fmt.Errorf("failed to create linked run: %w", err)
		}

		event.RunID = linkedRun.ID
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

func (s *EventContext) resolveCustomName(payload any, rootPayload any) (*string, error) {
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
		WithInput(map[string]any{s.node.NodeID: payload}).
		WithRootPayload(rootPayload)
	resolved, err := builder.ResolveTemplateExpressions(template)
	if err != nil {
		return nil, err
	}

	resolvedName := strings.TrimSpace(fmt.Sprintf("%v", resolved))
	if resolvedName == "" {
		return nil, nil
	}

	return &resolvedName, nil
}

func resolveParentRunID(payload any) (*uuid.UUID, error) {
	payloadMap, ok := payload.(map[string]any)
	if !ok {
		return nil, nil
	}

	rawSuperplane, ok := payloadMap["_superplane"]
	if !ok || rawSuperplane == nil {
		return nil, nil
	}

	superplaneMap, ok := rawSuperplane.(map[string]any)
	if !ok {
		return nil, nil
	}

	rawParentRunID, ok := superplaneMap["parentRunId"]
	if !ok {
		rawParentRunID = superplaneMap["parent_run_id"]
	}
	if rawParentRunID == nil {
		return nil, nil
	}

	parentRunID, ok := rawParentRunID.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", rawParentRunID)
	}

	parsedParentRunID, err := uuid.Parse(parentRunID)
	if err != nil {
		return nil, err
	}

	return &parsedParentRunID, nil
}
