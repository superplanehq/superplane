package factories

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/llmcatalog"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func DispatchWorkOrder(ctx context.Context, organizationID string, req *pb.DispatchWorkOrderRequest) (*pb.DispatchWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	db := database.DB(ctx)
	resolvedFactory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	resolvedOrder, err := findWorkOrder(db, resolvedFactory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}
	orderID := resolvedOrder.ID
	hasPriorDispatch, priorErr := resolvedOrder.HasLineDispatch(db)
	runKind := "initial"
	if hasPriorDispatch {
		runKind = "rerun"
	}

	lineName := strings.TrimSpace(req.GetLineName())
	if lineName == "" {
		return nil, factoryErrorToStatus(invalidArgument("line_name is required"), "failed to dispatch work order")
	}

	callerID, err := codexCallerID(ctx)
	if err != nil {
		return nil, err
	}
	actor := &callerID

	line, err := resolvedFactory.FindLineByName(db, lineName)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	factory, order, err := DispatchWorkOrderOnLine(
		db,
		resolvedFactory,
		orderID,
		line,
		actor,
		int(req.GetStartStepIndex()),
		req.GetReplaceActive(),
		req.GetModel(),
		req.GetThinkingLevel(),
	)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}
	dispatchID := productanalytics.StableEventID(productanalytics.EventFactoryTaskStarted, order.ID.String(), order.UpdatedAt.UTC().String())
	if dispatch, findErr := order.FindActiveLineDispatch(db); findErr == nil {
		dispatchID = productanalytics.StableEventID(productanalytics.EventFactoryTaskStarted, dispatch.ID.String())
	}
	if priorErr != nil {
		log.WithError(priorErr).Warn("Skipped factory task started analytics")
	} else {
		captureFactoryProductEvent(
			ctx,
			db,
			factory,
			order.ID,
			productanalytics.EventFactoryTaskStarted,
			dispatchID,
			productanalytics.Properties{"run_kind": runKind},
		)
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	return &pb.DispatchWorkOrderResponse{
		Order: serialized,
	}, nil
}

type workOrderLineDispatchResult struct {
	factory      *models.Factory
	order        *models.FactoryWorkOrder
	startedSteps []*models.FactoryLineStepResult
	planningStop *models.RunCancellationResult
	fromState    string
	logger       *log.Entry
}

func DispatchWorkOrderOnLine(
	db *gorm.DB,
	factory *models.Factory,
	orderID uuid.UUID,
	line *models.FactoryLine,
	actor *uuid.UUID,
	startIndex int,
	replaceActive bool,
	model string,
	thinkingLevel string,
) (*models.Factory, *models.FactoryWorkOrder, error) {
	var result *workOrderLineDispatchResult
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = dispatchWorkOrderOnLineTx(tx, factory, orderID, line, actor, startIndex, replaceActive, model, thinkingLevel)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	if result.planningStop != nil && result.planningStop.Run != nil {
		messages.PublishRunCancellationDrain(result.planningStop.Run.WorkflowID, result.planningStop.Drain)
		if result.planningStop.NewlyCancelling {
			if err := messages.NewCanvasRunMessage(result.planningStop.Run.WorkflowID.String(), result.planningStop.Run.ID.String()).Publish(); err != nil {
				result.logger.WithError(err).Warn("Failed to publish planning run completion")
			}
		}
	}

	publishDispatchedWorkOrder(
		result.logger,
		result.factory.OrganizationID,
		result.factory.ID,
		result.order,
		actor,
		result.fromState,
		result.startedSteps,
	)
	return result.factory, result.order, nil
}

