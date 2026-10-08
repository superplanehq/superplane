package factories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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

const (
	visualEvidenceCaptureAttempts = 3
	visualEvidenceCaptureDelay    = 200 * time.Millisecond
	visualEvidenceCaptureLease    = 2 * time.Minute
	visualEvidenceCaptureMaxDelay = 5 * time.Minute
)

var errVisualEvidenceCaptureTemporary = errors.New("visual evidence capture failed temporarily")

var publishVisualEvidenceEvent = messages.PublishCanvasEventCreatedMessage

var fetchVisualEvidencePullRequest = func(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	repository string,
	number int,
) (*github.PullRequest, error) {
	client, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		return nil, err
	}
	pullRequest, _, err := client.GetPullRequest(ctx, repository, number)
	if err != nil {
		return nil, err
	}
	if pullRequest == nil {
		return nil, errFactoryPullRequestMissing
	}
	return pullRequest, nil
}

// StartVisualEvidenceCapture starts the first capture after a GitHub pull
// request is attached to a task. The open webhook can arrive before that
// attachment, so the Verify app does not listen for it.
func StartVisualEvidenceCapture(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID, factoryID, pullRequestID uuid.UUID,
) {
	if organizationID == uuid.Nil || factoryID == uuid.Nil || pullRequestID == uuid.Nil {
		return
	}
	ctx = context.WithoutCancel(ctx)
	go func() {
		if err := startVisualEvidenceCapture(ctx, deps, organizationID, factoryID, pullRequestID); err != nil {
			log.WithError(err).Warnf("visual evidence: failed to start capture for pull request %s", pullRequestID)
		}
	}()
}

func startVisualEvidenceCapture(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID, factoryID, pullRequestID uuid.UUID,
) error {
	requests, err := enqueueVisualEvidenceCapture(ctx, organizationID, factoryID, pullRequestID)
	if err != nil {
		return err
	}
	if len(requests) == 0 {
		return nil
	}
	return retryVisualEvidenceCapture(ctx, deps, requests)
}

func retryVisualEvidenceCapture(
	ctx context.Context,
	deps IntakeDependencies,
	requests []models.CanvasNodeRequest,
) error {
	var err error
	for attempt := 1; attempt <= visualEvidenceCaptureAttempts; attempt++ {
		err = captureVisualEvidenceRequests(ctx, deps, requests)
		if err == nil || !errors.Is(err, errVisualEvidenceCaptureTemporary) {
			return err
		}
		log.WithError(err).Warnf("visual evidence: capture attempt %d failed", attempt)
		if attempt < visualEvidenceCaptureAttempts && !waitForVisualEvidenceCapture(ctx, attempt) {
			return ctx.Err()
		}
	}
	if scheduleErr := scheduleVisualEvidenceCaptureRetries(database.DB(ctx), requests); scheduleErr != nil {
		return scheduleErr
	}
	return fmt.Errorf("queued visual evidence capture for retry: %w", err)
}

func waitForVisualEvidenceCapture(ctx context.Context, attempt int) bool {
	timer := time.NewTimer(time.Duration(attempt) * visualEvidenceCaptureDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func enqueueVisualEvidenceCapture(
	ctx context.Context,
	organizationID, factoryID, pullRequestID uuid.UUID,
) ([]models.CanvasNodeRequest, error) {
	db := database.DB(ctx)
	factory, err := models.FindFactory(db, organizationID, factoryID)
	if err != nil {
		return nil, err
	}
	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequestID})
	if err != nil {
		return nil, err
	}
	if pullRequest.Provider != models.FactoryPullRequestProviderGitHub || pullRequest.State != models.FactoryPullRequestStateOpen {
		return nil, nil
	}

	nodes, err := models.ListFactoryAppTemplateNodes(db, organizationID, factoryID, models.FactoryAppTemplateVisualEvidenceID)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	requests := make([]models.CanvasNodeRequest, 0, len(nodes))
	for i := range nodes {
		node := nodes[i]
		var saved *models.CanvasNodeRequest
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := lockVisualEvidenceCapture(tx, node.WorkflowID, pullRequest.Number); err != nil {
				return err
			}
			started, err := models.CanvasHasGitHubPullRequestEvent(
				tx,
				node.WorkflowID,
				node.NodeID,
				pullRequest.Repository,
				pullRequest.Number,
			)
			if err != nil || started {
				return err
			}
			existing, err := models.FindPendingVisualEvidenceCaptureRequest(tx, node.WorkflowID, node.NodeID, pullRequest.ID)
			if err == nil {
				saved = existing
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			current, err := models.FindCanvasNode(tx, node.WorkflowID, node.NodeID)
			if err != nil {
				return err
			}
			saved, err = current.CreateVisualEvidenceCaptureRequest(tx, models.VisualEvidenceCaptureRequest{
				OrganizationID: organizationID.String(),
				FactoryID:      factoryID.String(),
				PullRequestID:  pullRequest.ID.String(),
			}, time.Now().Add(visualEvidenceCaptureLease))
			return err
		})
		if err != nil {
			return nil, err
		}
		if saved != nil {
			requests = append(requests, *saved)
		}
	}
	return requests, nil
}

