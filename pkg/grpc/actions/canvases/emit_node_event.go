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

	customName, err := resolveConfigTemplate(node, "customName", data)
	if err == nil && customName != nil {
		event.CustomName = customName
	}

	event.ReportEntry = resolveReportTemplate(node, data)

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

func resolveConfigTemplate(node *models.CanvasNode, fieldName string, payload map[string]any) (*string, error) {
	config := node.Configuration.Data()
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

	resolved, err := contexts.ResolveCustomNameTemplate(tmpl, payload)
	if err != nil {
		return nil, err
	}

	resolved = strings.TrimSpace(resolved)
	if resolved == "" {
		return nil, nil
	}

	return &resolved, nil
}

func resolveReportTemplate(node *models.CanvasNode, payload map[string]any) *string {
	config := node.Configuration.Data()
	if config == nil {
		return nil
	}

	rawTemplate, ok := config["reportTemplate"]
	if !ok || rawTemplate == nil {
		return nil
	}

	tmpl, ok := rawTemplate.(string)
	if !ok {
		return nil
	}

	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		return nil
	}

	resolved, errs := contexts.ResolveReportTemplateFromPayload(tmpl, payload)
	resolved = strings.TrimSpace(resolved)

	if len(errs) > 0 {
		var lines []string
		for _, err := range errs {
			lines = append(lines, fmt.Sprintf("> `%s`", err.Error()))
		}

		resolved += fmt.Sprintf("\n\n> [!CAUTION]\n> Expression errors:\n%s", strings.Join(lines, "\n"))
	}

	if resolved == "" {
		return nil
	}

	return &resolved
}
