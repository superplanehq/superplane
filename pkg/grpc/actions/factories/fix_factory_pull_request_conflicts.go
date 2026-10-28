package factories

import (
	"context"
	"errors"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const (
	factoryPullRequestConflictEventType = "factory.pullRequestConflict"
	factoryPullRequestConflictTitle     = "Resolve merge conflicts"
	factoryPullRequestConflictBody      = "Update the branch so it merges cleanly."
)

// FixFactoryPullRequestConflicts starts the Fix pull request checks runner for a
// conflicting pull request. The run is recorded inside SuperPlane. It does
// not post a pull request comment and it does not wait for status checks.
func FixFactoryPullRequestConflicts(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.FixFactoryPullRequestConflictsRequest,
) (*pb.FixFactoryPullRequestConflictsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start the conflict fix")
	}

	db := database.DB(ctx)
	factory, pullRequest, err := loadFactoryPullRequestForMerge(db, orgID, req.GetFactoryId(), req.GetPrId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start the conflict fix")
	}

	organization, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start the conflict fix")
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryPullRequestMerge) {
		return nil, factoryErrorToStatus(errFactoryPullRequestMergeDisabled, "failed to start the conflict fix")
	}

	result, err := readFactoryPullRequestMergeability(ctx, db, openFactoryVCS(db, deps, factory), pullRequest)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start the conflict fix")
	}
	if result.BlockedReason != pb.FactoryPullRequestMergeability_BLOCKED_REASON_CONFLICTING {
		message := result.Message
		if message == "" {
			message = "The pull request has no merge conflict."
		}
		return nil, factoryErrorToStatus(errors.Join(errFactoryPullRequestNotMergeable, errors.New(message)), "failed to start the conflict fix")
	}

	events, err := startFactoryPullRequestConflictFix(db, factory, pullRequest, true)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start the conflict fix")
	}

	for i := range events {
		if err := messages.PublishCanvasEventCreatedMessage(&events[i]); err != nil {
			log.WithError(err).Warnf("Failed to publish conflict fix event %s", events[i].ID)
		}
	}
	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		pullRequest.WorkOrderID.String(),
		factoryevents.EventTypeOrderPullRequestUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", pullRequest.WorkOrderID)
	}

	serialized, err := serializeFactoryPullRequests(ctx, db, []models.FactoryPullRequest{*pullRequest}, nil)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start the conflict fix")
	}
	if len(serialized) == 0 {
		return nil, factoryErrorToStatus(models.ErrFactoryPullRequestNotFound, "failed to start the conflict fix")
	}
	return &pb.FixFactoryPullRequestConflictsResponse{PullRequest: serialized[0]}, nil
}

func startFactoryPullRequestConflictFix(
	db *gorm.DB,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	manual bool,
) ([]models.CanvasEvent, error) {
	var events []models.CanvasEvent
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := assertNoActiveAutomationLocked(tx, factory, pullRequest); err != nil {
			return errors.Join(errFactoryPullRequestNotMergeable, errors.New(mergeBlockedActiveRun))
		}

		repairNode, handler, settings, err := checksConflictRepair(tx, factory, pullRequest.Repository)
		if err != nil {
			return err
		}
		if !manual && !conflictFixWindowOpen(pullRequest.CreatedAt, settings.MergeConflictHours, time.Now()) {
			return errConflictFixSkipped
		}

		if err := contexts.NewEventContext(tx, &repairNode, nil, func(created []models.CanvasEvent) {
			events = append(events, created...)
		}).Emit(factoryPullRequestConflictEventType, conflictFixEventData(pullRequest)); err != nil {
			return err
		}
		if len(events) == 0 {
			return errors.New("conflict fix did not create an event")
		}

		run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(tx, &events[len(events)-1])
		if err != nil {
			return err
		}

		handlerID := handler.ID
		activity, err := pullRequest.CreateActivity(tx, models.FactoryPullRequestActivityParams{
			RunID:             run.ID,
			Title:             factoryPullRequestConflictTitle,
			Description:       factoryPullRequestConflictBody,
			Access:            models.FactoryPullRequestAccessExclusive,
			FeedbackHandlerID: &handlerID,
		})
		if err != nil {
			return err
		}
		switch activity.Outcome {
		case models.FactoryPullRequestActivityOutcomeReady:
			return nil
		case models.FactoryPullRequestActivityOutcomeLimitReached:
			if manual {
				return errFactoryPullRequestConflictFixesPaused
			}
			return errConflictFixSkipped
		default:
			return errors.Join(errFactoryPullRequestNotMergeable, errors.New(mergeBlockedActiveRun))
		}
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func maybeStartFactoryPullRequestConflictFix(
	db *gorm.DB,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
) {
	if factory == nil || pullRequest == nil || pullRequest.MergeBlockedReason != "CONFLICTING" {
		return
	}
	organization, err := models.FindOrganizationByIDInTransaction(db, factory.OrganizationID.String())
	if err != nil {
		log.WithError(err).Warnf("factory conflict fix: organization %s not found", factory.OrganizationID)
		return
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryPullRequestMerge) {
		return
	}

	events, err := startFactoryPullRequestConflictFix(db, factory, pullRequest, false)
	if err != nil {
		if conflictFixStartQuiet(err) {
			return
		}
		log.WithError(err).Warnf("factory conflict fix: failed to start pull request %s", pullRequest.ID)
		return
	}
	for i := range events {
		if err := messages.PublishCanvasEventCreatedMessage(&events[i]); err != nil {
			log.WithError(err).Warnf("Failed to publish conflict fix event %s", events[i].ID)
		}
	}
	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		pullRequest.WorkOrderID.String(),
		factoryevents.EventTypeOrderPullRequestUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", pullRequest.WorkOrderID)
	}
}