func captureVisualEvidenceRequests(
	ctx context.Context,
	deps IntakeDependencies,
	requests []models.CanvasNodeRequest,
) error {
	var firstErr error
	for i := range requests {
		events, err := captureVisualEvidenceRequest(ctx, deps, requests[i].ID)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		for j := range events {
			if err := publishVisualEvidenceEvent(&events[j]); err != nil {
				log.WithError(err).Warnf("visual evidence: failed to publish event %s", events[j].ID)
			}
		}
	}
	return firstErr
}

// ProcessVisualEvidenceCaptureRequest finishes a due capture request.
// A temporary GitHub error leaves the request pending for a later attempt.
func ProcessVisualEvidenceCaptureRequest(
	ctx context.Context,
	deps IntakeDependencies,
	requestID uuid.UUID,
) ([]models.CanvasEvent, error) {
	db := database.DB(ctx)
	var claimed *models.CanvasNodeRequest
	err := db.Transaction(func(tx *gorm.DB) error {
		var claimErr error
		claimed, claimErr = models.ClaimDueVisualEvidenceCaptureRequest(tx, requestID, time.Now().Add(visualEvidenceCaptureLease))
		return claimErr
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	events, err := captureVisualEvidenceRequest(ctx, deps, claimed.ID)
	if err == nil {
		return events, nil
	}
	if !errors.Is(err, errVisualEvidenceCaptureTemporary) {
		return nil, err
	}
	if scheduleErr := scheduleVisualEvidenceCaptureRetry(db, requestID); scheduleErr != nil {
		return nil, scheduleErr
	}
	log.WithError(err).Warnf("visual evidence: capture for request %s will retry", requestID)
	return nil, nil
}

func captureVisualEvidenceRequest(
	ctx context.Context,
	deps IntakeDependencies,
	requestID uuid.UUID,
) ([]models.CanvasEvent, error) {
	db := database.DB(ctx)
	var request models.CanvasNodeRequest
	if err := db.First(&request, "id = ?", requestID).Error; err != nil {
		return nil, err
	}
	if request.State != models.NodeExecutionRequestStatePending {
		return nil, nil
	}
	spec := request.Spec.Data().VisualEvidenceCapture
	if spec == nil {
		if err := completeVisualEvidenceCapture(db, &request); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("visual evidence capture request %s has no pull request", request.ID)
	}
	organizationID, err := uuid.Parse(spec.OrganizationID)
	if err != nil {
		return nil, completeInvalidVisualEvidenceCapture(db, &request, err)
	}
	factoryID, err := uuid.Parse(spec.FactoryID)
	if err != nil {
		return nil, completeInvalidVisualEvidenceCapture(db, &request, err)
	}
	pullRequestID, err := uuid.Parse(spec.PullRequestID)
	if err != nil {
		return nil, completeInvalidVisualEvidenceCapture(db, &request, err)
	}

	factory, err := models.FindFactory(db, organizationID, factoryID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, completeVisualEvidenceCapture(db, &request)
	}
	if err != nil {
		return nil, err
	}
	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequestID})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, completeVisualEvidenceCapture(db, &request)
	}
	if err != nil {
		return nil, err
	}
	if pullRequest.Provider != models.FactoryPullRequestProviderGitHub || pullRequest.State != models.FactoryPullRequestStateOpen {
		return nil, completeVisualEvidenceCapture(db, &request)
	}

	node, err := models.FindCanvasNode(db, request.WorkflowID, request.NodeID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && node.State == models.CanvasNodeStateError) {
		return nil, completeVisualEvidenceCapture(db, &request)
	}
	if err != nil {
		return nil, err
	}

	started, err := models.CanvasHasGitHubPullRequestEvent(db, node.WorkflowID, node.NodeID, pullRequest.Repository, pullRequest.Number)
	if err != nil {
		return nil, err
	}
	if started {
		return nil, completeVisualEvidenceCapture(db, &request)
	}

	githubPR, err := fetchVisualEvidencePullRequest(ctx, db, deps, factory, pullRequest.Repository, int(pullRequest.Number))
	if err != nil {
		return nil, fmt.Errorf("%w: get pull request: %w", errVisualEvidenceCaptureTemporary, err)
	}
	if githubPR.GetDraft() || githubPR.GetState() != models.FactoryPullRequestStateOpen {
		return nil, completeVisualEvidenceCapture(db, &request)
	}
	headSHA := strings.TrimSpace(githubPR.GetHead().GetSHA())
	baseRef := strings.TrimSpace(githubPR.GetBase().GetRef())
	if headSHA == "" || baseRef == "" {
		return nil, fmt.Errorf("%w: pull request %s has no head revision or base branch", errVisualEvidenceCaptureTemporary, pullRequest.ID)
	}

	order, err := factory.FindWorkOrder(db, pullRequest.WorkOrderID)
	if err != nil {
		return nil, err
	}
	payload := visualEvidenceEventData(pullRequest, order, headSHA, baseRef)
	var created []models.CanvasEvent
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := lockVisualEvidenceCapture(tx, node.WorkflowID, pullRequest.Number); err != nil {
			return err
		}
		started, err := models.CanvasHasGitHubPullRequestEvent(tx, node.WorkflowID, node.NodeID, pullRequest.Repository, pullRequest.Number)
		if err != nil {
			return err
		}
		if started {
			return completeVisualEvidenceCapture(tx, &request)
		}
		current, err := models.FindCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			return err
		}
		if err := contexts.NewEventContext(tx, current, nil, func(events []models.CanvasEvent) {
			created = append(created, events...)
		}).Emit("github.pullRequest", payload); err != nil {
			return err
		}
		if len(created) == 0 {
			return errVisualEvidenceCaptureTemporary
		}
		return completeVisualEvidenceCapture(tx, &request)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func completeInvalidVisualEvidenceCapture(tx *gorm.DB, request *models.CanvasNodeRequest, err error) error {
	if completeErr := completeVisualEvidenceCapture(tx, request); completeErr != nil {
		return completeErr
	}
	return fmt.Errorf("visual evidence capture request %s is invalid: %w", request.ID, err)
}

