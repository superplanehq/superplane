package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const pullRequestChecksStartLease = 2 * time.Minute

// StartPullRequestChecks processes durable requests after attachment commits.
// The node request worker recovers requests if this prompt attempt fails.
func StartPullRequestChecks(ctx context.Context, deps IntakeDependencies, pullRequestID uuid.UUID) {
	requests, err := models.ListPullRequestChecksStartRequests(database.DB(ctx), pullRequestID)
	if err != nil {
		log.WithError(err).Warnf("checks: failed to load startup requests for pull request %s", pullRequestID)
		return
	}
	for _, request := range requests {
		events, err := ProcessPullRequestChecksStartRequest(ctx, deps, request.ID)
		if err != nil {
			log.WithError(err).Warnf("checks: failed to start monitoring for request %s", request.ID)
		}
		for i := range events {
			if err := messages.PublishCanvasEventCreatedMessage(&events[i]); err != nil {
				log.WithError(err).Warnf("checks: failed to publish event %s", events[i].ID)
			}
		}
	}
}

func ProcessPullRequestChecksStartRequest(ctx context.Context, deps IntakeDependencies, requestID uuid.UUID) ([]models.CanvasEvent, error) {
	db := database.DB(ctx)
	var request *models.CanvasNodeRequest
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		request, err = models.ClaimDuePullRequestChecksStartRequest(tx, requestID, time.Now().Add(pullRequestChecksStartLease))
		return err
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	events, err := startPullRequestChecksRequest(ctx, db, deps, request)
	if err == nil {
		return events, nil
	}
	spec := request.Spec.Data()
	if spec.PullRequestChecksStart == nil {
		if completeErr := request.Complete(db); completeErr != nil {
			return nil, completeErr
		}
		return nil, err
	}
	spec.PullRequestChecksStart.Attempts++
	request.Spec = datatypes.NewJSONType(spec)
	if scheduleErr := request.Reschedule(db, time.Now().Add(visualEvidenceCaptureRetryDelay(spec.PullRequestChecksStart.Attempts))); scheduleErr != nil {
		return nil, scheduleErr
	}
	return nil, fmt.Errorf("checks startup request %s will retry: %w", request.ID, err)
}

func startPullRequestChecksRequest(ctx context.Context, db *gorm.DB, deps IntakeDependencies, request *models.CanvasNodeRequest) ([]models.CanvasEvent, error) {
	spec := request.Spec.Data().PullRequestChecksStart
	if spec == nil {
		return nil, errors.New("checks startup request has no pull request")
	}
	organizationID, err := uuid.Parse(spec.OrganizationID)
	if err != nil {
		return nil, err
	}
	factoryID, err := uuid.Parse(spec.FactoryID)
	if err != nil {
		return nil, err
	}
	pullRequestID, err := uuid.Parse(spec.PullRequestID)
	if err != nil {
		return nil, err
	}
	factory, err := models.FindFactory(db, organizationID, factoryID)
	if errors.Is(err, models.ErrFactoryNotFound) {
		return nil, request.Complete(db)
	}
	if err != nil {
		return nil, err
	}
	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequestID})
	if errors.Is(err, models.ErrFactoryPullRequestNotFound) {
		return nil, request.Complete(db)
	}
	if err != nil {
		return nil, err
	}
	if pullRequest.Provider != models.FactoryPullRequestProviderGitHub || pullRequest.State != models.FactoryPullRequestStateOpen {
		return nil, request.Complete(db)
	}
	node, err := checksStartupTrigger(db, request, factory, pullRequest)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, request.Complete(db)
	}
	client, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		return nil, err
	}
	githubPR, _, err := client.GetPullRequest(ctx, pullRequest.Repository, int(pullRequest.Number))
	if err != nil {
		return nil, fmt.Errorf("get current GitHub pull request: %w", err)
	}
	if githubPR == nil {
		return nil, errFactoryPullRequestMissing
	}
	ignoreDrafts, _ := node.Configuration.Data()["ignoreDrafts"].(bool)
	if githubPR.GetState() != models.FactoryPullRequestStateOpen || (githubPR.GetDraft() && ignoreDrafts) {
		return nil, request.Complete(db)
	}
	if strings.TrimSpace(githubPR.GetHead().GetSHA()) == "" || strings.TrimSpace(githubPR.GetHead().GetRef()) == "" || strings.TrimSpace(githubPR.GetHead().GetRepo().GetFullName()) == "" || strings.TrimSpace(githubPR.GetBase().GetRef()) == "" {
		return nil, fmt.Errorf("pull request %s has incomplete head revision, branch, repository, or base branch", pullRequest.ID)
	}
	order, err := factory.FindWorkOrder(db, pullRequest.WorkOrderID)
	if err != nil {
		return nil, err
	}
	payload, err := pullRequestChecksEventData(pullRequest, order, githubPR)
	if err != nil {
		return nil, err
	}
	var events []models.CanvasEvent
	err = db.Transaction(func(tx *gorm.DB) error {
		current, err := checksStartupTrigger(tx, request, factory, pullRequest)
		if err != nil {
			return err
		}
		if current == nil {
			return request.Complete(tx)
		}
		if err := contexts.NewEventContext(tx, current, nil, func(created []models.CanvasEvent) { events = append(events, created...) }).Emit("github.pullRequest", payload); err != nil {
			return err
		}
		return request.Complete(tx)
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func checksStartupTrigger(tx *gorm.DB, request *models.CanvasNodeRequest, factory *models.Factory, pullRequest *models.FactoryPullRequest) (*models.CanvasNode, error) {
	handler, err := models.FindPRFeedbackHandlerByCanvasID(tx, request.WorkflowID)
	if err != nil || handler == nil {
		return nil, err
	}
	if handler.FactoryID != factory.ID || handler.OrganizationID != factory.OrganizationID {
		return nil, nil
	}
	node, err := handler.GitHubChecksTrigger(tx, pullRequest.Repository)
	if err != nil || node == nil {
		return nil, err
	}
	if node.NodeID != request.NodeID {
		return nil, nil
	}
	return node, nil
}

func pullRequestChecksEventData(pullRequest *models.FactoryPullRequest, order *models.FactoryWorkOrder, githubPR *github.PullRequest) (map[string]any, error) {
	// Marshal the API object to preserve GitHub's complete head and fork data.
	raw, err := json.Marshal(githubPR)
	if err != nil {
		return nil, err
	}
	var current map[string]any
	if err := json.Unmarshal(raw, &current); err != nil {
		return nil, err
	}
	payload := visualEvidenceEventData(pullRequest, order, githubPR.GetHead().GetSHA(), githubPR.GetBase().GetRef(), githubPR.GetBody())
	payload["pull_request"] = current
	payload["number"] = githubPR.GetNumber()
	payload["repository"] = githubPR.GetBase().GetRepo()
	if githubPR.GetBase().GetRepo().GetFullName() == "" {
		payload["repository"] = map[string]any{"full_name": pullRequest.Repository}
	}
	return payload, nil
}
