package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func DiscussionAuthorIsBotForRun(tx *gorm.DB, runID uuid.UUID) *bool {
	event, err := FindRootEventForRun(tx, runID)
	if err != nil || event == nil {
		return nil
	}
	return DiscussionAuthorIsBot(event.Data.Data())
}

func DiscussionAuthorIsBot(eventData any) *bool {
	author, found := discussionAuthor(eventData)
	if !found {
		return nil
	}
	authorType, _ := author["type"].(string)
	authorType = strings.TrimSpace(authorType)
	if authorType == "" {
		return nil
	}
	isBot := strings.EqualFold(authorType, "Bot") || authorType == "app_user"
	return &isBot
}

func FillUnsetDiscussionAuthorFlags(tx *gorm.DB, pullRequestIDs []uuid.UUID) error {
	if len(pullRequestIDs) == 0 {
		return nil
	}

	var activities []FactoryPullRequestRun
	err := tx.
		Where("pull_request_id IN ?", pullRequestIDs).
		Where("author_is_bot IS NULL").
		Where("author_flag_checked_at IS NULL").
		Where("feedback_handler_id IS NOT NULL").
		Find(&activities).Error
	if err != nil {
		return err
	}
	if len(activities) == 0 {
		return nil
	}

	handlerIDs := make([]uuid.UUID, 0, len(activities))
	seenHandlers := map[uuid.UUID]bool{}
	for _, activity := range activities {
		if activity.FeedbackHandlerID == nil || seenHandlers[*activity.FeedbackHandlerID] {
			continue
		}
		seenHandlers[*activity.FeedbackHandlerID] = true
		handlerIDs = append(handlerIDs, *activity.FeedbackHandlerID)
	}

	var handlers []FactoryPRFeedbackHandler
	if err := tx.Where("id IN ?", handlerIDs).Find(&handlers).Error; err != nil {
		return err
	}
	discussionHandlers := map[uuid.UUID]bool{}
	for _, handler := range handlers {
		if handler.Source == FactoryPRFeedbackHandlerSourcePullRequestDiscussion {
			discussionHandlers[handler.ID] = true
		}
	}

	runIDs := make([]uuid.UUID, 0, len(activities))
	for _, activity := range activities {
		if activity.FeedbackHandlerID == nil || !discussionHandlers[*activity.FeedbackHandlerID] {
			continue
		}
		runIDs = append(runIDs, activity.RunID)
	}
	if len(runIDs) == 0 {
		return nil
	}

	flags, err := discussionAuthorFlagsForRuns(tx, runIDs)
	if err != nil {
		return err
	}
	return saveDiscussionAuthorFlagAttempts(tx, runIDs, flags)
}

func discussionAuthorFlagsForRuns(tx *gorm.DB, runIDs []uuid.UUID) (map[uuid.UUID]*bool, error) {
	var roots []runRootEvent
	err := tx.Raw(`
		SELECT DISTINCT ON (run_id) run_id, root_event_id
		FROM workflow_node_executions
		WHERE run_id IN ?
		ORDER BY run_id
	`, runIDs).Scan(&roots).Error
	if err != nil {
		return nil, err
	}

	eventIDs := make([]uuid.UUID, 0, len(roots))
	runsByEvent := map[uuid.UUID][]uuid.UUID{}
	for _, root := range roots {
		if root.RootEventID == uuid.Nil {
			continue
		}
		if _, seen := runsByEvent[root.RootEventID]; !seen {
			eventIDs = append(eventIDs, root.RootEventID)
		}
		runsByEvent[root.RootEventID] = append(runsByEvent[root.RootEventID], root.RunID)
	}
	if len(eventIDs) == 0 {
		return map[uuid.UUID]*bool{}, nil
	}

	var events []CanvasEvent
	if err := tx.Where("id IN ?", eventIDs).Find(&events).Error; err != nil {
		return nil, err
	}

	flags := map[uuid.UUID]*bool{}
	for i := range events {
		flag := DiscussionAuthorIsBot(events[i].Data.Data())
		if flag == nil {
			continue
		}
		for _, runID := range runsByEvent[events[i].ID] {
			flags[runID] = flag
		}
	}
	return flags, nil
}

