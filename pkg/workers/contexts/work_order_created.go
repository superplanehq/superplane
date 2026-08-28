package contexts

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// EmitWorkOrderCreated fans a new work order out to every On Work Order
// trigger in the factory. Failures are logged: the work order already exists,
// and a missed score can be retried by creating the item again.
func EmitWorkOrderCreated(tx *gorm.DB, factoryModel *models.Factory, order *models.FactoryWorkOrder) {
	if factoryModel == nil || order == nil {
		return
	}

	if err := emitWorkOrderCreated(tx, factoryModel, order); err != nil {
		log.WithError(err).Warnf("failed to emit onWorkOrder for work order %s", order.ID)
	}
}

func emitWorkOrderCreated(tx *gorm.DB, factoryModel *models.Factory, order *models.FactoryWorkOrder) error {
	canvases, err := factoryModel.ListCanvases(tx)
	if err != nil {
		return err
	}

	live := make([]models.Canvas, 0, len(canvases))
	ids := make([]uuid.UUID, 0, len(canvases))
	for i := range canvases {
		if canvases[i].LiveVersionID == nil {
			continue
		}
		live = append(live, canvases[i])
		ids = append(ids, canvases[i].ID)
	}
	if len(live) == 0 {
		return nil
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, ids)
	if err != nil {
		return err
	}

	payload := workOrderCreatedPayload(order)
	emitted := []models.CanvasEvent{}

	for i := range live {
		spec, ok := specs[live[i].ID]
		if !ok {
			continue
		}
		nodeID := onWorkOrderNodeID(spec)
		if nodeID == "" {
			continue
		}

		node, err := models.FindCanvasNode(tx, live[i].ID, nodeID)
		if err != nil {
			return err
		}

		events := NewEventContext(tx, node, nil, func(created []models.CanvasEvent) {
			emitted = append(emitted, created...)
		})
		if err := events.Emit(factory.OnWorkOrderPayloadType, payload); err != nil {
			return err
		}
	}

	for i := range emitted {
		if err := messages.PublishCanvasEventCreatedMessage(&emitted[i]); err != nil {
			log.Warnf("failed to publish onWorkOrder event %s: %v", emitted[i].ID, err)
		}
	}

	return nil
}

func workOrderCreatedPayload(order *models.FactoryWorkOrder) map[string]any {
	workOrder := map[string]any{
		"id":          order.ID.String(),
		"title":       order.Title,
		"description": order.Description,
		"number":      order.Number,
		"state":       order.State,
	}
	if order.OriginURL != nil && *order.OriginURL != "" {
		origin := map[string]any{"url": *order.OriginURL}
		if order.OriginLabel != nil && *order.OriginLabel != "" {
			origin["label"] = *order.OriginLabel
		}
		workOrder["origin"] = origin
	}

	return map[string]any{"workOrder": workOrder}
}

func onWorkOrderNodeID(spec models.LiveCanvasSpec) string {
	for i := range spec.Nodes {
		if spec.Nodes[i].Type != models.NodeTypeTrigger {
			continue
		}
		if spec.Nodes[i].ComponentName() == factory.OnWorkOrderTriggerName {
			return spec.Nodes[i].ID
		}
	}
	return ""
}
