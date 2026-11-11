package models

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReviewRequestRetryEventData resolves an explicit retry to the latest report
// for the original PR. It creates no event and does not reuse the dedup key.
func ReviewRequestRetryEventData(tx *gorm.DB, canvas *Canvas, trigger *CanvasNode, original *CanvasEvent) (any, error) {
	message, _ := objectMap(RootEventSourcePayload(original.Data.Data()))
	payload, _ := objectMap(message["payload"])
	sourceID, err := uuid.Parse(reviewRequestString(trigger.Configuration.Data(), "app"))
	if err != nil {
		return nil, invalidReviewRequestSource("configured source app is required")
	}
	originalApp, _ := objectMap(message["app"])
	if reviewRequestString(originalApp, "id") != sourceID.String() {
		return nil, invalidReviewRequestSource("retry source app does not match")
	}
	originalRunID, err := uuid.Parse(reviewRequestString(payload, "reportId"))
	if err != nil {
		return nil, invalidReviewRequestSource("retry report identity is required")
	}
	if _, err := FindCanvasRunInTransaction(tx, sourceID, originalRunID); err != nil {
		return nil, reviewRequestLookupError(err)
	}
	activity, err := FindPullRequestActivityByRunID(tx, originalRunID)
	if err != nil {
		return nil, err
	}
	latest, err := latestReviewRequestSourceRun(tx, activity.PullRequestID, sourceID)
	if err != nil {
		return nil, err
	}
	app, err := FindCanvasInTransaction(tx, canvas.OrganizationID, sourceID)
	if err != nil {
		return nil, reviewRequestLookupError(err)
	}
	var broadcasts []CanvasEvent
	if err := tx.Where("workflow_id = ? AND run_id = ? AND execution_id IS NOT NULL", sourceID, latest.ID).
		Where("data ->> 'type' = 'app.broadcast'").Order("created_at DESC, id DESC").Find(&broadcasts).Error; err != nil {
		return nil, err
	}
	for _, broadcast := range broadcasts {
		output, _ := objectMap(broadcast.Data.Data())
		latestPayload, _ := objectMap(output["data"])
		if latestPayload["type"] != "merge-confidence.completed" {
			continue
		}
		latestMessage := map[string]any{
			"app":     map[string]any{"id": app.ID.String(), "name": app.Name},
			"node":    map[string]any{"id": broadcast.NodeID},
			"payload": latestPayload,
		}
		source, err := ValidateReviewRequestAdmission(tx, canvas, trigger, latestMessage)
		if err != nil {
			return nil, err
		}
		latestMessage["reviewRequestSource"] = map[string]any{
			"reportId": source.Run.ID.String(), "pullRequestId": source.PullRequest.ID.String(),
			"repository": source.PullRequest.Repository, "pullNumber": source.PullRequest.Number,
			"reviewedCommit": latestPayload["reviewedCommit"], "mergeConfidence": source.Report,
		}
		root, err := source.Run.RootEvent(tx)
		if err != nil {
			return nil, err
		}
		producer, _ := objectMap(RootEventSourcePayload(root.Data.Data()))
		for _, key := range []string{"repository", "pull_request", "workOrder"} {
			if value, ok := producer[key]; ok {
				latestMessage[key] = value
			}
		}
		latestMessage["pullRequest"] = map[string]any{"id": source.PullRequest.ID.String()}
		return map[string]any{"type": "app.broadcast", "data": latestMessage}, nil
	}
	return nil, fmt.Errorf("%w: latest report is not complete and published", ErrReviewRequestSourceInvalid)
}
