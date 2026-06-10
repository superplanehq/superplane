package loop

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "loop"

const (
	loopSessionKey    = "loop_session"
	nextIterationHook = "nextIteration"

	ChannelNameNext = "next"
	ChannelNameDone = "done"

	PayloadTypeNext = "loop.next"
	PayloadTypeDone = "loop.done"
)

const (
	DelayStrategyFixed       = "fixed"
	DelayStrategyExponential = "exponential"

	DelayMinIntervalSeconds = 1
	DelayMaxIntervalSeconds = 300
)

const defaultMaxIterations = 100

const (
	StopReasonConditionMet  = "conditionTrue"
	StopReasonMaxIterations = "max_iterations"
)

func init() {
	registry.RegisterAction(ComponentName, &Loop{})
}

type Loop struct{}

type Spec struct {
	UntilExpression        string     `json:"untilExpression"`
	MaxIterations          int        `json:"maxIterations"`
	DelayBetweenIterations *DelaySpec `json:"delayBetweenIterations,omitempty"`
}

type DelaySpec struct {
	Enabled         bool   `json:"enabled" mapstructure:"enabled"`
	Strategy        string `json:"strategy" mapstructure:"strategy"`
	IntervalSeconds int    `json:"intervalSeconds" mapstructure:"intervalSeconds"`
}

type ExecutionMetadata struct {
	Iteration                int       `json:"iteration"`
	MaxIterations            int       `json:"maxIterations"`
	Active                   bool      `json:"active"`
	StartedAt                time.Time `json:"startedAt"`
	WaitingBetweenIterations bool      `json:"waitingBetweenIterations,omitempty"`
}

func (c *Loop) Name() string {
	return ComponentName
}

func (c *Loop) Label() string {
	return "Loop"
}

func (c *Loop) Description() string {
	return "Repeat downstream steps until a condition is met"
}

func (c *Loop) Documentation() string {
	return `The Loop component runs downstream steps repeatedly until an exit condition becomes true.

## Use Cases

- Poll an API until a resource reaches a ready state
- Retry a workflow segment until validation passes
- Paginate through results until all pages are processed
- Run approval or review cycles until consensus is reached

## How It Works

1. On the first run, Loop emits to the **Next** channel and starts the loop session
2. Connect downstream nodes to the Next output and wire the last step back to Loop
3. When those steps finish, Loop evaluates the **Until Expression**
4. If the expression is ` + "`true`" + `, Loop emits on **Done** and the loop ends
5. If the expression is ` + "`false`" + `, Loop emits on **Next** again for another iteration

## Wiring

` + "```" + `
Trigger → Loop → Step A → Step B ──┐
              ↑                    │
              └────────────────────┘
` + "```" + `

Edges back into Loop are allowed so downstream steps can return control for the next condition check.

## Output Channels

- **Done**: Emitted once when the loop stops. Payload is under ` + "`data.done`" + ` with ` + "`iterations`" + `, ` + "`stopReason`" + ` (` + "`conditionTrue`" + ` or ` + "`max_iterations`" + `), and ` + "`elapsedMs`" + `
- **Next**: Emitted at the start of each iteration. Payload is under ` + "`data.next`" + ` with ` + "`iteration`" + ` and ` + "`maxIterations`" + `

## Limits

- **Max Iterations** caps how many iterations are allowed (default ` + fmt.Sprintf("%d", defaultMaxIterations) + `, maximum ` + fmt.Sprintf("%d", core.MaxEmitCount) + `)

## Delay Between Iterations

Optionally wait before starting the next iteration. Uses the same fixed or exponential backoff strategies as the HTTP component retry settings. The first iteration always starts immediately; delays apply between subsequent iterations only.

## Expression Environment

The until expression has access to:

- **$**: The run context data, including outputs from the latest iteration
- **root()**: Access root event data
- **previous()**: Access previous node outputs`
}

func (c *Loop) Icon() string {
	return "refresh-cw"
}

func (c *Loop) Color() string {
	return "indigo"
}

