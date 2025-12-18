package wait

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("wait", &Wait{})
}

type Wait struct{}

type Spec struct {
	Mode      string  `json:"mode"`
	WaitFor   *string `json:"waitFor"`
	Unit      *string `json:"unit"`
	WaitUntil *string `json:"waitUntil"`
}

type ExecutionMetadata struct {
	StartTime string `json:"start_time"`
}

const (
	ModeInterval  = "interval"
	ModeCountdown = "countdown"
)

func (w *Wait) Name() string {
	return "wait"
}

func (w *Wait) Label() string {
	return "Wait"
}

func (w *Wait) Description() string {
	return "Wait for a certain amount of time"
}

func (w *Wait) Icon() string {
	return "alarm-clock"
}

func (w *Wait) Color() string {
	return "yellow"
}

func (w *Wait) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (w *Wait) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "mode",
			Label:    "Wait Mode",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  ModeInterval,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Wait for a fixed time interval",
							Value: ModeInterval,
						},
						{
							Label: "Wait until a specific date/time",
							Value: ModeCountdown,
						},
					},
				},
			},
		},
		{
			Name:        "waitFor",
			Label:       "Wait for...",
			Type:        configuration.FieldTypeString,
			Description: "Component will wait for a fixed amount of time before emitting the event forward.\n\nSupports expressions and expects integer.\n\nExample expressions:\n{{$.wait_time}}\n{{$.wait_time + 5}}\n{{$.status == \"urgent\" ? 0 : 30}}",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeInterval}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeInterval}},
			},
		},
		{
			Name:        "unit",
			Label:       "Unit",
			Type:        configuration.FieldTypeSelect,
			Description: "Time unit for the interval",
			Default:     "seconds",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeInterval}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeInterval}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Seconds",
							Value: "seconds",
						},
						{
							Label: "Minutes",
							Value: "minutes",
						},
						{
							Label: "Hours",
							Value: "hours",
						},
					},
				},
			},
		},
		{
			Name:        "waitUntil",
			Label:       "Wait until",
			Type:        configuration.FieldTypeString,
			Description: "Component will countdown until the provided date/time before emitting an event forward.\n\nSupports expressions and expects date in [ISO 8601](https://www.timestamp-converter.com/) format.\n\nExample expressions:\n{{$.run_time}}\n{{$.run_time.In(timezone(\"UTC\"))}}\n{{$.run_time + duration(\"48h\")}}",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeCountdown}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeCountdown}},
			},
		},
	}
}

func evaluateIntegerExpression(expression string, data any) (int, error) {
	env := map[string]any{
		"$": data,
	}

	vm, err := expr.Compile(expression, []expr.Option{
		expr.Env(env),
		expr.WithContext("ctx"),
	}...)
	if err != nil {
		return 0, fmt.Errorf("expression compilation failed: %w", err)
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return 0, fmt.Errorf("expression evaluation failed: %w", err)
	}

	switch v := output.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:

		if parsed, parseErr := strconv.Atoi(v); parseErr == nil {
			return parsed, nil
		}
		return 0, fmt.Errorf("expression result is not a valid integer: %s", v)
	default:
		return 0, fmt.Errorf("expression must evaluate to integer, got %T", output)
	}
}

