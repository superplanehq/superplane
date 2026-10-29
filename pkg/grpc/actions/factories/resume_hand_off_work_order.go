package factories

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func ResumeHandOffWorkOrder(ctx context.Context, deps IntakeDependencies, organizationID string, req *pb.ResumeHandOffWorkOrderRequest) (*pb.ResumeHandOffWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	actor, err := codexCallerID(ctx)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load workspace")
	}
	order, err := findWorkOrder(db, factory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load task")
	}
	if order.PendingHandOff == nil {
		return nil, grpcFailedPrecondition("This task has no pending handoff.")
	}
	intent := order.PendingHandOff.Data()
	var pullRequest *handOffPullRequest
	if intent.Column == HandOffColumnVerify {
		pullRequest, err = loadHandOffPullRequest(ctx, db, deps, factory, intent.PullRequestURL)
		if isHandOffPullRequestGone(err) {
			return discardStaleHandOff(ctx, factory, order, intent)
		}
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to load pull request")
		}
		if !strings.EqualFold(strings.TrimSpace(pullRequest.State), models.FactoryPullRequestStateOpen) {
			return discardStaleHandOff(ctx, factory, order, intent)
		}
	}
	var dispatched *workOrderLineDispatchResult
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := order.LockForUpdate(tx); err != nil {
			return err
		}
		if order.PendingHandOff == nil || !order.IsDispatchable() || order.PendingHandOff.Data() != intent {
			return grpcFailedPrecondition("This handoff changed. Reload the task and try again.")
		}
		if err := order.RequireExecutionOwner(tx, false); err != nil {
			return err
		}
		line, err := factory.FindLine(tx, intent.LineID)
		if err != nil {
			return err
		}
		stages, err := classifyHandOffStages(tx, factory, line)
		if err != nil {
			return err
		}
		index := 0
		switch intent.Column {
		case HandOffColumnImplement:
			index, err = resolveImplementationStageIndex(stages)
		case HandOffColumnVerify:
			index, err = resolveVerifyStageIndex(stages)
		default:
			return invalidArgument("The pending handoff stage is invalid.")
		}
		if err != nil {
			return err
		}
		if err := order.SetPendingHandOff(tx, nil); err != nil {
			return err
		}
		if intent.Column == HandOffColumnImplement {
			dispatched, err = dispatchWorkOrderOnLineTx(tx, factory, order.ID, line, &actor, index, false, req.GetModel(), req.GetThinkingLevel())
			if err == nil {
				order = dispatched.order
			}
			return err
		}
		if err := validateHandOffOverrides(tx, factory, order, line, req.GetModel(), req.GetThinkingLevel()); err != nil {
			return err
		}
		normalizedThinking, err := runner.NormalizeDispatchThinkingLevel(req.GetThinkingLevel())
		if err != nil {
			return err
		}
		return recordVerifyHandOff(tx, factory, order, line, &actor, pullRequest, index, models.FactoryLineStartOptions{Model: req.GetModel(), ThinkingLevel: normalizedThinking})
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to resume handoff")
	}
	if dispatched != nil {
		publishDispatchedWorkOrder(dispatched.logger, orgID, factory.ID, order, &actor, dispatched.fromState, dispatched.startedSteps)
	} else {
		if _, err := publishVerificationHandOff(ctx, deps, db, factory, order); err != nil {
			return nil, factoryErrorToStatus(err, "failed to refresh pull request")
		}
	}
	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load task")
	}
	return &pb.ResumeHandOffWorkOrderResponse{Order: serialized, Column: intent.Column}, nil
}

// isHandOffPullRequestGone reports whether the provider confirmed that the
// saved pull request no longer exists. Other load errors, such as timeouts or
// rate limits, are temporary and must keep the saved handoff.
func isHandOffPullRequestGone(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errFactoryPullRequestMissing) {
		return true
	}
	var githubErr *github.ErrorResponse
	if errors.As(err, &githubErr) && githubErr.Response != nil {
		return isGoneStatus(githubErr.Response.StatusCode)
	}
	var bitbucketErr *bitbucket.APIError
	if errors.As(err, &bitbucketErr) {
		return isGoneStatus(bitbucketErr.StatusCode)
	}
	return false
}

func isGoneStatus(status int) bool {
	return status == http.StatusNotFound || status == http.StatusGone
}

// discardStaleHandOff clears a saved verify handoff whose pull request is no
// longer open. The task stays in the backlog with its plan, so a member can
// start it on any line.
func discardStaleHandOff(ctx context.Context, factory *models.Factory, order *models.FactoryWorkOrder, intent models.PendingHandOff) (*pb.ResumeHandOffWorkOrderResponse, error) {
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := order.LockForUpdate(tx); err != nil {
			return err
		}
		if order.PendingHandOff == nil || !order.IsDispatchable() || order.PendingHandOff.Data() != intent {
			return grpcFailedPrecondition("This handoff changed. Reload the task and try again.")
		}
		return order.SetPendingHandOff(tx, nil)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to discard handoff")
	}
	if err := messages.PublishFactoryWorkOrderUpdated(factory.ID.String(), order.ID.String(), factoryevents.EventTypeOrderUpdated); err != nil {
		log.WithError(err).Warn("Failed to publish discarded handoff")
	}
	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load task")
	}
	return &pb.ResumeHandOffWorkOrderResponse{Order: serialized, Column: "backlog"}, nil
}

func validateHandOffOverrides(tx *gorm.DB, f *models.Factory, o *models.FactoryWorkOrder, line *models.FactoryLine, model, thinking string) error {
	if strings.TrimSpace(model) != "" {
		allowed, _, err := listLineRunnerModels(tx, f.OrganizationID, f.ID, line.Name, subscriptionOrderOwnerID(tx, o))
		if err != nil {
			return err
		}
		found := false
		for _, id := range allowed {
			if id == strings.TrimSpace(model) {
				found = true
				break
			}
		}
		if !found {
			return invalidArgument("The selected model is unavailable for this task owner.")
		}
	}
	normalized, err := runner.NormalizeDispatchThinkingLevel(thinking)
	if err != nil {
		return invalidArgument(err.Error())
	}
	return validateLineThinkingLevel(tx, line, normalized)
}
