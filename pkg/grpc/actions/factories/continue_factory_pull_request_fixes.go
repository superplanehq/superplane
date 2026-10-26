package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

const continueFixesInternalMessage = "failed to continue factory pull request fixes"

// ContinueFactoryPullRequestFixes resumes automatic check fixes that the
// attempt limit paused. It starts the checks automation again with the stored
// trigger event, refreshed to the current pull request head, and resets the
// attempt count for this pull request.
func ContinueFactoryPullRequestFixes(
	ctx context.Context,
	organizationID string,
	req *pb.ContinueFactoryPullRequestFixesRequest,
) (*pb.ContinueFactoryPullRequestFixesResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, continueFixesInternalMessage)
	}
	runID, err := uuid.Parse(req.GetRunId())
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid run id"), continueFixesInternalMessage)
	}

	var factory *models.Factory
	var pullRequest *models.FactoryPullRequest
	var reemitted *models.CanvasEvent
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		loadedFactory, loadedPullRequest, err := loadFactoryPullRequest(tx, orgID, req.GetFactoryId(), req.GetPrId())
		if err != nil {
			return err
		}
		factory = loadedFactory
		pullRequest = loadedPullRequest

		event, err := continueFixes(tx, factory, pullRequest, runID)
		if err != nil {
			return err
		}
		reemitted = event
		return nil
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, continueFixesInternalMessage)
	}

	publishContinuedFixes(factory, pullRequest, reemitted)
	return &pb.ContinueFactoryPullRequestFixesResponse{EventId: reemitted.ID.String()}, nil
}

func loadFactoryPullRequest(
	tx *gorm.DB,
	orgID uuid.UUID,
	factoryID string,
	prID string,
) (*models.Factory, *models.FactoryPullRequest, error) {
	parsedPRID, err := parsePullRequestID(prID)
	if err != nil {
		return nil, nil, err
	}
	factory, err := findFactory(tx, orgID, factoryID)
	if err != nil {
		return nil, nil, err
	}
	pullRequest, err := factory.FindPullRequest(tx, models.FactoryPullRequestLookup{ID: parsedPRID})
	if err != nil {
		return nil, nil, err
	}
	return factory, pullRequest, nil
}

func continueFixes(
	tx *gorm.DB,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	runID uuid.UUID,
) (*models.CanvasEvent, error) {
	// Lock the pull request first. Two clicks then run one after the other.
	if err := pullRequest.LockForUpdate(tx); err != nil {
		return nil, err
	}
	if pullRequest.State == models.FactoryPullRequestStateClosed || pullRequest.State == models.FactoryPullRequestStateMerged {
		return nil, errFactoryPullRequestNotOpen
	}

	activity, err := models.FindPullRequestActivityByRunID(tx, runID)
	if err != nil {
		return nil, err
	}
	if activity.PullRequestID != pullRequest.ID {
		return nil, models.ErrFactoryPullRequestActivityNotFound
	}

	handler, err := checksHandlerForActivity(tx, factory, activity)
	if err != nil {
		return nil, err
	}

	rootEvent, err := checksTriggerRootEvent(tx, handler, activity)
	if err != nil {
		return nil, err
	}

	if err := activity.ContinueFixes(tx); err != nil {
		return nil, err
	}

	data, err := dataWithHeadSHA(rootEvent.Data.Data(), currentRevisionSHA(tx, pullRequest))
	if err != nil {
		return nil, err
	}

	now := time.Now()
	reemitted := &models.CanvasEvent{
		WorkflowID: rootEvent.WorkflowID,
		NodeID:     rootEvent.NodeID,
		Channel:    rootEvent.Channel,
		Data:       models.NewJSONValue(data),
		State:      models.CanvasEventStatePending,
		CreatedAt:  &now,
		CustomName: rootEvent.CustomName,
	}
	if err := tx.Create(reemitted).Error; err != nil {
		return nil, err
	}
	return reemitted, nil
}

func checksHandlerForActivity(
	tx *gorm.DB,
	factory *models.Factory,
	activity *models.FactoryPullRequestRun,
) (*models.FactoryPRFeedbackHandler, error) {
	if activity.FeedbackHandlerID == nil {
		return nil, models.ErrFactoryPullRequestFixesNotPaused
	}
	handler, err := models.FindPRFeedbackHandlerByID(tx, *activity.FeedbackHandlerID)
	if err != nil {
		return nil, err
	}
	if handler.FactoryID != factory.ID || handler.OrganizationID != factory.OrganizationID {
		return nil, models.ErrFactoryPRFeedbackHandlerNotFound
	}
	if handler.Source != models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
		return nil, models.ErrFactoryPullRequestFixesNotPaused
	}
	return handler, nil
}

func checksTriggerRootEvent(
	tx *gorm.DB,
	handler *models.FactoryPRFeedbackHandler,
	activity *models.FactoryPullRequestRun,
) (*models.CanvasEvent, error) {
	events, err := models.ListRootEventsForRuns(tx, handler.CanvasID, []uuid.UUID{activity.RunID})
	if err != nil {
		return nil, err
	}
	event, ok := events[activity.RunID]
	if !ok || event.NodeID != prFeedbackPullRequestTriggerNodeID {
		return nil, grpcerrors.FailedPrecondition(
			errors.New("checks trigger event not found"),
			"SuperPlane cannot find the pull request event that started this automation.",
		)
	}
	return &event, nil
}

func currentRevisionSHA(tx *gorm.DB, pullRequest *models.FactoryPullRequest) string {
	if pullRequest.CurrentRevisionID == nil {
		return ""
	}
	revision, err := models.FindPullRequestRevision(tx, *pullRequest.CurrentRevisionID)
	if err != nil {
		return ""
	}
	return revision.SHA
}

// dataWithHeadSHA copies the trigger event payload and sets the pull request
// head SHA. The checks automation reads the SHA from root().data.pull_request.head.sha.
// An empty SHA keeps the stored value.
func dataWithHeadSHA(payload any, sha string) (any, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var copied map[string]any
	if err := models.UnmarshalJSONValue(encoded, &copied); err != nil {
		return nil, fmt.Errorf("trigger event payload is not an object: %w", err)
	}
	if sha == "" {
		return copied, nil
	}

	node := copied
	for _, key := range []string{"data", "pull_request", "head"} {
		next, ok := node[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			node[key] = next
		}
		node = next
	}
	node["sha"] = sha
	return copied, nil
}

func publishContinuedFixes(factory *models.Factory, pullRequest *models.FactoryPullRequest, event *models.CanvasEvent) {
	err := messages.NewCanvasEventCreatedMessage(event.WorkflowID.String(), factory.OrganizationID.String(), event).Publish()
	if err != nil {
		log.WithError(err).Warnf("Failed to publish re-emitted pull request event %s", event.ID)
	}
	err = messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		pullRequest.WorkOrderID.String(),
		factoryevents.EventTypeOrderPullRequestUpdated,
	)
	if err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", pullRequest.WorkOrderID)
	}
}
