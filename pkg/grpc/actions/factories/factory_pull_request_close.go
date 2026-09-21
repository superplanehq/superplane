package factories

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
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

		outcome, err := applyGitHubPullRequestClosed(db, factory, pullRequest, merged, mergedAt, closedAt)
		if err != nil {
			log.WithError(err).Warnf("factory mergeability: failed to close work order for pull request %s", pullRequest.ID)
			continue
		}
		if outcome == nil || !outcome.closed {
			continue
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
) (*factoryPullRequestCloseResult, error) {
	var outcome *factoryPullRequestCloseResult
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := stampFactoryPullRequestClosed(tx, pullRequest, merged, mergedAt, closedAt); err != nil {
			return err
		}
		closed, err := closeFactoryWorkOrderForPullRequest(tx, factory, pullRequest, merged)
		if err != nil {
			return err
		}
		outcome = closed
		return nil
	})
	return outcome, err
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
) (*factoryPullRequestCloseResult, error) {
	order, err := factory.FindWorkOrder(tx, pullRequest.WorkOrderID)
	if err != nil {
		return nil, err
	}

	outcome := &factoryPullRequestCloseResult{
		order:     order,
		fromState: order.State,
		result:    models.FactoryWorkOrderResultRejected,
	}
	if merged {
		outcome.result = models.FactoryWorkOrderResultCompleted
	}
	if !order.IsOpen() {
		return outcome, nil
	}

	order, err = order.Close(tx, outcome.result, nil)
	if err != nil {
		return nil, err
	}
	outcome.order = order
	outcome.closed = true
	return outcome, nil
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
