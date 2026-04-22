package canvases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
)

func EmitNodeEvent(
	ctx context.Context,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	nodeID string,
	channel string,
	data map[string]any,
) (*pb.EmitNodeEventResponse, error) {
	canvas, err := models.FindCanvas(orgID, canvasID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	node, err := canvas.FindNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("canvas node not found: %w", err)
	}

	now := time.Now()
	event := models.CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     nodeID,
		Channel:    channel,
		Data:       datatypes.NewJSONType[any](data),
		State:      models.CanvasEventStatePending,
		CreatedAt:  &now,
	}

	customName, err := resolveCustomName(node, data)
	if err == nil && customName != nil {
		event.CustomName = customName
	}

	if reportEntry := resolveReportEntry(node, data); reportEntry != "" {
		event.ReportEntry = reportEntry
	}

	if err := database.Conn().Create(&event).Error; err != nil {
		log.Errorf("failed to publish workflow event: %v", err)
		return nil, fmt.Errorf("failed to create workflow event: %w", err)
	}

	err = messages.NewCanvasEventCreatedMessage(canvasID.String(), canvas.OrganizationID.String(), &event).Publish()

	if err != nil {
		log.Errorf("failed to publish workflow event RabbitMQ message: %v", err)
	}

	return &pb.EmitNodeEventResponse{
		EventId: event.ID.String(),
	}, nil
}

func resolveCustomName(node *models.CanvasNode, payload map[string]any) (*string, error) {
	config := node.Configuration.Data()
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

	builder := contexts.NewNodeConfigurationBuilder(database.Conn(), node.WorkflowID).
		WithNodeID(node.NodeID).
		WithInput(map[string]any{node.NodeID: payload})
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

func resolveReportEntry(node *models.CanvasNode, payload map[string]any) string {
	config := node.Configuration.Data()
	if config == nil {
		return ""
	}

	rawTemplate, ok := config["reportTemplate"]
	if !ok || rawTemplate == nil {
		return ""
	}

	template, ok := rawTemplate.(string)
	if !ok {
		return ""
	}

	template = strings.TrimSpace(template)
	if template == "" {
		return ""
	}

	//
	// Triggers emit events outside of any execution chain, so we resolve
	// against the raw payload with only root() available.
	//
	resolved, errs := contexts.ResolveReportTemplateFromPayload(template, payload)
	resolved = strings.TrimSpace(resolved)

	if len(errs) > 0 {
		lines := make([]string, 0, len(errs))
		for _, e := range errs {
			lines = append(lines, fmt.Sprintf("> `%s`", e.Error()))
		}
		resolved += fmt.Sprintf("\n\n> [!CAUTION]\n> Expression errors:\n%s", strings.Join(lines, "\n"))
	}

	return resolved
}