func (c *Loop) ExampleOutput() map[string]any {
	return exampleOutput()
}

func (c *Loop) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameDone, Label: "Done"},
		{Name: ChannelNameNext, Label: "Next"},
	}
}

func (c *Loop) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "untilExpression",
			Label:       "Until Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Boolean expression that must evaluate to true to stop looping",
			Required:    true,
		},
		{
			Name:        "maxIterations",
			Label:       "Max Iterations",
			Type:        configuration.FieldTypeNumber,
			Description: "Maximum number of iterations before the loop stops",
			Default:     defaultMaxIterations,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(core.MaxEmitCount),
				},
			},
		},
		{
			Name:        "delayBetweenIterations",
			Type:        configuration.FieldTypeObject,
			Label:       "Delay between iterations",
			Required:    false,
			Description: "Wait before starting the next iteration",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "enabled",
							Label:       "Enable delay",
							Type:        configuration.FieldTypeBool,
							Required:    false,
							Default:     false,
							Description: "Wait between loop iterations before emitting on Next again.",
						},
						{
							Name:        "strategy",
							Type:        configuration.FieldTypeSelect,
							Label:       "Strategy",
							Required:    false,
							Default:     DelayStrategyFixed,
							Description: "Delay strategy",
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Fixed", Value: DelayStrategyFixed},
										{Label: "Exponential", Value: DelayStrategyExponential},
									},
								},
							},
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "enabled", Values: []string{"true"}},
							},
						},
						{
							Name:        "intervalSeconds",
							Label:       "Delay interval (seconds)",
							Type:        configuration.FieldTypeNumber,
							Required:    false,
							Default:     15,
							Description: "Seconds to wait before the next iteration",
							TypeOptions: &configuration.TypeOptions{
								Number: &configuration.NumberTypeOptions{
									Min: intPtr(DelayMinIntervalSeconds),
									Max: intPtr(DelayMaxIntervalSeconds),
								},
							},
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "enabled", Values: []string{"true"}},
							},
						},
					},
				},
			},
		},
	}
}

func (c *Loop) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateSpec(spec)
}

func (c *Loop) Execute(ctx core.ExecutionContext) error {
	return nil
}

func (c *Loop) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if err := validateSpec(spec); err != nil {
		return nil, err
	}

	executionCtx, isNew, err := c.findOrCreateSession(ctx, spec)
	if err != nil {
		return nil, err
	}

	if executionCtx.ExecutionState.IsFinished() {
		if err := ctx.DequeueItem(); err != nil {
			return nil, fmt.Errorf("failed to dequeue item: %w", err)
		}
		return nil, nil
	}

	if err := ctx.DequeueItem(); err != nil {
		return nil, fmt.Errorf("failed to dequeue item: %w", err)
	}

	if err := ctx.UpdateNodeState(models.CanvasNodeStateReady); err != nil {
		return nil, fmt.Errorf("failed to update node state: %w", err)
	}

	if isNew {
		md, err := readMetadata(executionCtx)
		if err != nil {
			return nil, err
		}

		return &executionCtx.ID, executionCtx.ExecutionState.EmitAndContinue(
			ChannelNameNext,
			PayloadTypeNext,
			[]any{nextPayload(md.Iteration, spec.MaxIterations)},
		)
	}

	return c.handleFeedback(ctx, spec, executionCtx)
}

func (c *Loop) findOrCreateSession(ctx core.ProcessQueueContext, spec Spec) (*core.ExecutionContext, bool, error) {
	executionCtx, err := ctx.FindExecutionByKV(loopSessionKey, ctx.RootEventID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find loop session: %w", err)
	}

	if executionCtx != nil {
		return executionCtx, false, nil
	}

	executionCtx, err = ctx.CreateExecution()
	if err != nil {
		return nil, false, fmt.Errorf("failed to create loop execution: %w", err)
	}

	if err := executionCtx.ExecutionState.SetKV(loopSessionKey, ctx.RootEventID); err != nil {
		return nil, false, fmt.Errorf("failed to store loop session: %w", err)
	}

	md := ExecutionMetadata{
		Iteration:     1,
		MaxIterations: spec.MaxIterations,
		Active:        true,
		StartedAt:     time.Now(),
	}
	if err := executionCtx.Metadata.Set(md); err != nil {
		return nil, false, fmt.Errorf("failed to set loop metadata: %w", err)
	}

	return executionCtx, true, nil
}

