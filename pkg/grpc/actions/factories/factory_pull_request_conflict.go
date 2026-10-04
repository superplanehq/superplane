package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

var errConflictRepairNotEmitted = errors.New("conflict repair did not emit an event")

func startFactoryPullRequestConflictRepair(
	ctx context.Context,
	db *gorm.DB,
	factoryModel *models.Factory,
	pullRequest *models.FactoryPullRequest,
	result *factoryPullRequestMergeability,
) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if factoryModel == nil || pullRequest == nil || result == nil {
		return nil
	}
	headSHA := strings.TrimSpace(result.HeadSHA)
	if headSHA == "" {
		return nil
	}

	handler, err := factoryModel.FindPRFeedbackHandlerBySource(db, models.FactoryPRFeedbackHandlerSourcePullRequestConflicts)
	if err != nil {
		return err
	}
	if handler == nil {
		return nil
	}

	reached, err := pullRequest.ConflictRepairLimitReached(db, handler)
	if err != nil {
		return err
	}
	if reached {
		return nil
	}

	var emitted []models.CanvasEvent
	err = db.Transaction(func(tx *gorm.DB) error {
		claimed, claimErr := models.ClaimFactoryPullRequestConflictHead(tx, handler.ID, pullRequest.ID, headSHA)
		if claimErr != nil {
			return claimErr
		}
		if !claimed {
			return nil
		}
		if emitErr := emitPullRequestConflict(tx, handler, pullRequest, result, &emitted); emitErr != nil {
			return emitErr
		}
		if len(emitted) == 0 {
			return errConflictRepairNotEmitted
		}
		return nil
	})
	if errors.Is(err, errConflictRepairNotEmitted) {
		log.Warnf("factory mergeability: conflict repair did not emit an event for pull request %s", pullRequest.ID)
		return nil
	}
	if err != nil {
		return err
	}
	publishConflictEvents(emitted)
	return nil
}

func emitPullRequestConflict(
	db *gorm.DB,
	handler *models.FactoryPRFeedbackHandler,
	pullRequest *models.FactoryPullRequest,
	result *factoryPullRequestMergeability,
	emitted *[]models.CanvasEvent,
) error {
	node, err := conflictTriggerNode(db, handler.CanvasID)
	if err != nil {
		return err
	}
	if node == nil {
		return fmt.Errorf("conflict trigger is missing on canvas %s", handler.CanvasID)
	}

	events := contexts.NewEventContext(db, node, nil, func(created []models.CanvasEvent) {
		*emitted = append(*emitted, created...)
	})
	return events.Emit(factory.OnPullRequestConflictPayloadType, pullRequestConflictPayload(pullRequest, result))
}

func publishConflictEvents(emitted []models.CanvasEvent) {
	for i := range emitted {
		if err := messages.PublishCanvasEventCreatedMessage(&emitted[i]); err != nil {
			log.WithError(err).Warnf("factory mergeability: failed to publish conflict event %s", emitted[i].ID)
		}
	}
}

func conflictTriggerNode(db *gorm.DB, canvasID uuid.UUID) (*models.CanvasNode, error) {
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, []uuid.UUID{canvasID})
	if err != nil {
		return nil, err
	}
	spec, ok := specs[canvasID]
	if !ok {
		return nil, nil
	}

	nodeID := ""
	for i := range spec.Nodes {
		if spec.Nodes[i].ComponentName() == factory.OnPullRequestConflictTriggerName {
			nodeID = spec.Nodes[i].ID
			break
		}
	}
	if nodeID == "" {
		return nil, nil
	}
	return models.FindCanvasNode(db, canvasID, nodeID)
}

func pullRequestConflictPayload(pullRequest *models.FactoryPullRequest, result *factoryPullRequestMergeability) map[string]any {
	repository := strings.TrimSpace(pullRequest.Repository)
	return map[string]any{
		"repository": map[string]any{
			"full_name": repository,
			"html_url":  "https://github.com/" + repository,
		},
		"pull_request": map[string]any{
			"number":   pullRequest.Number,
			"html_url": pullRequest.URL,
			"head": map[string]any{
				"sha": result.HeadSHA,
				"ref": result.HeadRef,
				"repo": map[string]any{
					"full_name": result.HeadRepository,
				},
			},
			"base": map[string]any{
				"ref": result.BaseRef,
			},
		},
	}
}
