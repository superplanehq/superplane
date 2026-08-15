package contexts

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// ResolveQueueName returns the resolved queue name for a queue item and
// persists it on the item, so name expressions are evaluated exactly once.
// A node without an explicit queue uses its implicit queue, named after
// the node ID.
func ResolveQueueName(tx *gorm.DB, node *models.CanvasNode, item *models.CanvasNodeQueueItem) (string, error) {
	if item.QueueName != nil {
		return *item.QueueName, nil
	}

	name, err := resolveQueueNameTemplate(tx, node, item)
	if err != nil {
		return "", err
	}

	if err := tx.Model(item).Update("queue_name", name).Error; err != nil {
		return "", err
	}

	item.QueueName = &name
	return name, nil
}

func resolveQueueNameTemplate(tx *gorm.DB, node *models.CanvasNode, item *models.CanvasNodeQueueItem) (string, error) {
	spec := node.QueueSpec()
	if spec == nil || strings.TrimSpace(spec.Key) == "" {
		return node.NodeID, nil
	}

	template := strings.TrimSpace(spec.Key)
	if !strings.Contains(template, "{{") {
		return template, nil
	}

	event, err := models.FindCanvasEventInTransaction(tx, item.EventID)
	if err != nil {
		return "", err
	}

	builder := NewNodeConfigurationBuilder(tx, item.WorkflowID).
		WithNodeID(node.NodeID).
		WithRootEvent(&item.RootEventID).
		WithIncomingEventID(&event.ID).
		WithInput(map[string]any{event.NodeID: event.Data.Data()})
	if event.ExecutionID != nil {
		builder = builder.WithPreviousExecution(event.ExecutionID)
	}

	resolved, err := builder.ResolveTemplateExpressions(template)
	if err != nil {
		return "", fmt.Errorf("error resolving queue key %q: %w", template, err)
	}

	name, ok := resolved.(string)
	if !ok {
		name = fmt.Sprintf("%v", resolved)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("queue key %q resolved to an empty string", template)
	}

	return name, nil
}
