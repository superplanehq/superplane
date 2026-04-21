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
	tx             *gorm.DB
	node           *models.CanvasNode
	maxPayloadSize int
	onNewEvents    func([]models.CanvasEvent)
}

func NewEventContext(tx *gorm.DB, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) *EventContext {
	return &EventContext{tx: tx, node: node, maxPayloadSize: DefaultMaxPayloadSize, onNewEvents: onNewEvents}
}

func (s *EventContext) Emit(payloadType string, payload any) error {
	now := time.Now()
	structuredPayload := map[string]any{
		"type":      payloadType,
		"timestamp": now.UTC().Format(time.RFC3339Nano),
		"data":      payload,
	}
	rootPayload := BuildRootEventPayload(payload, payloadType, now)

	data, err := json.Marshal(structuredPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	if len(data) > s.maxPayloadSize {
		return fmt.Errorf("event payload too large: %d bytes (max %d)", len(data), s.maxPayloadSize)
	}

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

	runTitle, err := ResolveRootEventRunTitle(
		s.tx,
		s.node,
		rootPayload,
		BuildRootEventRunTitleInput(payload, payloadType, now, "default"),
	)
	if err == nil && runTitle != nil {
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

func BuildRootEventPayload(payload any, payloadType string, createdAt time.Time) map[string]any {
	return map[string]any{
		"type":      payloadType,
		"timestamp": createdAt.UTC().Format(time.RFC3339Nano),
		"data":      payload,
	}
}

func BuildRootEventRunTitleInput(payload any, payloadType string, createdAt time.Time, channel string) map[string]any {
	input := map[string]any{
		"data": payload,
		"event": map[string]any{
			"createdAt": createdAt.UTC().Format(time.RFC3339Nano),
			"type":      payloadType,
			"channel":   channel,
		},
	}

	payloadMap, ok := payload.(map[string]any)
	if !ok {
		return input
	}

	for key, value := range payloadMap {
		input[key] = value
	}

	return input
}

func ResolveRootEventRunTitle(tx *gorm.DB, node *models.CanvasNode, rootPayload any, input any) (*string, error) {
	template, err := resolveRootEventRunTitleTemplate(tx, node)
	if err != nil {
		return nil, err
	}

	if template == "" {
		return nil, nil
	}

	builder := NewNodeConfigurationBuilder(tx, node.WorkflowID).
		WithNodeID(node.NodeID).
		WithRootPayload(rootPayload).
		WithInput(map[string]any{node.NodeID: input})
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

func resolveRootEventRunTitleTemplate(tx *gorm.DB, node *models.CanvasNode) (string, error) {
	liveNodes, _, err := models.FindLiveCanvasSpecInTransaction(tx, node.WorkflowID)
	if err != nil {
		return "", err
	}

	for _, liveNode := range liveNodes {
		if liveNode.ID != node.NodeID {
			continue
		}

		if liveNode.RunTitleTemplate != nil {
			template := strings.TrimSpace(*liveNode.RunTitleTemplate)
			if template != "" {
				return template, nil
			}
		}

		break
	}

	ref := node.Ref.Data()
	if ref.Trigger == nil || ref.Trigger.Name == "" {
		return "", nil
	}

	return registry.DefaultRunTitleForTrigger(ref.Trigger.Name), nil
}
