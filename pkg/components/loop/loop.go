package loop

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "loop"

const (
	loopSessionKey = "loop_session"

	ChannelNameBody = "body"
	ChannelNameDone = "done"

	PayloadTypeBody = "loop.body"
	PayloadTypeDone = "loop.done"
)

const defaultMaxIterations = 100

func init() {
	registry.RegisterAction(ComponentName, &Loop{})
}

type Loop struct{}

type Spec struct {
	UntilExpression string `json:"untilExpression"`
	MaxIterations   int    `json:"maxIterations"`
}

type ExecutionMetadata struct {
	Iteration int  `json:"iteration"`
	Active    bool `json:"active"`
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

1. On the first run, Loop emits to the **Body** channel and starts the loop session
2. Connect downstream nodes to the Body output and wire the last step back to Loop
3. When the body finishes, Loop evaluates the **Until Expression**
4. If the expression is ` + "`true`" + `, Loop emits on **Done** and the loop ends
5. If the expression is ` + "`false`" + `, Loop emits on **Body** again for another iteration

## Wiring

` + "```" + `
Trigger → Loop → Step A → Step B ──┐
              ↑                    │
              └────────────────────┘
` + "```" + `

Edges back into Loop are allowed so the body can return control for the next condition check.

## Output Channels

- **Body**: Emitted at the start of each iteration
- **Done**: Emitted once when the until expression evaluates to true

## Limits

- **Max Iterations** caps how many body runs are allowed (default ` + fmt.Sprintf("%d", defaultMaxIterations) + `, maximum ` + fmt.Sprintf("%d", core.MaxEmitCount) + `)

## Expression Environment

The until expression has access to:

- **$**: The run context data, including outputs from the latest body run
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
		{Name: ChannelNameBody, Label: "Body"},
		{Name: ChannelNameDone, Label: "Done"},
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
			Description: "Maximum number of body iterations before the loop fails",
			Default:     defaultMaxIterations,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(core.MaxEmitCount),
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

	anchor, err := ctx.FindExecutionByKV(loopSessionKey, ctx.RootEventID)
	if err != nil {
		return nil, fmt.Errorf("failed to find loop session: %w", err)
	}

	if anchor == nil {
		return c.startLoop(ctx, spec)
	}

	return c.handleFeedback(ctx, spec, anchor)
}

func (c *Loop) startLoop(ctx core.ProcessQueueContext, spec Spec) (*uuid.UUID, error) {
	executionCtx, err := ctx.CreateExecution()
	if err != nil {
		return nil, fmt.Errorf("failed to create loop execution: %w", err)
	}

	if err := executionCtx.ExecutionState.SetKV(loopSessionKey, ctx.RootEventID); err != nil {
		return nil, fmt.Errorf("failed to store loop session: %w", err)
	}

	md := ExecutionMetadata{
		Iteration: 1,
		Active:    true,
	}
	if err := executionCtx.Metadata.Set(md); err != nil {
		return nil, fmt.Errorf("failed to set loop metadata: %w", err)
	}

	if err := ctx.DequeueItem(); err != nil {
		return nil, fmt.Errorf("failed to dequeue item: %w", err)
	}

	if err := ctx.UpdateNodeState(models.CanvasNodeStateReady); err != nil {
		return nil, fmt.Errorf("failed to update node state: %w", err)
	}

	return &executionCtx.ID, executionCtx.ExecutionState.Emit(
		ChannelNameBody,
		PayloadTypeBody,
		[]any{bodyPayload(md.Iteration, spec.MaxIterations)},
	)
}

func (c *Loop) handleFeedback(ctx core.ProcessQueueContext, spec Spec, anchor *core.ExecutionContext) (*uuid.UUID, error) {
	if anchor.ExecutionState.IsFinished() == false {
		return nil, fmt.Errorf("loop session execution is still running")
	}

	md, err := readMetadata(anchor)
	if err != nil {
		return nil, err
	}

	if err := ctx.DequeueItem(); err != nil {
		return nil, fmt.Errorf("failed to dequeue item: %w", err)
	}

	if err := ctx.UpdateNodeState(models.CanvasNodeStateReady); err != nil {
		return nil, fmt.Errorf("failed to update node state: %w", err)
	}

	if !md.Active {
		return nil, nil
	}

	done, err := evaluateUntil(spec.UntilExpression, ctx.Expressions)
	if err != nil {
		return c.failLoop(anchor, md, err.Error())
	}

	if done {
		md.Active = false
		if err := anchor.Metadata.Set(md); err != nil {
			return nil, fmt.Errorf("failed to update loop metadata: %w", err)
		}

		executionCtx, err := ctx.CreateExecution()
		if err != nil {
			return nil, fmt.Errorf("failed to create loop completion execution: %w", err)
		}

		return &executionCtx.ID, executionCtx.ExecutionState.Emit(
			ChannelNameDone,
			PayloadTypeDone,
			[]any{donePayload(md.Iteration)},
		)
	}

	if md.Iteration >= spec.MaxIterations {
		return c.failLoop(anchor, md, fmt.Sprintf(
			"loop reached max iterations (%d) before until expression became true",
			spec.MaxIterations,
		))
	}

	md.Iteration++
	if err := anchor.Metadata.Set(md); err != nil {
		return nil, fmt.Errorf("failed to update loop metadata: %w", err)
	}

	executionCtx, err := ctx.CreateExecution()
	if err != nil {
		return nil, fmt.Errorf("failed to create loop iteration execution: %w", err)
	}

	return &executionCtx.ID, executionCtx.ExecutionState.Emit(
		ChannelNameBody,
		PayloadTypeBody,
		[]any{bodyPayload(md.Iteration, spec.MaxIterations)},
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
	md := ExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &md); err != nil {
		return ExecutionMetadata{}, fmt.Errorf("failed to decode loop metadata: %w", err)
	}
	return md, nil
}

func bodyPayload(iteration, maxIterations int) map[string]any {
	return map[string]any{
		"iteration":     iteration,
		"maxIterations": maxIterations,
	}
}

func donePayload(iteration int) map[string]any {
	return map[string]any{
		"iteration": iteration,
		"completed": true,
	}
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
	return nil
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
	return []core.Hook{}
}

func (c *Loop) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
