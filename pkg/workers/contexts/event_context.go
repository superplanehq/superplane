package contexts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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
	customName, err := s.resolveConfigTemplate("customName", wrappedPayload)
	if err == nil && customName != nil {
		event.CustomName = customName
	}

	reportEntry, err := s.resolveConfigTemplate("reportTemplate", wrappedPayload)
	if err != nil {
		log.Errorf("failed to resolve reportTemplate for node %s: %v", s.node.NodeID, err)
	} else if reportEntry != nil {
		event.ReportEntry = reportEntry
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

func (s *EventContext) resolveConfigTemplate(fieldName string, payload any) (*string, error) {
	config := s.node.Configuration.Data()
	if config == nil {
		return nil, nil
	}

	rawTemplate, ok := config[fieldName]
	if !ok || rawTemplate == nil {
		return nil, nil
	}

	tmpl, ok := rawTemplate.(string)
	if !ok {
		return nil, nil
	}

	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		return nil, nil
	}

	resolved, err := ResolveCustomNameTemplate(tmpl, payload)
	if err != nil {
		return nil, err
	}

	resolved = strings.TrimSpace(resolved)
	if resolved == "" {
		return nil, nil
	}

	return &resolved, nil
}