func (c *Loop) handleFeedback(ctx core.ProcessQueueContext, spec Spec, anchor *core.ExecutionContext) (*uuid.UUID, error) {

	md, err := readMetadata(anchor)
	if err != nil {
		return nil, err
	}

	if !md.Active {
		return nil, nil
	}

	done, err := evaluateUntil(spec.UntilExpression, ctx.Expressions)
	if err != nil {
		return c.failLoop(anchor, md, err.Error())
	}

	if done {
		return c.completeLoop(ctx, anchor, md, StopReasonConditionMet)
	}

	if md.Iteration >= spec.MaxIterations {
		return c.completeLoop(ctx, anchor, md, StopReasonMaxIterations)
	}

	md.Iteration++
	if err := anchor.Metadata.Set(md); err != nil {
		return nil, fmt.Errorf("failed to update loop metadata: %w", err)
	}

	return c.emitNextIteration(anchor, spec, md)
}

func (c *Loop) emitNextIteration(anchor *core.ExecutionContext, spec Spec, md ExecutionMetadata) (*uuid.UUID, error) {
	delay := iterationDelay(spec.DelayBetweenIterations, md.Iteration)
	if delay <= 0 {
		return &anchor.ID, anchor.ExecutionState.EmitAndContinue(
			ChannelNameNext,
			PayloadTypeNext,
			[]any{nextPayload(md.Iteration, spec.MaxIterations)},
		)
	}

	md.WaitingBetweenIterations = true
	if err := anchor.Metadata.Set(md); err != nil {
		return nil, fmt.Errorf("failed to set loop waiting metadata: %w", err)
	}

	if err := anchor.Requests.ScheduleActionCall(nextIterationHook, map[string]any{}, delay); err != nil {
		return nil, fmt.Errorf("failed to schedule next iteration: %w", err)
	}

	return &anchor.ID, nil
}

func (c *Loop) completeLoop(_ core.ProcessQueueContext, anchor *core.ExecutionContext, md ExecutionMetadata, stopReason string) (*uuid.UUID, error) {
	md.Active = false
	md.WaitingBetweenIterations = false
	if err := anchor.Metadata.Set(md); err != nil {
		return nil, fmt.Errorf("failed to update loop metadata: %w", err)
	}

	return &anchor.ID, anchor.ExecutionState.Emit(
		ChannelNameDone,
		PayloadTypeDone,
		[]any{donePayload(md.Iteration, stopReason, loopElapsedMilliseconds(md))},
	)
}

func (c *Loop) failLoop(anchor *core.ExecutionContext, md ExecutionMetadata, message string) (*uuid.UUID, error) {
	md.Active = false
	if err := anchor.Metadata.Set(md); err != nil {
		return nil, fmt.Errorf("failed to update loop metadata: %w", err)
	}

	if err := anchor.ExecutionState.Fail("error", message); err != nil {
		return nil, err
	}

	return &anchor.ID, nil
}

func evaluateUntil(expression string, expressions core.ExpressionContext) (bool, error) {
	output, err := expressions.Run(expression)
	if err != nil {
		return false, fmt.Errorf("until expression evaluation failed: %w", err)
	}

	done, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("until expression must evaluate to boolean, got %T: %v", output, output)
	}

	return done, nil
}

func readMetadata(executionCtx *core.ExecutionContext) (ExecutionMetadata, error) {
	raw, err := json.Marshal(executionCtx.Metadata.Get())
	if err != nil {
		return ExecutionMetadata{}, fmt.Errorf("failed to marshal loop metadata: %w", err)
	}

	md := ExecutionMetadata{}
	if err := json.Unmarshal(raw, &md); err != nil {
		return ExecutionMetadata{}, fmt.Errorf("failed to decode loop metadata: %w", err)
	}
	return md, nil
}