func dispatchWorkOrderOnLineTx(
	tx *gorm.DB,
	factory *models.Factory,
	orderID uuid.UUID,
	line *models.FactoryLine,
	actor *uuid.UUID,
	startIndex int,
	replaceActive bool,
	model string,
	thinkingLevel string,
) (*workOrderLineDispatchResult, error) {
	if factory == nil || line == nil || orderID == uuid.Nil {
		return nil, invalidArgument("work order dispatch is incomplete")
	}

	orgID := factory.OrganizationID
	factoryID := factory.ID
	lineID := line.ID

	f, err := models.FindFactory(tx, orgID, factoryID)
	if err != nil {
		return nil, err
	}
	factory = f

	order, err := factory.FindWorkOrder(tx, orderID)
	if err != nil {
		return nil, err
	}
	if err := order.LockForUpdate(tx); err != nil {
		return nil, err
	}

	logger := logging.WithWorkOrder(logging.ForFactory(*factory), *order).
		WithField("organization_id", factory.OrganizationID)
	if order.PendingHandOff != nil {
		return nil, grpcFailedPrecondition("Use Start to resume this pending handoff.")
	}
	if !order.IsDispatchable() {
		return nil, models.ErrFactoryWorkOrderNotDispatchable
	}

	planningStop, err := order.CompletePlanningForStart(tx, actor)
	if err != nil {
		return nil, err
	}
	if err := assignWorkOrderStarter(tx, factory, order, actor); err != nil {
		return nil, err
	}
	if err := order.RequireExecutionOwner(tx, false); err != nil {
		return nil, err
	}

	currentLine, err := factory.FindLine(tx, lineID)
	if err != nil {
		return nil, err
	}

	if len(currentLine.Steps) == 0 {
		return nil, models.ErrFactoryLineHasNoSteps
	}

	model = strings.TrimSpace(model)
	normalizedThinking, err := runner.NormalizeDispatchThinkingLevel(thinkingLevel)
	if err != nil {
		return nil, invalidArgument(err.Error())
	}
	if err := validateLineDispatchModel(tx, orgID, factoryID, currentLine, model, normalizedThinking, subscriptionOrderOwnerID(tx, order)); err != nil {
		return nil, err
	}
	if err := requireSavedBranchForImplementation(tx, factory, order, currentLine, startIndex); err != nil {
		return nil, err
	}
	if resolvedWorkOrderDefaultBranch(order, factory.OnboardingConfigValue()) != "" {
		if err := order.ClearStatusNote(tx, missingImplementBranchNoteKey); err != nil {
			return nil, err
		}
	}

	fromState := order.State
	if err := fillMissingWorkOrderTitle(tx, order); err != nil {
		return nil, err
	}
	if err := order.TransitionOnDispatch(tx, actor); err != nil {
		return nil, err
	}
	if _, err := order.ClearAutoStart(tx); err != nil {
		return nil, err
	}

	var startedSteps []*models.FactoryLineStepResult
	if replaceActive && startIndex > 0 {
		_, started, err := order.RetryLineStep(tx, currentLine, startIndex, models.FactoryLineStartOptions{Model: model, ThinkingLevel: normalizedThinking})
		if err != nil {
			return nil, err
		}
		startedSteps = started
		return &workOrderLineDispatchResult{
			factory:      factory,
			order:        order,
			startedSteps: startedSteps,
			planningStop: planningStop,
			fromState:    fromState,
			logger:       logger,
		}, nil
	}

	var abandoned []*models.FactoryLineStepResult
	_, err = order.FindActiveLineDispatch(tx)
	if err == nil {
		if !replaceActive {
			return nil, models.ErrFactoryWorkOrderLineDispatchActive
		}
		abandoned, err = order.AbandonActiveLineDispatch(tx)
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	_, result, err := currentLine.DispatchFromWithModel(tx, order, startIndex, model, normalizedThinking)
	if err != nil {
		return nil, err
	}

	return &workOrderLineDispatchResult{
		factory:      factory,
		order:        order,
		startedSteps: append(abandoned, result),
		planningStop: planningStop,
		fromState:    fromState,
		logger:       logger,
	}, nil
}

// validateLineDispatchModel checks overrides and each agent's effective model
// and effort before Start, including settings inherited from the agent.
func validateLineDispatchModel(tx *gorm.DB, orgID, factoryID uuid.UUID, line *models.FactoryLine, model, thinking string, ownerID uuid.UUID) error {
	if model == "" {
		return validateDispatchThinkingForRunners(tx, orgID, factoryID, line, model, thinking, ownerID)
	}
	options, _, err := listLineRunnerModelOptions(tx, orgID, factoryID, line.Name, ownerID)
	if err != nil {
		return err
	}
	index := slices.IndexFunc(options, func(option lineRunnerModel) bool { return option.ID == model })
	if index < 0 {
		return invalidArgument("model is not available on this line")
	}
	if dispatchThinkingOverridesLevel(thinking) && !slices.Contains(options[index].ThinkingLevels, thinking) {
		return invalidArgument("thinking level is not available for this model")
	}
	return validateDispatchThinkingForRunners(tx, orgID, factoryID, line, model, thinking, ownerID)
}

// validateDispatchThinkingForRunners checks inherited and explicit efforts
// against the model each agent will run, including agents that retain a model.
func validateDispatchThinkingForRunners(tx *gorm.DB, orgID, factoryID uuid.UUID, line *models.FactoryLine, model, thinking string, ownerID uuid.UUID) error {
	var catalog *llmcatalog.ThinkingCatalog
	for _, step := range line.Steps {
		if step.AppID == uuid.Nil {
			continue
		}
		version, err := models.FindLiveCanvasVersionInTransaction(tx, step.AppID)
		if err != nil {
			return err
		}
		for _, node := range version.Nodes {
			_, isRunner := runnerNodeProvider(node)
			if !isRunner && node.ComponentName() != models.SuperPlaneRunnerComponent {
				continue
			}
			savedThinking, _ := node.Configuration[runner.ThinkingLevelKey].(string)
			effectiveThinking, overridden := runner.OverlayThinkingLevel(savedThinking, thinking)
			effectiveThinking, err = runner.NormalizeCatalogThinkingLevel(effectiveThinking)
			if err != nil && !overridden {
				// Execution decides how to handle a saved value that is not an effort.
				continue
			}
			if err != nil {
				return invalidArgument(err.Error())
			}
			if effectiveThinking == "" {
				continue
			}
			if catalog == nil {
				catalog, err = llmcatalog.LoadThinkingCatalog(tx)
				if err != nil {
					return err
				}
			}
			levels, err := dispatchRunnerThinkingLevels(tx, orgID, factoryID, node, model, ownerID, catalog)
			var unresolved *unresolvedDispatchModelError
			if !overridden && errors.As(err, &unresolved) {
				// Execution resolves the model and reports problems such as a
				// missing ChatGPT connection, as it does for agents without a level.
				continue
			}
			if err != nil {
				return err
			}
			if overridden && !slices.Contains(levels, effectiveThinking) {
				return invalidArgument("thinking level is not available for every agent on this line")
			}
			if !overridden && savedThinkingUnsupported(node, levels, effectiveThinking) {
				return invalidArgument("An agent on this line has a saved thinking level that its model does not support. Choose a thinking level, or change the agent settings.")
			}
		}
	}
	return nil
}

// requireSavedBranchForImplementation stops Implement when checkout would
// discover a branch the later pull request cannot read. Create pull request
// uses the saved task branch. An empty value fails after commits are pushed.
// The check covers every step from the start index, so a line that reaches
// Implement later cannot start without a saved branch. Read-only lines still
// start and follow the remote default.
func requireSavedBranchForImplementation(
	tx *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	line *models.FactoryLine,
	startIndex int,
) error {
	if line == nil || startIndex < 0 || startIndex >= len(line.Steps) {
		return nil
	}
	appIDs := make([]uuid.UUID, 0, len(line.Steps)-startIndex)
	for _, step := range line.Steps[startIndex:] {
		appIDs = append(appIDs, step.AppID)
	}
	blocked, err := implementStepsMissingSavedBranch(tx, factory, order, appIDs)
	if err != nil || !blocked {
		return err
	}
	return invalidArgument(missingImplementBranchMessage)
}

const (
	missingImplementBranchNoteKey = "missing-implement-branch"
	missingImplementBranchMessage = "Set the default branch in Workspace settings → Repository before you start Implement."
)

// NoteMissingImplementBranch tells the task why the line stopped.
func NoteMissingImplementBranch(tx *gorm.DB, order *models.FactoryWorkOrder) error {
	if order == nil {
		return nil
	}
	_, err := order.SetStatusNote(tx, models.FactoryWorkOrderStatusNoteParams{
		Key:      missingImplementBranchNoteKey,
		Headline: "Set the default branch",
		Body:     missingImplementBranchMessage,
	})
	return err
}

// ImplementStepBlockedByMissingBranch reports whether this app is Implement
// and the task has no branch to publish to. A database error is returned
// separately so a caller can retry that lookup.
func ImplementStepBlockedByMissingBranch(
	tx *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	appID uuid.UUID,
) (bool, error) {
	return implementStepsMissingSavedBranch(tx, factory, order, []uuid.UUID{appID})
}

func implementStepsMissingSavedBranch(
	tx *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	appIDs []uuid.UUID,
) (bool, error) {
	if factory == nil {
		return false, nil
	}
	ids := make([]uuid.UUID, 0, len(appIDs))
	for _, appID := range appIDs {
		if appID != uuid.Nil {
			ids = append(ids, appID)
		}
	}
	if len(ids) == 0 {
		return false, nil
	}
	if resolvedWorkOrderDefaultBranch(order, factory.OnboardingConfigValue()) != "" {
		return false, nil
	}
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, ids)
	if err != nil {
		return false, err
	}
	for _, appID := range ids {
		spec, ok := specs[appID]
		if !ok {
			continue
		}
		if models.FactoryAppTemplateID(spec.Nodes) == lineImplementationTemplateID {
			return true, nil
		}
	}
	return false, nil
}

