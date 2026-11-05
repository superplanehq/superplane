package factories

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func RunFactoryCustomAutomation(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.RunFactoryCustomAutomationRequest,
) (*pb.RunFactoryCustomAutomationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom automation")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom automation")
	}
	canvas, err := findCustomAutomationCanvas(db, factory, req.GetAutomationId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom automation")
	}
	order, err := findWorkOrder(db, factory, req.GetWorkOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom automation")
	}
	mode, err := customAutomationCanvasMode(db, canvas)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom automation")
	}
	if mode != pb.FactoryCustomAutomation_START_MODE_MANUAL {
		return nil, factoryErrorToStatus(invalidArgument("automation does not use manual start"), "failed to run custom automation")
	}

	events, err := startCustomAutomation(ctx, db, deps, factory, canvas, order, false)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom automation")
	}
	if len(events) == 0 {
		return nil, factoryErrorToStatus(errors.New("custom automation did not create a run"), "failed to run custom automation")
	}
	for _, event := range events {
		if err := messages.PublishCanvasEventCreatedMessage(event); err != nil {
			log.WithError(err).Warnf("custom automation: failed to publish event %s", event.ID)
		}
	}
	return &pb.RunFactoryCustomAutomationResponse{
		RunId:   events[0].RunID.String(),
		EventId: events[0].ID.String(),
	}, nil
}

// StartAutomaticCustomAutomations starts all automatic Verify automations
// for the current pull request head. Webhook starts use the same dedup identity.
func StartAutomaticCustomAutomations(
	ctx context.Context,
	deps IntakeDependencies,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
) error {
	if factory == nil || order == nil {
		return nil
	}
	db := database.DB(ctx)
	if err := requireWorkOrderInVerify(db, order); err != nil {
		if code, _, ok := grpcerrors.HandlerStatus(err); ok && code == codes.FailedPrecondition {
			return nil
		}
		return err
	}
	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return err
	}
	for i := range canvases {
		canvas := &canvases[i]
		if canvas.LiveVersionID == nil || canvas.ColumnKey == nil || *canvas.ColumnKey != models.CanvasColumnKeyVerify {
			continue
		}
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
		if err != nil || models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomAutomationID {
			continue
		}
		mode, err := customAutomationCanvasModeFromVersion(version)
		if err != nil || mode != pb.FactoryCustomAutomation_START_MODE_AUTOMATIC {
			continue
		}
		events, err := startCustomAutomation(ctx, db, deps, factory, canvas, order, true)
		if err != nil {
			return err
		}
		for _, event := range events {
			if err := messages.PublishCanvasEventCreatedMessage(event); err != nil {
				log.WithError(err).Warnf("custom automation: failed to publish event %s", event.ID)
			}
		}
	}
	return nil
}