func evaluateDateExpression(expression string, data any) (time.Time, error) {
	env := map[string]any{
		"$": data,
	}

	vm, err := expr.Compile(expression, []expr.Option{
		expr.Env(env),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	}...)
	if err != nil {
		return time.Time{}, fmt.Errorf("expression compilation failed: %w", err)
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return time.Time{}, fmt.Errorf("expression evaluation failed: %w", err)
	}

	switch v := output.(type) {
	case time.Time:
		return v, nil
	case string:

		if parsed, parseErr := time.Parse(time.RFC3339, v); parseErr == nil {
			return parsed, nil
		}

		formats := []string{
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}
		for _, format := range formats {
			if parsed, parseErr := time.Parse(format, v); parseErr == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("expression result is not a valid date format: %s", v)
	default:
		return time.Time{}, fmt.Errorf("expression must evaluate to date/time, got %T", output)
	}
}

func calculateIntervalDuration(value int, unit string) (time.Duration, error) {
	if value <= 0 {
		return 0, fmt.Errorf("wait interval must be positive, got: %d", value)
	}

	switch unit {
	case "seconds":
		return time.Duration(value) * time.Second, nil
	case "minutes":
		return time.Duration(value) * time.Minute, nil
	case "hours":
		return time.Duration(value) * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid unit: %s", unit)
	}
}

func createCompletionOutput(startTime, finishTime, result, reason string, actor *core.User) map[string]any {
	output := map[string]any{
		"timestamp_started":  startTime,
		"timestamp_finished": finishTime,
		"result":             result,
		"reason":             reason,
		"actor":              nil,
	}

	if actor != nil {
		output["actor"] = map[string]any{
			"email":        actor.Email,
			"display_name": actor.Name,
		}
	}

	return output
}

func getStartTimeFromMetadata(metadataCtx core.MetadataContext) string {
	metadata := ExecutionMetadata{}
	if err := mapstructure.Decode(metadataCtx.Get(), &metadata); err == nil && metadata.StartTime != "" {
		return metadata.StartTime
	}
	// Fallback to current time if no start time found
	return time.Now().Format(time.RFC3339)
}

func (w *Wait) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	// Store start time in metadata
	startTime := time.Now().Format(time.RFC3339)
	metadata := ExecutionMetadata{StartTime: startTime}
	ctx.MetadataContext.Set(metadata)

	var interval time.Duration

	switch spec.Mode {
	case ModeInterval:

		if spec.WaitFor == nil || spec.Unit == nil {
			return fmt.Errorf("waitFor and unit are required for interval mode")
		}

		value, err := evaluateIntegerExpression(*spec.WaitFor, ctx.Data)
		if err != nil {
			return fmt.Errorf("failed to evaluate waitFor expression: %w", err)
		}

		interval, err = calculateIntervalDuration(value, *spec.Unit)
		if err != nil {
			return err
		}

	case ModeCountdown:

		if spec.WaitUntil == nil {
			return fmt.Errorf("waitUntil is required for countdown mode")
		}

		targetTime, err := evaluateDateExpression(*spec.WaitUntil, ctx.Data)
		if err != nil {
			return fmt.Errorf("failed to evaluate waitUntil expression: %w", err)
		}

		now := time.Now()
		interval = targetTime.Sub(now)

		if interval <= 0 {
			return fmt.Errorf("target time %s is in the past", targetTime.Format(time.RFC3339))
		}

	default:
		return fmt.Errorf("invalid mode: %s. Must be either '%s' or '%s'", spec.Mode, ModeInterval, ModeCountdown)
	}

	return ctx.RequestContext.ScheduleActionCall("timeReached", map[string]any{}, interval)
}

func (w *Wait) Actions() []core.Action {
	return []core.Action{
		{
			Name: "timeReached",
		},
		{
			Name:           "pushThrough",
			Description:    "Push Through",
			UserAccessible: true,
		},
	}
}

func (w *Wait) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "timeReached":
		return w.HandleTimeReached(ctx)
	case "pushThrough":
		return w.HandlePushThrough(ctx)

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (w *Wait) HandleTimeReached(ctx core.ActionContext) error {
	if ctx.ExecutionStateContext.IsFinished() {
		return nil
	}

	startTime := getStartTimeFromMetadata(ctx.MetadataContext)
	finishTime := time.Now().Format(time.RFC3339)
	completionOutput := createCompletionOutput(startTime, finishTime, "completed", "timeout", nil)

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		core.DefaultOutputChannel.Name: {completionOutput},
	})
}

func (w *Wait) HandlePushThrough(ctx core.ActionContext) error {
	if ctx.ExecutionStateContext.IsFinished() {
		return nil
	}

	// Create completion output for manual override
	startTime := getStartTimeFromMetadata(ctx.MetadataContext)
	finishTime := time.Now().Format(time.RFC3339)
	actor := ctx.AuthContext.AuthenticatedUser()
	completionOutput := createCompletionOutput(startTime, finishTime, "completed", "manual_override", actor)

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		core.DefaultOutputChannel.Name: {completionOutput},
	})
}

func (w *Wait) Setup(ctx core.SetupContext) error {
	return nil
}

func (w *Wait) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (w *Wait) ProcessQueueItem(ctx core.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	return ctx.DefaultProcessing()
}

func (w *Wait) Cancel(ctx core.ExecutionContext) error {
	startTime := getStartTimeFromMetadata(ctx.MetadataContext)
	finishTime := time.Now().Format(time.RFC3339)
	actor := ctx.AuthContext.AuthenticatedUser()
	completionOutput := createCompletionOutput(startTime, finishTime, "cancelled", "user_cancel", actor)

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		core.DefaultOutputChannel.Name: {completionOutput},
	})
}
