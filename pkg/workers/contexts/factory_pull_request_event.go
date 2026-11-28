package contexts

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
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
	if created && isCustomAutomationIntegrationTrigger(node) {
		if err := recordPullRequestMergeHandoff(tx, node, payloadType, payload); err != nil {
			return nil, false, err
		}
	}
	return event, created, nil
}

// recordPullRequestMergeHandoff marks a merged factory pull request when an
// integration trigger in the current stage of its task gets the merge. That
// automation moves the task, so PR Closure leaves the task open. The webhook
// handler publishes events after all nodes ran, so PR Closure sees the mark.
func recordPullRequestMergeHandoff(tx *gorm.DB, node *models.CanvasNode, payloadType string, payload any) error {
	data, _ := payload.(map[string]any)
	pullRequest, _ := data["pullRequest"].(*core.PullRequest)
	if pullRequest == nil || !pullRequestMergedPayload(payloadType, data) {
		return nil
	}
	pullRequestID, err := uuid.Parse(pullRequest.ID)
	if err != nil {
		return nil
	}
	orderID, err := uuid.Parse(pullRequest.WorkOrderID)
	if err != nil {
		return nil
	}
	canvas, err := models.FindUnscopedCanvasInTransaction(tx, node.WorkflowID)
	if err != nil || canvas.FactoryID == nil || canvas.ColumnKey == nil {
		return err
	}
	factory, err := models.FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return err
	}
	order, err := factory.FindWorkOrder(tx, orderID)
	if err != nil {
		return err
	}
	stage, err := order.CurrentStageKey(tx)
	if err != nil || stage != *canvas.ColumnKey {
		return err
	}
	return order.RecordPullRequestMergeHandoff(tx, pullRequestID, canvas.ID)
}

func pullRequestMergedPayload(payloadType string, data map[string]any) bool {
	switch payloadType {
	case "github.pullRequest":
		pullRequest, _ := data["pull_request"].(map[string]any)
		return data["action"] == "closed" && pullRequest["merged"] == true
	case "bitbucket.pullRequest":
		return data["action"] == "merged"
	}
	return false
}

func (c *FactoryContext) pullRequestMergeHandedOff(order *models.FactoryWorkOrder) (bool, error) {
	if c.execution == nil {
		return false, nil
	}
	event, err := models.FindRootEventForRun(c.tx, c.execution.RunID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	payload, _ := event.Data.Data().(map[string]any)
	payloadType, _ := payload["type"].(string)
	data, _ := payload["data"].(map[string]any)
	if !pullRequestMergedPayload(payloadType, data) {
		return false, nil
	}
	pullRequest, _ := data["pullRequest"].(map[string]any)
	id, _ := pullRequest["id"].(string)
	pullRequestID, err := uuid.Parse(id)
	if err != nil {
		return false, nil
	}
	return order.PullRequestMergeHandedOff(c.tx, pullRequestID)
}