func saveDiscussionAuthorFlagAttempts(tx *gorm.DB, runIDs []uuid.UUID, flags map[uuid.UUID]*bool) error {
	botRunIDs := make([]uuid.UUID, 0)
	personRunIDs := make([]uuid.UUID, 0)
	unknownRunIDs := make([]uuid.UUID, 0)
	for _, runID := range runIDs {
		flag := flags[runID]
		if flag == nil {
			unknownRunIDs = append(unknownRunIDs, runID)
			continue
		}
		if *flag {
			botRunIDs = append(botRunIDs, runID)
			continue
		}
		personRunIDs = append(personRunIDs, runID)
	}

	now := time.Now()
	if err := updateAuthorIsBot(tx, botRunIDs, true, now); err != nil {
		return err
	}
	if err := updateAuthorIsBot(tx, personRunIDs, false, now); err != nil {
		return err
	}
	return markAuthorFlagChecked(tx, unknownRunIDs, now)
}

func updateAuthorIsBot(tx *gorm.DB, runIDs []uuid.UUID, authorIsBot bool, now time.Time) error {
	if len(runIDs) == 0 {
		return nil
	}
	return tx.Model(&FactoryPullRequestRun{}).
		Where("run_id IN ?", runIDs).
		Where("author_is_bot IS NULL").
		Updates(map[string]any{
			"author_is_bot":          authorIsBot,
			"author_flag_checked_at": now,
			"updated_at":             now,
		}).Error
}

func markAuthorFlagChecked(tx *gorm.DB, runIDs []uuid.UUID, now time.Time) error {
	if len(runIDs) == 0 {
		return nil
	}
	return tx.Model(&FactoryPullRequestRun{}).
		Where("run_id IN ?", runIDs).
		Where("author_is_bot IS NULL").
		Where("author_flag_checked_at IS NULL").
		Updates(map[string]any{
			"author_flag_checked_at": now,
			"updated_at":             now,
		}).Error
}

func discussionAuthor(eventData any) (map[string]any, bool) {
	payload, ok := RootEventSourcePayload(eventData).(map[string]any)
	if !ok {
		return nil, false
	}
	if user := githubDiscussionAuthor(githubDiscussionEventType(eventData), payload); user != nil {
		return user, true
	}
	if author := bitbucketDiscussionAuthor(payload); author != nil {
		return author, true
	}
	return nil, false
}

func githubDiscussionEventType(eventData any) string {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return ""
	}
	eventType, _ := envelope["type"].(string)
	switch eventType {
	case "github.prReview":
		return "pull_request_review"
	case "github.prReviewComment":
		return "pull_request_review_comment"
	case "github.prComment":
		return "issue_comment"
	default:
		return ""
	}
}

func githubDiscussionAuthor(eventType string, data map[string]any) map[string]any {
	if eventType == "pull_request_review" {
		if user := nestedMap(data, "review", "user"); user != nil {
			return user
		}
	}
	if user := nestedMap(data, "comment", "user"); user != nil {
		return user
	}
	if eventType == "" {
		if user := nestedMap(data, "review", "user"); user != nil {
			return user
		}
	}
	if sender, ok := data["sender"].(map[string]any); ok {
		return sender
	}
	return nil
}

func bitbucketDiscussionAuthor(data map[string]any) map[string]any {
	return nestedMap(data, "comment", "author")
}

func nestedMap(data map[string]any, outer, inner string) map[string]any {
	parent, ok := data[outer].(map[string]any)
	if !ok {
		return nil
	}
	child, ok := parent[inner].(map[string]any)
	if !ok {
		return nil
	}
	return child
}

type runRootEvent struct {
	RunID       uuid.UUID `gorm:"column:run_id"`
	RootEventID uuid.UUID `gorm:"column:root_event_id"`
}