func startCustomAutomation(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	canvas *models.Canvas,
	order *models.FactoryWorkOrder,
	automatic bool,
) ([]*models.CanvasEvent, error) {
	if canvas.ColumnKey != nil && *canvas.ColumnKey == models.CanvasColumnKeyBacklog {
		event, err := contexts.StartBacklogCustomAutomation(db, factory, canvas, order, automatic)
		if err != nil || event == nil {
			return nil, err
		}
		return []*models.CanvasEvent{event}, nil
	}
	if err := requireWorkOrderInVerify(db, order); err != nil {
		return nil, err
	}
	pullRequests, err := trackedPullRequests(db, order.ID)
	if err != nil {
		return nil, err
	}
	if len(pullRequests) == 0 {
		return nil, grpcFailedPrecondition("The task has no pull request.")
	}
	openPullRequests := slices.DeleteFunc(slices.Clone(pullRequests), func(pullRequest models.FactoryPullRequest) bool {
		return pullRequest.State != models.FactoryPullRequestStateOpen
	})
	if len(openPullRequests) == 0 {
		return nil, grpcFailedPrecondition("The pull request is not open.")
	}
	trigger, err := models.FindCanvasNode(db, canvas.ID, customAutomationTriggerID)
	if err != nil {
		return nil, err
	}

	openPullRequests = slices.DeleteFunc(openPullRequests, func(pullRequest models.FactoryPullRequest) bool {
		return !customAutomationCoversRepository(trigger, pullRequest.Repository)
	})
	if len(openPullRequests) == 0 {
		if automatic {
			return nil, nil
		}
		return nil, grpcFailedPrecondition("The task has no open pull request in a repository of this automation.")
	}

	// A task can have one pull request in each workspace repository. Each
	// pull request gets its own run, like merge confidence. Load every pull
	// request before the first event is saved, so a failed GitHub request
	// does not leave runs that a retry starts again.
	starts := make([]customAutomationStart, 0, len(openPullRequests))
	for i := range openPullRequests {
		payload, revision, payloadType, err := customAutomationPullRequestPayload(ctx, db, deps, factory, order, &openPullRequests[i])
		if err != nil {
			return nil, err
		}
		starts = append(starts, customAutomationStart{
			dedupKey: openPullRequests[i].ID.String() + "|" + revision, payloadType: payloadType, payload: payload,
		})
	}
	events := make([]*models.CanvasEvent, 0, len(starts))
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, start := range starts {
			event, err := emitCustomAutomationStart(tx, canvas, trigger, start, automatic)
			if err != nil {
				return err
			}
			if event != nil {
				events = append(events, event)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

type customAutomationStart struct {
	dedupKey    string
	payloadType string
	payload     map[string]any
}

// customAutomationCoversRepository reports whether the automation checks out
// the repository. An automation saved before workspace scope checks out its
// configured repository only.
func customAutomationCoversRepository(trigger *models.CanvasNode, repository string) bool {
	configured, _ := trigger.Configuration.Data()["repository"].(string)
	return models.IsWorkspaceRepositoriesExpression(configured) || strings.EqualFold(strings.TrimSpace(configured), strings.TrimSpace(repository))
}

func emitCustomAutomationStart(
	db *gorm.DB,
	canvas *models.Canvas,
	trigger *models.CanvasNode,
	start customAutomationStart,
	automatic bool,
) (*models.CanvasEvent, error) {
	var created []models.CanvasEvent
	events := contexts.NewEventContext(db, trigger, nil, func(events []models.CanvasEvent) {
		created = append(created, events...)
	})
	var err error
	if automatic {
		err = events.EmitDeduplicated(start.payloadType, start.payload, start.dedupKey)
	} else {
		err = events.Emit("factory.customAutomation", start.payload)
	}
	if err != nil {
		return nil, err
	}
	if len(created) == 0 {
		if automatic {
			return nil, nil
		}
		return nil, grpcFailedPrecondition("SuperPlane could not start the automation.")
	}
	event := &created[0]
	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, event.RunID)
	if err != nil {
		return nil, err
	}
	if err := models.MaybeLinkCustomAutomationRun(db, canvas, event, run); err != nil {
		return nil, err
	}
	return event, nil
}

func customAutomationPullRequestPayload(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	pullRequest *models.FactoryPullRequest,
) (map[string]any, string, string, error) {
	if pullRequest.Provider == models.FactoryPullRequestProviderBitbucket {
		client, err := newFactoryBitbucketAPI(db, deps, factory)
		if err != nil {
			return nil, "", "", err
		}
		current, err := client.GetPullRequest(pullRequest.Repository, pullRequest.Number)
		if err != nil {
			return nil, "", "", fmt.Errorf("get current Bitbucket pull request: %w", err)
		}
		if current == nil {
			return nil, "", "", errFactoryPullRequestMissing
		}
		revision := strings.TrimSpace(current.SourceHash)
		if revision == "" {
			return nil, "", "", grpcFailedPrecondition("The pull request has no head revision.")
		}
		return bitbucketCustomAutomationEventData(pullRequest, order, current.SourceBr, revision), revision, "bitbucket.pullRequest", nil
	}

	client, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		return nil, "", "", err
	}
	current, _, err := client.GetPullRequest(ctx, pullRequest.Repository, int(pullRequest.Number))
	if err != nil {
		return nil, "", "", fmt.Errorf("get current GitHub pull request: %w", err)
	}
	if current == nil {
		return nil, "", "", errFactoryPullRequestMissing
	}
	revision := strings.TrimSpace(current.GetHead().GetSHA())
	if revision == "" {
		return nil, "", "", grpcFailedPrecondition("The pull request has no head revision.")
	}
	payload, err := pullRequestChecksEventData(pullRequest, order, current)
	return payload, revision, "github.pullRequest", err
}

func bitbucketCustomAutomationEventData(
	pullRequest *models.FactoryPullRequest,
	order *models.FactoryWorkOrder,
	headBranch, revision string,
) map[string]any {
	return map[string]any{
		"action": "updated",
		"repository": map[string]any{
			"full_name": pullRequest.Repository,
		},
		"pullrequest": map[string]any{
			"id":    pullRequest.Number,
			"title": pullRequest.Title,
			"links": map[string]any{"html": map[string]any{"href": pullRequest.URL}},
			"source": map[string]any{
				"branch": map[string]any{"name": headBranch},
				"commit": map[string]any{"hash": revision},
			},
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

func requireWorkOrderInVerify(tx *gorm.DB, order *models.FactoryWorkOrder) error {
	if order == nil {
		return grpcFailedPrecondition("The task is not in Verify.")
	}
	inVerify, err := models.WorkOrderIsWaitingAfterPassedStage(tx, order.ID)
	if err != nil {
		return err
	}
	if !inVerify {
		return grpcFailedPrecondition("The task is not in Verify.")
	}
	return nil
}

func customAutomationCanvasMode(tx *gorm.DB, canvas *models.Canvas) (pb.FactoryCustomAutomation_StartMode, error) {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return pb.FactoryCustomAutomation_START_MODE_UNSPECIFIED, err
	}
	return customAutomationCanvasModeFromVersion(version)
}

func customAutomationCanvasModeFromVersion(version *models.CanvasVersion) (pb.FactoryCustomAutomation_StartMode, error) {
	for i := range version.Nodes {
		node := version.Nodes[i]
		if node.ID != customAutomationTriggerID && node.ID != customAutomationTaskTriggerID {
			continue
		}
		template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		mode, _ := template["startMode"].(string)
		return customAutomationStartMode(mode), nil
	}
	return pb.FactoryCustomAutomation_START_MODE_UNSPECIFIED, errors.New("custom automation trigger is missing")
}

func grpcFailedPrecondition(message string) error {
	return grpcerrors.FailedPrecondition(errors.New(message), message)
}
