package factories

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

// These factory-owned actions share the decision services used by publication.
// The engine only invokes their generic background hooks.
func init() {
	registry.RegisterAction("prepareHumanReview", &prepareHumanReviewComponent{humanReviewComponent{"prepareHumanReview", "Prepare Human Review"}})
	registry.RegisterAction("recordHumanReviewDecision", &recordHumanReviewDecisionComponent{humanReviewComponent{"recordHumanReviewDecision", "Record Human Review Decision"}})
	registry.RegisterAction("refreshHumanReviewGate", &refreshHumanReviewGateComponent{humanReviewComponent{"refreshHumanReviewGate", "Refresh Human Review Gate"}})
	registry.RegisterAction("listHumanReviewRecoveryCandidates", &listHumanReviewRecoveryComponent{humanReviewComponent{"listHumanReviewRecoveryCandidates", "List Human Review Recovery Candidates"}})
}

type humanReviewComponent struct{ name, label string }

func (c humanReviewComponent) Name() string  { return c.name }
func (c humanReviewComponent) Label() string { return c.label }
func (c humanReviewComponent) Description() string {
	return "Maintain human review for tracked pull requests"
}
func (c humanReviewComponent) Documentation() string {
	return "Use this component in the Request review automation. It uses the current policy and saved assessment for each tracked pull request."
}
func (c humanReviewComponent) Icon() string  { return "users" }
func (c humanReviewComponent) Color() string { return "blue" }
func (c humanReviewComponent) ExampleOutput() map[string]any {
	return map[string]any{"pullRequestId": "2b3cf24d-c0e2-4d42-bbe7-4c30ff2cb2a4"}
}
func (c humanReviewComponent) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (c humanReviewComponent) Configuration() []configuration.Field {
	return nil
}
func (c humanReviewComponent) Setup(core.SetupContext) error   { return nil }
func (c humanReviewComponent) Cleanup(core.SetupContext) error { return nil }
func (c humanReviewComponent) Cancel(ctx core.ExecutionContext) error {
	return ctx.ExecutionState.Cancel()
}
func (c humanReviewComponent) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c humanReviewComponent) Hooks() []core.Hook {
	return []core.Hook{{Name: "reconcile", Type: core.HookTypeInternal}}
}
func (c humanReviewComponent) HandleHook(core.ActionHookContext) error {
	return errors.New("human review requires a background hook")
}
func (c humanReviewComponent) Execute(ctx core.ExecutionContext) error {
	return ctx.Requests.ScheduleActionCall("reconcile", nil, time.Second)
}

type prepareHumanReviewComponent struct{ humanReviewComponent }

func (c *prepareHumanReviewComponent) Description() string {
	return "Authorize reviewer selection from the current policy and merge confidence assessment"
}

func (c *prepareHumanReviewComponent) Configuration() []configuration.Field {
	fields := c.humanReviewComponent.Configuration()
	return append(fields,
		configuration.Field{Name: "enabled", Label: "Enable Request review", Description: "Require a human review decision before the merge gate can pass.", Type: configuration.FieldTypeBool, Default: false},
		configuration.Field{Name: "defaultReviewer", Label: "Default reviewer", Type: configuration.FieldTypeString, Required: false},
		configuration.Field{Name: "fallbackTeam", Label: "Fallback team", Type: configuration.FieldTypeString, Required: false},
	)
}
func (c *prepareHumanReviewComponent) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{{Name: "agent", Label: "Agent needed"}, core.DefaultOutputChannel}
}
func (c *prepareHumanReviewComponent) HandleBackgroundHook(hook core.BackgroundHookContext) (core.BackgroundHookResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := database.DB(ctx)
	state, err := loadHumanReviewComponent(db, hook)
	if err != nil {
		return core.BackgroundHookResult{}, err
	}
	if state == nil || state.pr == nil {
		return core.BackgroundHookResult{}, hook.ExecutionState.Pass()
	}
	client, err := humanReviewComponentClient(hook)
	if err != nil {
		return core.BackgroundHookResult{}, err
	}
	prepared, err := prepareHumanReview(ctx, db, state.factory, state.pr, state.automation, client, state.run, state.root)
	if err != nil {
		return core.BackgroundHookResult{}, err
	}
	if prepared.Decision == nil {
		return core.BackgroundHookResult{}, hook.ExecutionState.Pass()
	}
	payload := map[string]any{"pullRequestId": state.pr.ID.String(), "humanReview": map[string]any{"decisionId": prepared.Decision.ID.String()}}
	channel := "default"
	if prepared.AgentNeeded {
		active, err := humanReviewPublicationCurrent(db, state.pr, prepared.Decision)
		if err != nil {
			return core.BackgroundHookResult{}, err
		}
		if !active {
			return core.BackgroundHookResult{}, hook.ExecutionState.Pass()
		}
		current, err := state.pr.CurrentHumanReview(db)
		if err != nil {
			return core.BackgroundHookResult{}, err
		}
		if current == nil || current.RunID == nil || *current.RunID != state.run.ID || current.State != models.FactoryHumanReviewStateWaitingDecision {
			return core.BackgroundHookResult{}, hook.ExecutionState.Pass()
		}
		payload, channel = prepared.Payload, "agent"
	}
	return core.BackgroundHookResult{}, hook.ExecutionState.Emit(channel, "factory.humanReview.prepared", []any{payload})
}

