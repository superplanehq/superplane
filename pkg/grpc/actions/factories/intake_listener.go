package factories

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// intakeListeningByID reports whether each intake can receive live items from
// its source. A bound trigger whose webhook failed to register is not listening,
// even when the canvas graph itself is still connected.
func intakeListeningByID(
	tx *gorm.DB,
	intakes []models.FactoryIntake,
	specs map[uuid.UUID]models.LiveCanvasSpec,
) map[uuid.UUID]bool {
	listening := make(map[uuid.UUID]bool, len(intakes))
	for i := range intakes {
		listening[intakes[i].ID] = true
	}
	if tx == nil || len(intakes) == 0 {
		return listening
	}

	canvasIDs := make([]uuid.UUID, 0, len(intakes))
	nodeIDs := make([]string, 0, len(intakes))
	triggerByCanvas := make(map[uuid.UUID]string, len(intakes))
	for i := range intakes {
		graph := resolveIntakeGraph(intakes[i].Source, specs[intakes[i].CanvasID])
		if graph.TriggerNodeID == "" {
			continue
		}
		canvasIDs = append(canvasIDs, intakes[i].CanvasID)
		nodeIDs = append(nodeIDs, graph.TriggerNodeID)
		triggerByCanvas[intakes[i].CanvasID] = graph.TriggerNodeID
	}
	if len(canvasIDs) == 0 {
		return listening
	}

	var nodes []models.CanvasNode
	if err := tx.
		Where("workflow_id IN ? AND node_id IN ?", canvasIDs, nodeIDs).
		Find(&nodes).Error; err != nil {
		markUnknownListenersDown(listening, intakes, triggerByCanvas)
		return listening
	}

	nodeByCanvas := make(map[uuid.UUID]models.CanvasNode, len(nodes))
	webhookIDs := make([]uuid.UUID, 0, len(nodes))
	for i := range nodes {
		expected, ok := triggerByCanvas[nodes[i].WorkflowID]
		if !ok || nodes[i].NodeID != expected {
			continue
		}
		nodeByCanvas[nodes[i].WorkflowID] = nodes[i]
		if nodes[i].WebhookID != nil {
			webhookIDs = append(webhookIDs, *nodes[i].WebhookID)
		}
	}

	webhookByID := map[uuid.UUID]models.Webhook{}
	if len(webhookIDs) > 0 {
		var webhooks []models.Webhook
		if err := tx.Where("id IN ?", webhookIDs).Find(&webhooks).Error; err != nil {
			markUnknownListenersDown(listening, intakes, triggerByCanvas)
			return listening
		}
		for i := range webhooks {
			webhookByID[webhooks[i].ID] = webhooks[i]
		}
	}

	for i := range intakes {
		node, ok := nodeByCanvas[intakes[i].CanvasID]
		if !ok {
			continue
		}
		listening[intakes[i].ID] = intakeTriggerIsListening(node, webhookByID)
	}

	return listening
}

func markUnknownListenersDown(
	listening map[uuid.UUID]bool,
	intakes []models.FactoryIntake,
	triggerByCanvas map[uuid.UUID]string,
) {
	for i := range intakes {
		if triggerByCanvas[intakes[i].CanvasID] != "" {
			listening[intakes[i].ID] = false
		}
	}
}

func intakeTriggerIsListening(node models.CanvasNode, webhookByID map[uuid.UUID]models.Webhook) bool {
	if node.WebhookID != nil {
		webhook, ok := webhookByID[*node.WebhookID]
		return ok && webhook.State != models.WebhookStateFailed
	}
	return node.AppInstallationID == nil
}
