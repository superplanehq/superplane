package factories

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	"gorm.io/gorm"
)

type factoryPullRequestCloseResult struct {
	order     *models.FactoryWorkOrder
	fromState string
	result    string
	closed    bool
}

func closeFactoryWorkOrdersFromGitHubPullRequestClosed(
	db *gorm.DB,
	integration *models.Integration,
	payload githubMergeabilityWebhookPayload,
) {
	if payload.PullRequest == nil || payload.PullRequest.Number <= 0 {
		return
	}

	pullRequests, err := models.ListGitHubFactoryPullRequestsForWebhook(
		db,
		integration.OrganizationID,
		payload.Repository.FullName,
		[]int64{payload.PullRequest.Number},
	)
	if err != nil {
		log.WithError(err).Warn("factory mergeability: failed to list pull requests for closed GitHub event")
		return
	}
	if len(pullRequests) == 0 {
		return
	}

	merged := payload.PullRequest.Merged
	mergedAt := parseGitHubTimestamp(payload.PullRequest.MergedAt)
	closedAt := parseGitHubTimestamp(payload.PullRequest.ClosedAt)

	factoriesByID := map[string]*models.Factory{}
	for i := range pullRequests {
		pullRequest := &pullRequests[i]
		wasMerged := pullRequest.State == models.FactoryPullRequestStateMerged
		factory := factoriesByID[pullRequest.FactoryID.String()]
		if factory == nil {
			loaded, findErr := models.FindFactory(db, pullRequest.OrganizationID, pullRequest.FactoryID)
			if findErr != nil {
				log.WithError(findErr).Warnf("factory mergeability: factory %s not found", pullRequest.FactoryID)
				continue
			}
			factory = loaded
			factoriesByID[factory.ID.String()] = factory
		}

		automation := githubCloseAutomation(payload, merged)
		outcome, err := applyGitHubPullRequestClosed(
			db,
			factory,
			pullRequest,
			merged,
			mergedAt,
			closedAt,
			automation,
		)
		if err != nil {
			log.WithError(err).Warnf("factory mergeability: failed to close work order for pull request %s", pullRequest.ID)
			continue
		}
		identity := ""
		if automation != nil {
			identity = automation.AppName
		}
		if merged && !wasMerged {
			captureExternalFactoryProductEvent(
				db,
				factory,
				pullRequest.WorkOrderID,
				productanalytics.EventFactoryTaskMerged,
				productanalytics.StableEventID(productanalytics.EventFactoryTaskMerged, pullRequest.ID.String()),
				models.ProviderGitHub,
				identity,
				productanalytics.Properties{
					"provider":     models.ProviderGitHub,
					"merge_source": "vcs_webhook",
				},
			)
		}
		if outcome == nil || !outcome.closed {
			continue
		}
		if !merged {
			captureExternalFactoryProductEvent(
				db,
				factory,
				pullRequest.WorkOrderID,
				productanalytics.EventFactoryTaskRejected,
				productanalytics.StableEventID(productanalytics.EventFactoryTaskRejected, pullRequest.WorkOrderID.String(), models.FactoryWorkOrderResultPRClosed),
				models.ProviderGitHub,
				identity,
				productanalytics.Properties{"rejection_source": "pull_request_closed"},
			)
		}
		publishWorkOrderClosed(
			factory.OrganizationID,
			factory,
			outcome.order,
			nil,
			outcome.fromState,
			outcome.result,
			false,
		)
	}
}

func applyGitHubPullRequestClosed(
	db *gorm.DB,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	merged bool,
	mergedAt, closedAt *time.Time,
	automation *factoryevents.AutomationRef,
) (*factoryPullRequestCloseResult, error) {
	var outcome *factoryPullRequestCloseResult
	err := db.Transaction(func(tx *gorm.DB) error {
		order, err := factory.FindWorkOrder(tx, pullRequest.WorkOrderID)
		if err != nil {
			return err
		}
		if err := order.LockForStatusChange(tx); err != nil {
			return err
		}
		if err := pullRequest.LockForUpdate(tx); err != nil {
			return err
		}
		// A reject or archive already recorded this close. A later reopen must
		// survive the webhook for that same close.
		alreadyClosed := !merged && pullRequest.State == models.FactoryPullRequestStateClosed
		if err := stampFactoryPullRequestClosed(tx, pullRequest, merged, mergedAt, closedAt); err != nil {
			return err
		}
		if alreadyClosed {
			return nil
		}
		closed, err := closeFactoryWorkOrderForPullRequest(tx, factory, pullRequest, merged, automation)
		if err != nil {
			return err
		}
		outcome = closed
		return nil
	})
	return outcome, err
}