// resolvedWorkOrderDefaultBranch matches resolveOrderRepository. A task
// snapshot is used only when it has both a repository and a branch.
// Otherwise the workspace branch is the branch Implement will publish to.
func resolvedWorkOrderDefaultBranch(order *models.FactoryWorkOrder, config models.FactoryOnboardingConfig) string {
	if order != nil && strings.TrimSpace(derefString(order.Repository)) != "" && strings.TrimSpace(derefString(order.DefaultBranch)) != "" {
		return strings.TrimSpace(derefString(order.DefaultBranch))
	}
	return strings.TrimSpace(config.DefaultBranch)
}

// dispatchThinkingOverridesLevel reports whether a Start thinking value sets
// a level. Empty keeps the agent level and default uses the runner default.
func dispatchThinkingOverridesLevel(thinking string) bool {
	return thinking != "" && thinking != runner.DispatchThinkingDefault
}

// dispatchRunnerThinkingLevels uses the same model acceptance and default
// rules as execution, then reads that model's capabilities instead of CLI limits.
// savedThinkingUnsupported reports whether the model catalog shows that the
// effective model lacks a saved agent level. A ChatGPT subscription catalog
// is complete. For other providers, a model without listed levels can lack
// cached metadata, for example before the first catalog refresh, so the saved
// level reaches the agent as it did before.
func savedThinkingUnsupported(node models.Node, levels []string, thinking string) bool {
	if slices.Contains(levels, thinking) {
		return false
	}
	return len(levels) > 0 || runnerUsesCodexSubscription(node)
}

