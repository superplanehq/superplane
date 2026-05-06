package contexts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type EventContext struct {
	tx                      *gorm.DB
	node                    *models.CanvasNode
	maxPayloadSize          int
	defaultRunTitleTemplate string
	onNewEvents             func([]models.CanvasEvent)
}

func NewEventContext(
	tx *gorm.DB,
	node *models.CanvasNode,
	onNewEvents func([]models.CanvasEvent),
	registries ...*registry.Registry,
) *EventContext {
	ctx := &EventContext{tx: tx, node: node, maxPayloadSize: DefaultMaxPayloadSize, onNewEvents: onNewEvents}
	if len(registries) > 0 {
		ctx.defaultRunTitleTemplate = defaultRunTitleTemplate(registries[0], node)
	}

	return ctx
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
	template := strings.TrimSpace(s.defaultRunTitleTemplate)
	if s.node.RunTitleTemplate != nil {
		template = strings.TrimSpace(*s.node.RunTitleTemplate)
	}

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

func defaultRunTitleTemplate(registry *registry.Registry, node *models.CanvasNode) string {
	if registry == nil || node == nil || node.Type != models.NodeTypeTrigger {
		return ""
	}

	ref := node.Ref.Data()
	if ref.Trigger == nil {
		return ""
	}

	trigger, err := registry.GetTrigger(ref.Trigger.Name)
	if err != nil {
		return ""
	}

	return trigger.DefaultRunTitle()
}