func reopenFactoryPullRequestsFromGitHub(
	db *gorm.DB,
	integration *models.Integration,
	payload githubMergeabilityWebhookPayload,
) []models.FactoryPullRequest {
	if payload.PullRequest == nil || payload.PullRequest.Number <= 0 {
		return nil
	}

	pullRequests, err := models.ListGitHubFactoryPullRequestsForWebhook(
		db,
		integration.OrganizationID,
		payload.Repository.FullName,
		[]int64{payload.PullRequest.Number},
	)
	if err != nil {
		log.WithError(err).Warn("factory mergeability: failed to list pull requests for reopened GitHub event")
		return nil
	}

	var reopened []models.FactoryPullRequest
	factoriesByID := map[string]*models.Factory{}
	for i := range pullRequests {
		pullRequest := &pullRequests[i]
		if pullRequest.State != models.FactoryPullRequestStateClosed {
			continue
		}
		factory := factoriesByID[pullRequest.FactoryID.String()]
		if factory == nil {
			loaded, findErr := models.FindFactory(db, pullRequest.OrganizationID, pullRequest.FactoryID)
			if findErr != nil {
				log.WithError(findErr).Warnf("factory mergeability: factory %s not found", pullRequest.FactoryID)
				continue
			}
			factory = loaded
			factoriesByID[factory.ID.String()] = factory
		}
		changed, err := reopenFactoryPullRequest(db, factory, pullRequest, payload.PullRequest.Draft)
		if err != nil {
			log.WithError(err).Warnf("factory mergeability: failed to reopen pull request %s", pullRequest.ID)
			continue
		}
		if !changed {
			continue
		}
		reopened = append(reopened, *pullRequest)
	}
	return reopened
}

func reopenFactoryPullRequest(
	db *gorm.DB,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	draft bool,
) (bool, error) {
	changed := false
	err := db.Transaction(func(tx *gorm.DB) error {
		order, err := factory.FindWorkOrder(tx, pullRequest.WorkOrderID)
		if err != nil {
			return err
		}
		if err := order.LockForStatusChange(tx); err != nil {
			return err
		}
		if err := pullRequest.LockForUpdate(tx); err != nil {
			return err
		}
		if pullRequest.State != models.FactoryPullRequestStateClosed {
			return nil
		}
		state := models.FactoryPullRequestStateOpen
		if draft {
			state = models.FactoryPullRequestStateDraft
		}
		if err := pullRequest.Update(tx, models.FactoryPullRequestPatch{State: &state}); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

func stampFactoryPullRequestClosed(
	tx *gorm.DB,
	pullRequest *models.FactoryPullRequest,
	merged bool,
	mergedAt, closedAt *time.Time,
) error {
	if merged {
		if pullRequest.State == models.FactoryPullRequestStateMerged {
			return nil
		}
		state := models.FactoryPullRequestStateMerged
		return pullRequest.Update(tx, models.FactoryPullRequestPatch{
			State:    &state,
			MergedAt: mergedAt,
		})
	}
	if pullRequest.State == models.FactoryPullRequestStateClosed {
		return nil
	}
	state := models.FactoryPullRequestStateClosed
	return pullRequest.Update(tx, models.FactoryPullRequestPatch{
		State:    &state,
		ClosedAt: closedAt,
	})
}

func closeFactoryWorkOrderForPullRequest(
	tx *gorm.DB,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	merged bool,
	automation *factoryevents.AutomationRef,
) (*factoryPullRequestCloseResult, error) {
	order, err := factory.FindWorkOrder(tx, pullRequest.WorkOrderID)
	if err != nil {
		return nil, err
	}
	if err := order.LockForStatusChange(tx); err != nil {
		return nil, err
	}

	outcome := &factoryPullRequestCloseResult{
		order:     order,
		fromState: order.State,
		result:    models.FactoryWorkOrderResultPRClosed,
	}
	if merged {
		outcome.result = models.FactoryWorkOrderResultCompleted
	}
	if !order.IsOpen() {
		return outcome, nil
	}

	if _, err := order.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{
		ToState:    models.FactoryWorkOrderStateClosed,
		Result:     outcome.result,
		Automation: automation,
		SkipSame:   true,
	}); err != nil {
		return nil, err
	}
	outcome.order = order
	outcome.closed = true
	return outcome, nil
}

func githubCloseAutomation(payload githubMergeabilityWebhookPayload, merged bool) *factoryevents.AutomationRef {
	if payload.PullRequest == nil {
		return githubUserAutomation(payload.Sender)
	}
	if merged && payload.PullRequest.MergedBy != nil {
		return githubUserAutomation(payload.PullRequest.MergedBy)
	}
	return githubUserAutomation(payload.Sender)
}

func githubUserAutomation(user *githubWebhookUser) *factoryevents.AutomationRef {
	if user == nil {
		return nil
	}
	login := strings.TrimSpace(user.Login)
	if login == "" {
		return nil
	}
	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = login
	}
	htmlURL := strings.TrimSpace(user.HTMLURL)
	if htmlURL == "" {
		htmlURL = "https://github.com/" + login
	}
	return &factoryevents.AutomationRef{
		AppName:  login,
		NodeName: name,
		NodeID:   htmlURL,
	}
}

func parseGitHubTimestamp(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return &parsed
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed
	}
	return nil
}
