package factories

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

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
	db := database.DB(ctx)
	factory, err := models.FindFactory(db, organizationID, factoryID)
	if err != nil {
		return err
	}
	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequestID})
	if err != nil {
		return err
	}
	if pullRequest.Provider != models.FactoryPullRequestProviderGitHub || pullRequest.State != models.FactoryPullRequestStateOpen {
		return nil
	}

	nodes, err := models.ListFactoryAppTemplateNodes(db, organizationID, factoryID, models.FactoryAppTemplateVisualEvidenceID)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return nil
	}

	githubPR, err := fetchVisualEvidencePullRequest(ctx, db, deps, factory, pullRequest.Repository, int(pullRequest.Number))
	if err != nil {
		return fmt.Errorf("get pull request: %w", err)
	}
	if githubPR.GetDraft() || githubPR.GetState() != models.FactoryPullRequestStateOpen {
		return nil
	}
	headSHA := strings.TrimSpace(githubPR.GetHead().GetSHA())
	baseRef := strings.TrimSpace(githubPR.GetBase().GetRef())
	if headSHA == "" || baseRef == "" {
		return fmt.Errorf("pull request %s has no head revision or base branch", pullRequest.ID)
	}

	order, err := factory.FindWorkOrder(db, pullRequest.WorkOrderID)
	if err != nil {
		return err
	}
	payload := visualEvidenceEventData(pullRequest, order, headSHA, baseRef)
	var events []models.CanvasEvent
	for i := range nodes {
		node := nodes[i]
		created, err := emitVisualEvidenceEvent(db, &node, pullRequest, payload)
		if err != nil {
			return err
		}
		events = append(events, created...)
	}
	for i := range events {
		if err := publishVisualEvidenceEvent(&events[i]); err != nil {
			log.WithError(err).Warnf("visual evidence: failed to publish event %s", events[i].ID)
		}
	}
	return nil
}

func emitVisualEvidenceEvent(
	db *gorm.DB,
	node *models.CanvasNode,
	pullRequest *models.FactoryPullRequest,
	payload map[string]any,
) ([]models.CanvasEvent, error) {
	var created []models.CanvasEvent
	err := db.Transaction(func(tx *gorm.DB) error {
		lockKey := node.WorkflowID.String() + ":" + strconv.FormatInt(pullRequest.Number, 10)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}
		started, err := visualEvidenceCaptureStarted(tx, node.WorkflowID, node.NodeID, pullRequest.Repository, pullRequest.Number)
		if err != nil {
			return err
		}
		if started {
			return nil
		}
		current, err := models.FindCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			return err
		}
		return contexts.NewEventContext(tx, current, nil, func(events []models.CanvasEvent) {
			created = append(created, events...)
		}).Emit("github.pullRequest", payload)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func visualEvidenceCaptureStarted(tx *gorm.DB, canvasID uuid.UUID, nodeID, repository string, number int64) (bool, error) {
	var count int64
	err := tx.Model(&models.CanvasEvent{}).
		Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).
		Where("data #>> '{data,repository,full_name}' = ?", repository).
		Where("data #>> '{data,pull_request,number}' = ?", strconv.FormatInt(number, 10)).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
