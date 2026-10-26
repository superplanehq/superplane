package factories

import (
	"context"
	"errors"
	"fmt"
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

func RunFactoryCustomVerification(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.RunFactoryCustomVerificationRequest,
) (*pb.RunFactoryCustomVerificationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom verification")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom verification")
	}
	canvas, err := findCustomVerificationCanvas(db, factory, req.GetAutomationId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom verification")
	}
	order, err := findWorkOrder(db, factory, req.GetWorkOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom verification")
	}
	mode, err := customVerificationCanvasMode(db, canvas)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom verification")
	}
	if mode != pb.FactoryCustomVerification_START_MODE_MANUAL {
		return nil, factoryErrorToStatus(invalidArgument("automation does not use manual start"), "failed to run custom verification")
	}

	event, err := startCustomVerification(ctx, db, deps, factory, canvas, order, false)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run custom verification")
	}
	if event == nil {
		return nil, factoryErrorToStatus(errors.New("custom verification did not create a run"), "failed to run custom verification")
	}
	if err := messages.PublishCanvasEventCreatedMessage(event); err != nil {
		log.WithError(err).Warnf("custom verification: failed to publish event %s", event.ID)
	}
	return &pb.RunFactoryCustomVerificationResponse{
		RunId:   event.RunID.String(),
		EventId: event.ID.String(),
	}, nil
}

// StartAutomaticCustomVerifications starts all automatic Verify automations
// for the current pull request head. Webhook starts use the same dedup identity.
func StartAutomaticCustomVerifications(
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
		if canvas.LiveVersionID == nil {
			continue
		}
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
		if err != nil || models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomVerifyID {
			continue
		}
		mode, err := customVerificationCanvasModeFromVersion(version)
		if err != nil || mode != pb.FactoryCustomVerification_START_MODE_AUTOMATIC {
			continue
		}
		event, err := startCustomVerification(ctx, db, deps, factory, canvas, order, true)
		if err != nil {
			return err
		}
		if event != nil {
			if err := messages.PublishCanvasEventCreatedMessage(event); err != nil {
				log.WithError(err).Warnf("custom verification: failed to publish event %s", event.ID)
			}
		}
	}
	return nil
}

func startCustomVerification(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	canvas *models.Canvas,
	order *models.FactoryWorkOrder,
	automatic bool,
) (*models.CanvasEvent, error) {
	if err := requireWorkOrderInVerify(db, order); err != nil {
		return nil, err
	}
	pullRequest, err := latestTrackedPullRequest(db, order.ID)
	if err != nil {
		return nil, err
	}
	if pullRequest == nil {
		return nil, grpcFailedPrecondition("The task has no pull request.")
	}
	if pullRequest.State != models.FactoryPullRequestStateOpen {
		return nil, grpcFailedPrecondition("The pull request is not open.")
	}

	payload, revision, payloadType, err := customVerificationPullRequestPayload(ctx, db, deps, factory, order, pullRequest)
	if err != nil {
		return nil, err
	}
	if err := ensureCustomVerificationCheckoutEnvironment(db, factory, canvas); err != nil {
		return nil, err
	}
	trigger, err := models.FindCanvasNode(db, canvas.ID, customVerificationTriggerID)
	if err != nil {
		return nil, err
	}
	var created []models.CanvasEvent
	events := contexts.NewEventContext(db, trigger, nil, func(events []models.CanvasEvent) {
		created = append(created, events...)
	})
	if automatic {
		err = events.EmitDeduplicated(payloadType, payload, pullRequest.ID.String()+"|"+revision)
	} else {
		err = events.Emit("factory.customVerification", payload)
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
	if err := models.MaybeLinkCustomVerificationRun(db, canvas, event, run); err != nil {
		return nil, err
	}
	return event, nil
}

func customVerificationPullRequestPayload(
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
		return bitbucketCustomVerificationEventData(pullRequest, order, current.SourceBr, revision), revision, "bitbucket.pullRequest", nil
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

func bitbucketCustomVerificationEventData(
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

func customVerificationCanvasMode(tx *gorm.DB, canvas *models.Canvas) (pb.FactoryCustomVerification_StartMode, error) {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return pb.FactoryCustomVerification_START_MODE_UNSPECIFIED, err
	}
	return customVerificationCanvasModeFromVersion(version)
}

func customVerificationCanvasModeFromVersion(version *models.CanvasVersion) (pb.FactoryCustomVerification_StartMode, error) {
	for i := range version.Nodes {
		node := version.Nodes[i]
		if node.ID != customVerificationTriggerID {
			continue
		}
		template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		mode, _ := template["startMode"].(string)
		return customVerificationStartMode(mode), nil
	}
	return pb.FactoryCustomVerification_START_MODE_UNSPECIFIED, errors.New("custom verification trigger is missing")
}

func grpcFailedPrecondition(message string) error {
	return grpcerrors.FailedPrecondition(errors.New(message), message)
}