func completeVisualEvidenceCapture(tx *gorm.DB, request *models.CanvasNodeRequest) error {
	if request.State == models.NodeExecutionRequestStateCompleted {
		return nil
	}
	if err := request.Complete(tx); err != nil {
		return err
	}
	request.State = models.NodeExecutionRequestStateCompleted
	return nil
}

func scheduleVisualEvidenceCaptureRetries(db *gorm.DB, requests []models.CanvasNodeRequest) error {
	for i := range requests {
		if err := scheduleVisualEvidenceCaptureRetry(db, requests[i].ID); err != nil {
			return err
		}
	}
	return nil
}

func scheduleVisualEvidenceCaptureRetry(db *gorm.DB, requestID uuid.UUID) error {
	var request models.CanvasNodeRequest
	if err := db.First(&request, "id = ?", requestID).Error; err != nil {
		return err
	}
	if request.State != models.NodeExecutionRequestStatePending {
		return nil
	}
	spec := request.Spec.Data()
	if spec.VisualEvidenceCapture == nil {
		spec.VisualEvidenceCapture = &models.VisualEvidenceCaptureRequest{}
	}
	spec.VisualEvidenceCapture.Attempts++
	request.Spec = datatypes.NewJSONType(spec)
	return request.Reschedule(db, time.Now().Add(visualEvidenceCaptureRetryDelay(spec.VisualEvidenceCapture.Attempts)))
}

func visualEvidenceCaptureRetryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 16 {
		attempts = 16
	}
	delay := visualEvidenceCaptureDelay << (attempts - 1)
	if delay > visualEvidenceCaptureMaxDelay {
		return visualEvidenceCaptureMaxDelay
	}
	return delay
}

func lockVisualEvidenceCapture(tx *gorm.DB, canvasID uuid.UUID, number int64) error {
	lockKey := canvasID.String() + ":" + strconv.FormatInt(number, 10)
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error
}

func visualEvidenceEventData(
	pullRequest *models.FactoryPullRequest,
	order *models.FactoryWorkOrder,
	headSHA, baseRef string,
) map[string]any {
	return map[string]any{
		"action": "opened",
		"number": pullRequest.Number,
		"repository": map[string]any{
			"full_name": pullRequest.Repository,
		},
		"pull_request": map[string]any{
			"number":   pullRequest.Number,
			"title":    pullRequest.Title,
			"html_url": pullRequest.URL,
			"draft":    false,
			"head":     map[string]any{"sha": headSHA},
			"base":     map[string]any{"ref": baseRef},
		},
		"pullRequest": map[string]any{
			"id":          pullRequest.ID.String(),
			"workOrderId": pullRequest.WorkOrderID.String(),
			"provider":    pullRequest.Provider,
			"repository":  pullRequest.Repository,
			"number":      pullRequest.Number,
			"url":         pullRequest.URL,
			"title":       pullRequest.Title,
			"state":       pullRequest.State,
		},
		"workOrder": map[string]any{
			"id":          order.ID.String(),
			"title":       order.Title,
			"description": order.Description,
		},
	}
}