func nextPayload(iteration, maxIterations int) map[string]any {
	return map[string]any{
		"next": map[string]any{
			"iteration":     iteration,
			"maxIterations": maxIterations,
		},
	}
}

func donePayload(iterations int, stopReason string, elapsedMs int64) map[string]any {
	return map[string]any{
		"done": map[string]any{
			"iterations": iterations,
			"stopReason": stopReason,
			"elapsedMs":  elapsedMs,
		},
	}
}

func loopElapsedMilliseconds(md ExecutionMetadata) int64 {
	if md.StartedAt.IsZero() {
		return 0
	}

	return time.Since(md.StartedAt).Milliseconds()
}

func decodeSpec(raw any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return Spec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if spec.MaxIterations == 0 {
		spec.MaxIterations = defaultMaxIterations
	}
	return spec, nil
}

func validateSpec(spec Spec) error {
	if spec.UntilExpression == "" {
		return fmt.Errorf("untilExpression is required")
	}
	if spec.MaxIterations < 1 {
		return fmt.Errorf("maxIterations must be at least 1")
	}
	if spec.MaxIterations > core.MaxEmitCount {
		return fmt.Errorf("maxIterations cannot exceed %d", core.MaxEmitCount)
	}

	return validateDelaySpec(spec.DelayBetweenIterations)
}

func intPtr(v int) *int {
	return &v
}

func (c *Loop) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Loop) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *Loop) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *Loop) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: nextIterationHook,
			Type: core.HookTypeInternal,
		},
	}
}

func (c *Loop) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case nextIterationHook:
		return c.handleNextIteration(ctx)
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (c *Loop) handleNextIteration(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	md, err := readHookMetadata(ctx)
	if err != nil {
		return err
	}

	md.WaitingBetweenIterations = false
	if err := ctx.Metadata.Set(md); err != nil {
		return fmt.Errorf("failed to update loop metadata: %w", err)
	}

	return ctx.ExecutionState.EmitAndContinue(
		ChannelNameNext,
		PayloadTypeNext,
		[]any{nextPayload(md.Iteration, md.MaxIterations)},
	)
}

func readHookMetadata(ctx core.ActionHookContext) (ExecutionMetadata, error) {
	raw, err := json.Marshal(ctx.Metadata.Get())
	if err != nil {
		return ExecutionMetadata{}, fmt.Errorf("failed to marshal loop metadata: %w", err)
	}

	md := ExecutionMetadata{}
	if err := json.Unmarshal(raw, &md); err != nil {
		return ExecutionMetadata{}, fmt.Errorf("failed to decode loop metadata: %w", err)
	}
	return md, nil
}

func iterationDelay(delay *DelaySpec, iteration int) time.Duration {
	if delay == nil || !delay.Enabled || iteration < 2 {
		return 0
	}

	switch delay.Strategy {
	case DelayStrategyExponential:
		return time.Duration(delay.IntervalSeconds) * time.Second * time.Duration(math.Pow(2, float64(iteration-2)))
	default:
		return time.Duration(delay.IntervalSeconds) * time.Second
	}
}

func validateDelaySpec(delay *DelaySpec) error {
	if delay == nil || !delay.Enabled {
		return nil
	}

	if delay.Strategy != DelayStrategyFixed && delay.Strategy != DelayStrategyExponential {
		return fmt.Errorf("invalid delay strategy: %s", delay.Strategy)
	}

	if delay.IntervalSeconds < DelayMinIntervalSeconds {
		return fmt.Errorf("delay interval seconds must be greater than or equal to %d", DelayMinIntervalSeconds)
	}

	if delay.IntervalSeconds > DelayMaxIntervalSeconds {
		return fmt.Errorf("delay interval seconds must be less than or equal to %d", DelayMaxIntervalSeconds)
	}

	return nil
}