func runnerUsesCodexSubscription(node models.Node) bool {
	credentials, _ := node.Configuration["credentials"].(map[string]any)
	return node.ComponentName() == runnerCodex && credentials["source"] == "subscription"
}

// unresolvedDispatchModelError reports that Start cannot resolve the model
// that an agent runs. Execution resolves it again and reports the cause.
type unresolvedDispatchModelError struct {
	err error
}

func (e *unresolvedDispatchModelError) Error() string { return e.err.Error() }

func (e *unresolvedDispatchModelError) Unwrap() error { return e.err }

func dispatchRunnerThinkingLevels(tx *gorm.DB, orgID, factoryID uuid.UUID, node models.Node, selected string, ownerID uuid.UUID, catalog *llmcatalog.ThinkingCatalog) ([]string, error) {
	model := storedRunnerModel(node.Configuration)
	if selected != "" {
		accepted, err := models.RunnerAcceptsModel(tx, orgID, &factoryID, node.ComponentName(), node.Configuration, selected, ownerID)
		if err != nil {
			return nil, err
		}
		if accepted {
			model = selected
		}
	}
	if runnerUsesCodexSubscription(node) {
		resolved, err := models.ResolveFactoryCodexSubscriptionModel(tx, orgID, factoryID, ownerID, model, "")
		if err != nil {
			return nil, &unresolvedDispatchModelError{err: err}
		}
		levels, err := lineRunnerThinkingLevels(tx, orgID, factoryID, node, catalog, ownerID)
		if err != nil {
			return nil, err
		}
		return levels(resolved), nil
	}
	if node.ComponentName() == models.SuperPlaneRunnerComponent {
		resolved, err := dispatchSuperPlaneModel(tx, orgID, factoryID, node, model)
		if err != nil {
			return nil, &unresolvedDispatchModelError{err: err}
		}
		return catalog.ThinkingLevels(resolved.Provider, resolved.Model, models.UsageFundingSourceHosted), nil
	}
	if model == "" || models.IsClaudeFamilyAlias(model) {
		ids, err := lineRunnerModelsForNode(tx, orgID, factoryID, node, ownerID)
		if err != nil {
			return nil, err
		}
		if model == "" && len(ids) > 0 {
			model = slices.Min(ids)
		}
		model = models.ConcreteClaudeModelID(model, ids)
	}
	levels, err := lineRunnerThinkingLevels(tx, orgID, factoryID, node, catalog, ownerID)
	if err != nil {
		return nil, err
	}
	return levels(model), nil
}

