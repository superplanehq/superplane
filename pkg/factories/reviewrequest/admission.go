package reviewrequest

import (
	"fmt"
	"maps"

	"github.com/superplanehq/superplane/pkg/factories/lifecycle"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// admitEvent lets only completion messages start this installed automation.
// Ordinary PR events and broadcasts cannot start reviewer selection.
func admitEvent(event lifecycle.EventAdmission) (lifecycle.EventAdmission, error) {
	template, _ := event.Node.Metadata.Data()[models.FactoryAppTemplateMetadataKey].(map[string]any)
	if template["id"] != TemplateID {
		return event, nil
	}
	event.Suppress = true
	if event.Run != nil || event.Node.ComponentName() != triggerComponent || event.PayloadType != broadcastEventType {
		return event, nil
	}
	if event.Node.Configuration.Data()["enabled"] == false {
		return event, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(event.Tx, event.Node.WorkflowID)
	if err != nil {
		return event, err
	}
	source, err := models.ValidateReviewRequestAdmission(event.Tx, canvas, event.Node, event.Payload)
	if err != nil {
		if isSourceRejection(err) {
			return event, nil
		}
		return event, err
	}
	message := maps.Clone(source.Message)
	root, err := source.Run.RootEvent(event.Tx)
	if err != nil {
		return event, err
	}
	producer, _ := models.RootEventSourcePayload(root.Data.Data()).(map[string]any)
	for _, key := range []string{"repository", "pull_request", "workOrder"} {
		if value, ok := producer[key]; ok {
			message[key] = value
		}
	}
	message["reviewRequestSource"] = message["payload"]
	message["pullRequest"] = map[string]any{"id": source.PullRequest.ID.String()}
	event.Payload = message
	event.DedupKey = "review-request:" + source.App.ID.String() + ":" + source.Run.ID.String()
	event.Suppress = false
	return event, nil
}

// retryEventData resolves an explicit retry to the latest applicable report.
func retryEventData(tx *gorm.DB, canvas *models.Canvas, trigger *models.CanvasNode, source *models.CanvasEvent) (any, bool, error) {
	exists, err := installed(tx, canvas)
	if err != nil || !exists {
		return nil, false, err
	}
	data, err := models.ReviewRequestRetryEventData(tx, canvas, trigger, source)
	if err != nil {
		return nil, true, fmt.Errorf("cannot retry review request: %w", err)
	}
	return data, true, nil
}