type recordHumanReviewDecisionComponent struct{ humanReviewComponent }

func (c *recordHumanReviewDecisionComponent) Description() string {
	return "Validate and save the authorized agent decision from its stored completion"
}

func (c *recordHumanReviewDecisionComponent) HandleBackgroundHook(hook core.BackgroundHookContext) (core.BackgroundHookResult, error) {
	db := database.DB(context.Background())
	state, err := loadHumanReviewComponent(db, hook)
	if err != nil {
		return core.BackgroundHookResult{}, err
	}
	if state == nil || state.pr == nil {
		return core.BackgroundHookResult{}, hook.ExecutionState.Pass()
	}
	// Read the persisted incoming event. Configuration and agent-supplied IDs do not grant authority.
	event, err := models.FindCanvasEventForCanvas(db, state.canvas.ID, state.execution.EventID)
	if err != nil {
		return core.BackgroundHookResult{}, err
	}
	if event.ExecutionID != nil {
		agent, err := models.FindNodeExecutionInTransaction(db, state.canvas.ID, *event.ExecutionID)
		if err != nil {
			return core.BackgroundHookResult{}, err
		}
		var decision *models.FactoryHumanReviewDecision
		err = db.Transaction(func(tx *gorm.DB) error {
			var err error
			decision, err = models.RecordHumanReviewDecision(tx, state.canvas, agent, event)
			return err
		})
		if err != nil {
			return core.BackgroundHookResult{}, err
		}
		if decision != nil {
			notifyHumanReviewDecision(db, state.pr, decision)
		}
	}
	return core.BackgroundHookResult{}, hook.ExecutionState.Emit("default", "factory.humanReview.recorded", []any{map[string]any{"pullRequestId": state.pr.ID.String()}})
}

type refreshHumanReviewGateComponent struct{ humanReviewComponent }

func (c *refreshHumanReviewGateComponent) Description() string {
	return "Evaluate current reviews, request eligible reviewers, and publish the review gate"
}

func (c *refreshHumanReviewGateComponent) HandleBackgroundHook(hook core.BackgroundHookContext) (core.BackgroundHookResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := database.DB(ctx)
	state, err := loadHumanReviewComponent(db, hook)
	if err != nil {
		return core.BackgroundHookResult{}, err
	}
	if state == nil || state.pr == nil {
		return core.BackgroundHookResult{}, hook.ExecutionState.Pass()
	}
	client, err := humanReviewComponentClient(hook)
	if err != nil {
		return core.BackgroundHookResult{}, err
	}
	if err := refreshHumanReviewGate(ctx, db, state.factory, state.pr, state.automation, client); err != nil {
		return core.BackgroundHookResult{}, err
	}
	return core.BackgroundHookResult{}, hook.ExecutionState.Pass()
}

type listHumanReviewRecoveryComponent struct{ humanReviewComponent }

func (c *listHumanReviewRecoveryComponent) Description() string {
	return "List tracked pull requests that need review recovery in the current workspace"
}

func (c *listHumanReviewRecoveryComponent) HandleBackgroundHook(hook core.BackgroundHookContext) (core.BackgroundHookResult, error) {
	db := database.DB(context.Background())
	state, err := loadHumanReviewComponent(db, hook)
	if err != nil {
		return core.BackgroundHookResult{}, err
	}
	if state == nil {
		return core.BackgroundHookResult{}, hook.ExecutionState.Pass()
	}
	cursor, _ := hook.ExecutionState.GetKV("afterPullRequestId")
	after, _ := uuid.Parse(cursor)
	pageSize := min(100, config.MaxEmitCount())
	prs, err := models.ListFactoryHumanReviewPullRequestsToReconcile(db.Where("factory_pull_requests.organization_id = ? AND factory_pull_requests.factory_id = ?", state.factory.OrganizationID, state.factory.ID), after, pageSize)
	if err != nil {
		return core.BackgroundHookResult{}, err
	}
	payloads := make([]any, 0, len(prs))
	for _, pr := range prs {
		payloads = append(payloads, map[string]any{"pullRequestId": pr.ID.String(), "repository": pr.Repository, "number": pr.Number})
	}
	if len(payloads) > 0 {
		if err := hook.ExecutionState.EmitAndContinue("default", "factory.humanReview.recovery", payloads); err != nil {
			return core.BackgroundHookResult{}, err
		}
	}
	if len(prs) == pageSize {
		if err := hook.ExecutionState.SetKV("afterPullRequestId", prs[len(prs)-1].ID.String()); err != nil {
			return core.BackgroundHookResult{}, err
		}
		return core.BackgroundHookResult{RetryAfter: time.Second}, nil
	}
	return core.BackgroundHookResult{}, hook.ExecutionState.Pass()
}