func dispatchSuperPlaneModel(tx *gorm.DB, orgID, factoryID uuid.UUID, node models.Node, model string) (models.DefaultHostedLLMModel, error) {
	if model == "" {
		return models.ResolveRunnableHostedLLMModel(tx, orgID, &factoryID)
	}
	parsed, err := models.ParseHostedLLMModelKey(model)
	if err == nil && parsed.IsSet() {
		return parsed, nil
	}
	provider, _ := node.Configuration["hostedProvider"].(string)
	return models.NormalizeDefaultHostedLLMModel(provider, model)
}

func publishDispatchedWorkOrder(
	logger *log.Entry,
	orgID uuid.UUID,
	factoryID uuid.UUID,
	order *models.FactoryWorkOrder,
	actor *uuid.UUID,
	fromState string,
	startedSteps []*models.FactoryLineStepResult,
) {
	for _, result := range startedSteps {
		publishPendingStepRun(logger, result)
	}

	publishedOrders := map[uuid.UUID]struct{}{order.ID: {}}
	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		order.ID.String(),
		factoryevents.EventTypeLineStepExecutionCreated,
	); err != nil {
		logger.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}
	for _, result := range startedSteps {
		if result == nil || result.Execution == nil {
			continue
		}
		extraID := result.Execution.WorkOrderID
		if _, seen := publishedOrders[extraID]; seen {
			continue
		}
		publishedOrders[extraID] = struct{}{}
		if err := messages.PublishFactoryWorkOrderUpdated(
			factoryID.String(),
			extraID.String(),
			factoryevents.EventTypeLineStepExecutionCreated,
		); err != nil {
			logger.WithError(err).Warnf("Failed to publish factory work order updated for order %s", extraID)
		}
	}

	if fromState != order.State {
		notification := messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: orgID.String(),
			FactoryID:      factoryID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			FromState:      fromState,
			ToState:        order.State,
		}
		if actor != nil {
			notification.ActorUserID = actor.String()
		}
		if err := notification.Publish(); err != nil {
			logger.WithError(err).Warnf("Failed to publish work order notification for order %s", order.ID)
		}
	}
}

// publishPendingStepRun publishes one started step and logs that step's
// work order. A replace or retry can also start a queued run for another
// order, so the dispatch order on logger is not the owner of every run.
func publishPendingStepRun(logger *log.Entry, result *models.FactoryLineStepResult) {
	if result == nil || result.Run == nil {
		return
	}

	runLogger := logger.WithField("run_id", result.Run.ID)
	if result.Execution != nil && result.Execution.WorkOrderID != uuid.Nil {
		runLogger = runLogger.WithField("order_id", result.Execution.WorkOrderID)
	}

	if err := messages.NewCanvasRunMessage(result.Run.WorkflowID.String(), result.Run.ID.String()).PublishPending(); err != nil {
		runLogger.WithError(err).Errorf("Error publishing pending canvas run message: %v", err)
		return
	}
	runLogger.Info("Published pending canvas run")
}
