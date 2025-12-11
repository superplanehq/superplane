package merge

import (
	"fmt"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
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

func (m *Merge) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
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

func (m *Merge) Actions() []components.Action {
	return []components.Action{
		{Name: "timeoutReached"},
	}
}

func (m *Merge) Setup(ctx components.SetupContext) error {
	return nil
}

func (m *Merge) ProcessQueueItem(ctx components.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	mergeGroup := ctx.RootEventID

	execID, err := m.findOrCreateExecution(ctx, mergeGroup)
	if err != nil {
		return nil, err
	}

	if err := ctx.DequeueItem(); err != nil {
		return nil, err
	}

	if err := ctx.UpdateNodeState(models.WorkflowNodeStateReady); err != nil {
		return nil, err
	}

	incoming, err := ctx.CountDistinctIncomingSources()
	if err != nil {
		return nil, err
	}

	md, err := m.addEventToMetadata(ctx, execID)
	if err != nil {
		return nil, err
	}

	// Decode config to check for optional stop expression
	spec := &Spec{}
	_ = mapstructure.Decode(ctx.Configuration, &spec)

	// If already short-circuited, do not finish again
	if md.StopEarly {
		return nil, nil
	}

	// Evaluate stop expression if provided
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
		if b, ok := out.(bool); ok && b {
			// Mark metadata and fail immediately
			md.StopEarly = true
			if err := ctx.SetExecutionMetadata(execID, md); err != nil {
				return nil, err
			}
			return ctx.FailExecution(execID, "stopped", "Stopped by stopIfExpression")
		}
	}

	if len(md.Sources) >= incoming {
		return ctx.PassExecution(execID, map[string][]any{
			components.DefaultOutputChannel.Name: {md},
		})
	}

	return nil, nil
}

func (m *Merge) findOrCreateExecution(ctx components.ProcessQueueContext, mergeGroup string) (uuid.UUID, error) {
	execID, found, err := ctx.FindExecutionIDByKV("merge_group", mergeGroup)
	if err != nil {
		return uuid.Nil, err
	}

	if found {
		return execID, nil
	}

	execID, err = ctx.CreateExecution()
	if err != nil {
		return uuid.Nil, err
	}

	err = ctx.SetExecutionKV(execID, "merge_group", mergeGroup)
	if err != nil {
		return uuid.Nil, err
	}

	md := &ExecutionMetadata{
		GroupKey: mergeGroup,
		EventIDs: []string{},
		Sources:  []string{},
	}

	err = ctx.SetExecutionMetadata(execID, md)
	if err != nil {
		return uuid.Nil, err
	}

	return execID, nil
}

func (m *Merge) addEventToMetadata(ctx components.ProcessQueueContext, execID uuid.UUID) (*ExecutionMetadata, error) {
	md := &ExecutionMetadata{}

	rawMeta, err := ctx.GetExecutionMetadata(execID)
	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(rawMeta, md)
	if err != nil {
		return nil, err
	}

	md.EventIDs = append(md.EventIDs, ctx.EventID)
	// Track distinct source nodes that reached this merge
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

	err = ctx.SetExecutionMetadata(execID, md)
	if err != nil {
		return nil, err
	}

	return md, nil
}

func (m *Merge) HandleAction(ctx components.ActionContext) error {
	switch ctx.Name {
	case "timeoutReached":
		return m.HandleTimeout(ctx)
	default:
		return fmt.Errorf("merge does not support action: %s", ctx.Name)
	}
}

func (m *Merge) HandleTimeout(ctx components.ActionContext) error {
	if ctx.ExecutionStateContext.IsFinished() {
		return nil
	}

	return ctx.ExecutionStateContext.Fail("timeoutReached", "Execution timed out waiting for other inputs")
}

func (m *Merge) Execute(ctx components.ExecutionContext) error {
	spec := &Spec{}

	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	interval := durationFrom(spec.ExecutionTimeout.Value, spec.ExecutionTimeout.Unit)
	if interval > 0 {
		return ctx.RequestContext.ScheduleActionCall("timeoutReached", map[string]any{}, interval)
	}

	return nil
}

func (m *Merge) Cancel(ctx components.ExecutionContext) error {
	return nil
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