type humanReviewComponentState struct {
	canvas     *models.Canvas
	factory    *models.Factory
	automation *humanReviewAutomation
	run        *models.CanvasRun
	execution  *models.CanvasNodeExecution
	root       *models.CanvasEvent
	pr         *models.FactoryPullRequest
}

func loadHumanReviewComponent(db *gorm.DB, hook core.BackgroundHookContext) (*humanReviewComponentState, error) {
	canvas, err := models.FindCanvasInTransaction(db, hook.OrganizationID, hook.WorkflowID)
	if err != nil {
		return nil, err
	}
	if canvas.FactoryID == nil || canvas.LiveVersionID == nil {
		return nil, nil
	}
	factory, err := models.FindFactory(db, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	automation, err := findHumanReviewAutomation(db, factory)
	if err != nil {
		return nil, err
	}
	if automation == nil || automation.canvas.ID != canvas.ID {
		return nil, nil
	}
	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, hook.RunID)
	if err != nil {
		return nil, err
	}
	if run.CancelledAt != nil || run.State == models.CanvasRunStateCancelling || run.Result == models.CanvasRunResultCancelled || run.VersionID != automation.version.ID {
		return nil, nil
	}
	execution, err := models.FindNodeExecutionInTransaction(db, canvas.ID, hook.ExecutionID)
	if err != nil {
		return nil, err
	}
	root, err := models.FindCanvasEventForCanvas(db, canvas.ID, execution.RootEventID)
	if err != nil {
		return nil, err
	}
	state := &humanReviewComponentState{canvas: canvas, factory: factory, automation: automation, run: run, execution: execution, root: root}
	decision, err := models.FindFactoryHumanReviewDecisionByRun(db, run.ID)
	if err != nil {
		return nil, err
	}
	lookup := humanReviewRootLookup(root)
	if decision != nil {
		lookup = models.FactoryPullRequestLookup{ID: decision.PullRequestID}
	}
	pr, err := factory.FindPullRequest(db, lookup)
	if errors.Is(err, models.ErrFactoryPullRequestNotFound) || errors.Is(err, models.ErrFactoryPullRequestLookupIncomplete) {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	if pr.Provider != models.FactoryPullRequestProviderGitHub {
		return state, nil
	}
	state.pr = pr
	return state, nil
}

func humanReviewRootLookup(root *models.CanvasEvent) models.FactoryPullRequestLookup {
	data, _ := models.RootEventSourcePayload(root.Data.Data()).(map[string]any)
	if payload, ok := data["payload"].(map[string]any); ok {
		data = payload
	}
	if parameters, ok := data["parameters"].(map[string]any); ok {
		data = parameters
	}
	id, _ := data["pullRequestId"].(string)
	if id == "" {
		review, _ := data["humanReview"].(map[string]any)
		id, _ = review["pullRequestId"].(string)
	}
	if parsed, err := uuid.Parse(id); err == nil {
		return models.FactoryPullRequestLookup{ID: parsed}
	}
	if repository, ok := data["repository"].(string); ok {
		number, _ := strconv.ParseInt(fmt.Sprint(data["number"]), 10, 64)
		return models.FactoryPullRequestLookup{Provider: models.FactoryPullRequestProviderGitHub, Repository: repository, Number: number}
	}
	pr, _ := data["pull_request"].(map[string]any)
	repository, _ := data["repository"].(map[string]any)
	name, _ := repository["full_name"].(string)
	number, _ := strconv.ParseInt(fmt.Sprint(pr["number"]), 10, 64)
	return models.FactoryPullRequestLookup{Provider: models.FactoryPullRequestProviderGitHub, Repository: name, Number: number}
}

func humanReviewComponentClient(hook core.BackgroundHookContext) (humanReviewGitHub, error) {
	if hook.Integration == nil {
		return nil, fmt.Errorf("Request review requires a GitHub integration")
	}
	return common.NewClient(hook.Integration, hook.HTTP)
}
