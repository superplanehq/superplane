package messages

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const EventsExchange = "superplane.events-exchange"

const (
	EventCreatedRoutingKey  = "event.created"
	EventTerminalRoutingKey = "event.terminal"
)

type CanvasEventCreatedMessage struct {
	message *pb.CanvasNodeEventMessage
}

func NewCanvasEventCreatedMessage(canvasId string, organizationID string, event *models.CanvasEvent) CanvasEventCreatedMessage {
	return CanvasEventCreatedMessage{
		message: &pb.CanvasNodeEventMessage{
			Id:             event.ID.String(),
			CanvasId:       canvasId,
			NodeId:         event.NodeID,
			Timestamp:      timestamppb.Now(),
			OrganizationId: organizationID,
		},
	}
}

func (m CanvasEventCreatedMessage) Publish() error {
	return Publish(EventsExchange, EventCreatedRoutingKey, toBytes(m.message))
}

func PublishCanvasEventCreatedMessage(event *models.CanvasEvent) error {
	return PublishCanvasEventCreatedMessages([]models.CanvasEvent{*event})
}

func PublishCanvasEventCreatedMessages(events []models.CanvasEvent) error {
	if len(events) == 0 {
		return nil
	}

	workflowIDs := make([]uuid.UUID, 0, len(events))
	seen := make(map[uuid.UUID]struct{}, len(events))
	for _, event := range events {
		if _, ok := seen[event.WorkflowID]; ok {
			continue
		}
		seen[event.WorkflowID] = struct{}{}
		workflowIDs = append(workflowIDs, event.WorkflowID)
	}

	canvases, err := models.FindCanvasesByIDs(database.Conn(), workflowIDs)
	if err != nil {
		return err
	}

	for i := range events {
		event := events[i]
		canvas, ok := canvases[event.WorkflowID]
		if !ok {
			return fmt.Errorf("canvas %s not found for event %s", event.WorkflowID, event.ID)
		}
		if err := NewCanvasEventCreatedMessage(
			event.WorkflowID.String(),
			canvas.OrganizationID.String(),
			&event,
		).Publish(); err != nil {
			return err
		}
	}

	return nil
}

type CanvasEventTerminalMessage struct {
	message *pb.CanvasEventTerminalMessage
}

func NewCanvasEventTerminalMessage(canvasID, runID, eventID uuid.UUID) CanvasEventTerminalMessage {
	return CanvasEventTerminalMessage{
		message: &pb.CanvasEventTerminalMessage{
			EventId:   eventID.String(),
			CanvasId:  canvasID.String(),
			RunId:     runID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasEventTerminalMessage) Publish() error {
	return Publish(EventsExchange, EventTerminalRoutingKey, toBytes(m.message))
}

func PublishEventTerminal(canvasID, runID, eventID uuid.UUID) error {
	return NewCanvasEventTerminalMessage(canvasID, runID, eventID).Publish()
}
