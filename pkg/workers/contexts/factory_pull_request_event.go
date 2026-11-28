package contexts

import (
	"errors"
	"strings"

	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// EmitPullRequestEvent is the explicit factory PR producer used by webhooks and startup.
func (c *FactoryContext) EmitPullRequestEvent(nodeID, payloadType string, payload any) error {
	node, err := models.FindCanvasNode(c.tx, c.canvas.ID, nodeID)
	if err != nil {
		return err
	}
	return EmitFactoryPullRequestEvent(c.tx, node, payloadType, payload, c.onNewEvents)
}

// EmitFactoryPullRequestEvent applies producer policy to an already loaded trigger.
func EmitFactoryPullRequestEvent(tx *gorm.DB, node *models.CanvasNode, payloadType string, payload any, onNewEvents func([]models.CanvasEvent)) error {
	event, created, err := CreateFactoryPullRequestEvent(tx, node, payloadType, payload)
	if err != nil {
		return err
	}
	if created && onNewEvents != nil {
		onNewEvents([]models.CanvasEvent{*event})
	}
	return nil
}

// CreateFactoryPullRequestEvent returns the new or reused event under producer policy.
// The caller publishes newly created events after its transaction commits.
func CreateFactoryPullRequestEvent(tx *gorm.DB, node *models.CanvasNode, payloadType string, payload any) (*models.CanvasEvent, bool, error) {
	events := NewEventContext(tx, node, nil, nil)
	event, err := events.buildEvent(payloadType, payload)
	if err != nil {
		return nil, false, err
	}
	handled, created, err := mergeconfidence.CreateWebhookEvent(tx, node, event, payloadType, payload)
	if !handled {
		key, suppress := customAutomationWebhookPolicy(tx, node, payloadType, payload)
		if suppress {
			return nil, false, nil
		}
		switch {
		case key != "":
			created, err = node.CreateDeduplicatedRootEvent(tx, event, key)
		case (payloadType == "github.pullRequest" || payloadType == "bitbucket.pullRequest") && node.Type == models.NodeTypeTrigger && node.ComponentName() == strings.TrimSuffix(payloadType, ".pullRequest")+".onPullRequest" &&
			!isCustomAutomationIntegrationTrigger(node):
			created, err = node.CreatePullRequestChecksEvent(tx, event)
		default:
			created, err = true, tx.Create(event).Error
		}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return event, created, nil
}