func conflictFixStartQuiet(err error) bool {
	return errors.Is(err, errConflictFixSkipped) ||
		errors.Is(err, errFactoryPullRequestConflictHandlerMissing) ||
		errors.Is(err, errFactoryPullRequestConflictFixDisabled) ||
		errors.Is(err, errFactoryPullRequestNotMergeable)
}

func conflictFixWindowOpen(createdAt time.Time, hours int, now time.Time) bool {
	if hours <= 0 {
		hours = prFeedbackDefaultMergeConflictHours
	}
	return now.Before(createdAt.Add(time.Duration(hours) * time.Hour))
}

func checksConflictRepair(
	tx *gorm.DB,
	factory *models.Factory,
	repository string,
) (models.CanvasNode, models.FactoryPRFeedbackHandler, prFeedbackSettings, error) {
	handlers, err := factory.ListPRFeedbackHandlers(tx)
	if err != nil {
		return models.CanvasNode{}, models.FactoryPRFeedbackHandler{}, prFeedbackSettings{}, err
	}
	for i := range handlers {
		handler := handlers[i]
		if handler.Source != models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
			continue
		}
		nodes, edges, err := models.FindLiveCanvasSpecInTransaction(tx, handler.CanvasID)
		if err != nil {
			return models.CanvasNode{}, models.FactoryPRFeedbackHandler{}, prFeedbackSettings{}, err
		}
		spec := models.LiveCanvasSpec{Nodes: nodes, Edges: edges}
		graph := resolvePRFeedbackGraph(spec)
		if graph.StartRepairNodeID == "" {
			continue
		}
		settings := prFeedbackSettingsFromGraph(graph, spec)
		if !strings.EqualFold(settings.Repository, strings.TrimSpace(repository)) {
			continue
		}
		if !settings.FixMergeConflicts {
			return models.CanvasNode{}, models.FactoryPRFeedbackHandler{}, prFeedbackSettings{}, errFactoryPullRequestConflictFixDisabled
		}
		node, err := models.FindCanvasNode(tx, handler.CanvasID, graph.StartRepairNodeID)
		if err != nil {
			return models.CanvasNode{}, models.FactoryPRFeedbackHandler{}, prFeedbackSettings{}, err
		}
		return *node, handler, settings, nil
	}
	return models.CanvasNode{}, models.FactoryPRFeedbackHandler{}, prFeedbackSettings{}, errFactoryPullRequestConflictHandlerMissing
}

func conflictFixEventData(pullRequest *models.FactoryPullRequest) map[string]any {
	repository := map[string]any{"full_name": pullRequest.Repository}
	return map[string]any{
		"conflict_fix": true,
		"repository":   repository,
		"pull_request": map[string]any{
			"number":   pullRequest.Number,
			"html_url": pullRequest.URL,
			"head":     map[string]any{"repo": repository},
		},
		"pullrequest": map[string]any{
			"id": pullRequest.Number,
			"links": map[string]any{
				"html": map[string]any{"href": pullRequest.URL},
			},
		},
	}
}
