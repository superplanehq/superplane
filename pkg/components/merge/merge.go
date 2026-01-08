package merge

import (
	"fmt"
	"net/http"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

// Merge is a component that passes its input downstream on
// the default channel. The queue/worker layer is responsible
// for aggregating inputs from multiple parents.

func init() {
	registry.RegisterComponent("merge", &Merge{})
}

type Merge struct{}

func (m *Merge) Name() string        { return "merge" }
func (m *Merge) Label() string       { return "Merge" }
func (m *Merge) Description() string { return "Merge multiple upstream inputs and forward" }
func (m *Merge) Icon() string        { return "arrow-right-from-line" }
func (m *Merge) Color() string       { return "gray" }

func (m *Merge) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

type Spec struct {
	ExecutionTimeout struct {
		Value int    `json:"value"`
		Unit  string `json:"unit"`
	} `json:"executionTimeout"`

	// Optional expression to short-circuit waiting for all inputs.
	// The expression is evaluated against the incoming event input using
	// the Expr language with the input bound to the variable '$'.
	// If it evaluates to true, the merge finishes immediately.
	StopIfExpression string `json:"stopIfExpression" mapstructure:"stopIfExpression"`
}

func (m *Merge) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "executionTimeout",
			Label:    "Execution Timeout",
			Type:     configuration.FieldTypeObject,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "value",
							Label:    "Timeout",
							Type:     configuration.FieldTypeNumber,
							Required: true,
						},
						{
							Name:     "unit",
							Label:    "Unit",
							Type:     configuration.FieldTypeSelect,
							Required: true,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Seconds", Value: "seconds"},
										{Label: "Minutes", Value: "minutes"},
										{Label: "Hours", Value: "hours"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "stopIfExpression",
			Label:       "Stop if",
			Type:        configuration.FieldTypeString,
			Description: "When true, stop waiting and finish immediately.",
			Placeholder: "e.g. $.result == 'fail'",
			Required:    false,
		},
	}
}

func (m *Merge) Actions() []core.Action {
	return []core.Action{
		{Name: "timeoutReached"},
	}
}

func (m *Merge) Setup(ctx core.SetupContext) error {
	return nil
}

func (m *Merge) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	spec := &Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}

	executionCtx, err := m.findOrCreateExecution(ctx, ctx.RootEventID)
	if err != nil {
		return nil, fmt.Errorf("error finding or creating execution: %v", err)
	}

	if err := ctx.DequeueItem(); err != nil {
		return nil, fmt.Errorf("error dequeuing item: %v", err)
	}

	if err := ctx.UpdateNodeState(models.WorkflowNodeStateReady); err != nil {
		return nil, fmt.Errorf("error updating node state: %v", err)
	}

	incoming, err := ctx.CountDistinctIncomingSources()
	if err != nil {
		return nil, fmt.Errorf("error counting distinct incoming sources: %v", err)
	}

	md, err := m.addEventToMetadata(ctx, executionCtx)
	if err != nil {
		return nil, fmt.Errorf("error adding event to metadata: %v", err)
	}

	//
	// Check for optional stop expression
	// If already short-circuited, do not finish again
	//
	if md.StopEarly {
		return nil, nil
	}

	//
	// Evaluate stop expression if provided
	//
	if spec.StopIfExpression != "" {
		env := map[string]any{
			"$": ctx.Input,
		}

		vm, err := expr.Compile(spec.StopIfExpression, expr.Env(env), expr.AsBool())
		if err != nil {
			return nil, fmt.Errorf("stopIfExpression compilation failed: %w", err)
		}

		out, err := expr.Run(vm, env)
		if err != nil {
			return nil, fmt.Errorf("stopIfExpression evaluation failed: %w", err)
		}

		//
		// If stopExpression is truthy,
		// we mark metadata and fail immediately
		//
		if b, ok := out.(bool); ok && b {
			md.StopEarly = true
			err := executionCtx.Metadata.Set(md)
			if err != nil {
				return nil, err
			}

			return &executionCtx.ID, executionCtx.ExecutionState.Fail(models.WorkflowNodeExecutionResultReasonError, "Stopped by stopIfExpression")
		}
	}

	if len(md.Sources) >= incoming {
		return &executionCtx.ID, executionCtx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"merge.finished",
			[]any{md},
		)
	}

	return nil, nil
}

func (m *Merge) findOrCreateExecution(ctx core.ProcessQueueContext, mergeGroup string) (*core.ExecutionContext, error) {
	executionCtx, err := ctx.FindExecutionByKV("merge_group", mergeGroup)
	if err != nil {
		return nil, err
	}

	//
	// Execution already exists, just return it.
	//
	if executionCtx != nil {
		return executionCtx, nil
	}

	//
	// Execution does not exist yet, create it.
	//
	executionCtx, err = ctx.CreateExecution()
	if err != nil {
		return nil, err
	}

	err = executionCtx.ExecutionState.SetKV("merge_group", mergeGroup)
	if err != nil {
		return nil, err
	}

	md := &ExecutionMetadata{
		GroupKey: mergeGroup,
		EventIDs: []string{},
		Sources:  []string{},
	}

	err = executionCtx.Metadata.Set(md)
	if err != nil {
		return nil, err
	}

	return executionCtx, nil
}

func (m *Merge) addEventToMetadata(ctx core.ProcessQueueContext, executionCtx *core.ExecutionContext) (*ExecutionMetadata, error) {
	md := &ExecutionMetadata{}
	err := mapstructure.Decode(executionCtx.Metadata.Get(), md)
	if err != nil {
		return nil, err
	}

	md.EventIDs = append(md.EventIDs, ctx.EventID)

	//
	// Track distinct source nodes that reached this merge
	//
	if ctx.SourceNodeID != "" {
		exists := false
		for _, s := range md.Sources {
			if s == ctx.SourceNodeID {
				exists = true
				break
			}
		}
		if !exists {
			md.Sources = append(md.Sources, ctx.SourceNodeID)
		}
	}

	err = executionCtx.Metadata.Set(md)
	if err != nil {
		return nil, err
	}

	return md, nil
}

func (m *Merge) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "timeoutReached":
		return m.HandleTimeout(ctx)
	default:
		return fmt.Errorf("merge does not support action: %s", ctx.Name)
	}
}

func (m *Merge) HandleTimeout(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	return ctx.ExecutionState.Fail(models.WorkflowNodeExecutionResultReasonError, "Execution timed out waiting for other inputs")
}

func (m *Merge) Execute(ctx core.ExecutionContext) error {
	spec := &Spec{}

	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	interval := durationFrom(spec.ExecutionTimeout.Value, spec.ExecutionTimeout.Unit)
	if interval > 0 {
		return ctx.Requests.ScheduleActionCall("timeoutReached", map[string]any{}, interval)
	}

	return nil
}

func (m *Merge) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (m *Merge) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func durationFrom(value int, unit string) time.Duration {
	switch unit {
	case "seconds":
		return time.Duration(value) * time.Second
	case "minutes":
		return time.Duration(value) * time.Minute
	case "hours":
		return time.Duration(value) * time.Hour
	default:
		return 0
	}
}
