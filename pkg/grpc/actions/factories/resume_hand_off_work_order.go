package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to load pull request")
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
