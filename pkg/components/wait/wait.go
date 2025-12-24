package wait

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const PayloadType = "wait.finished"

func init() {
	registry.RegisterComponent("wait", &Wait{})
}

type Wait struct{}

type Spec struct {
	Mode      string  `json:"mode"`
	WaitFor   any     `json:"waitFor"`
	Unit      *string `json:"unit"`
	WaitUntil any     `json:"waitUntil"`
}

type ExecutionMetadata struct {
	StartTime        string `json:"start_time" mapstructure:"start_time"`
	IntervalDuration int64  `json:"interval_duration" mapstructure:"interval_duration"` // Duration in milliseconds
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
							Label: "Interval",
							Value: ModeInterval,
						},
						{
							Label: "Countdown",
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

func parseIntegerValue(value any) (int, error) {
	switch v := value.(type) {
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
		return 0, fmt.Errorf("value is not a valid integer: %s", v)
	default:
		return 0, fmt.Errorf("value must be an integer, got %T", value)
	}
}

func parseDateValue(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		if parsed, parseErr := time.Parse(time.RFC3339, v); parseErr == nil {
			return parsed, nil
		}

		formats := []string{
			time.RFC3339Nano,
			time.RFC822,   // "02 Jan 06 15:04 MST"
			time.RFC822Z,  // "02 Jan 06 15:04 -0700"
			time.RFC850,   // "Monday, 02-Jan-06 15:04:05 MST"
			time.RFC1123,  // "Mon, 02 Jan 2006 15:04:05 MST"
			time.RFC1123Z, // "Mon, 02 Jan 2006 15:04:05 -0700"
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02 15:04:05 -0700 MST", // Format like "2027-08-14 02:00:00 +0200 CEST"
			"2006-01-02 15:04:05 -0700",     // Format like "2027-08-14 02:00:00 +0200"
			"2006-01-02 15:04:05 MST",       // Format like "2027-08-14 02:00:00 CEST"
			"2006-01-02T15:04:05-07:00",     // ISO with timezone offset
			"2006-01-02T15:04:05.000-07:00", // ISO with milliseconds and timezone
			"2006-01-02T15:04Z",
			"2006-01-02T15:04",
			"2006-01-02 15:04",
			"2006-01-02", // Simple date format like "2025-12-25"
			"01/02/2006", // US date format
			"02/01/2006", // European date format
			"2006/01/02", // ISO-like with slashes
		}
		for _, format := range formats {
			if parsed, parseErr := time.Parse(format, v); parseErr == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("value is not a valid date format: %s", v)
	default:
		return time.Time{}, fmt.Errorf("value must be a date/time, got %T", value)
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

func createPayload(startedAt, finishedAt, result, reason string, actor *core.User) map[string]any {
	output := map[string]any{
		"started_at":  startedAt,
		"finished_at": finishedAt,
		"result":      result,
		"reason":      reason,
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

	log.Info("no start time found in metadata")
	return time.Now().Format(time.RFC3339)
}

func (w *Wait) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	// Store start time in metadata
	startTime := time.Now().Format(time.RFC3339)

	var interval time.Duration

	switch spec.Mode {
	case ModeInterval:

		if spec.WaitFor == nil || spec.Unit == nil {
			return errors.New("waitFor and unit are required for interval mode")
		}

		value, err := parseIntegerValue(spec.WaitFor)
		if err != nil {
			return fmt.Errorf("Failed to parse waitFor value: %v", err)
		}

		interval, err = calculateIntervalDuration(value, *spec.Unit)
		if err != nil {
			return fmt.Errorf("Invalid interval configuration: %v", err)
		}

	case ModeCountdown:

		if spec.WaitUntil == nil {
			return errors.New("waitUntil is required for countdown mode")
		}

		targetTime, err := parseDateValue(spec.WaitUntil)
		if err != nil {
			return fmt.Errorf("Failed to parse waitUntil value: %v", err)
		}

		now := time.Now()
		interval = targetTime.Sub(now)
		if interval <= 0 {
			return fmt.Errorf("Target time %s is in the past", targetTime.Format(time.RFC3339))
		}

	default:
		return fmt.Errorf("Invalid mode: %s. Must be either '%s' or '%s'", spec.Mode, ModeInterval, ModeCountdown)
	}

	// Store start time and calculated interval duration in metadata
	err = ctx.MetadataContext.Set(ExecutionMetadata{
		StartTime:        startTime,
		IntervalDuration: interval.Milliseconds(),
	})

	if err != nil {
		return err
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

	return ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{
			createPayload(
				getStartTimeFromMetadata(ctx.MetadataContext),
				time.Now().Format(time.RFC3339),
				"completed",
				"timeout",
				nil,
			),
		},
	)
}

func (w *Wait) HandlePushThrough(ctx core.ActionContext) error {
	if ctx.ExecutionStateContext.IsFinished() {
		return nil
	}

	return ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{
			createPayload(
				getStartTimeFromMetadata(ctx.MetadataContext),
				time.Now().Format(time.RFC3339),
				"completed",
				"manual_override",
				ctx.AuthContext.AuthenticatedUser(),
			),
		},
	)
}

func (w *Wait) Setup(ctx core.SetupContext) error {
	return nil
}

func (w *Wait) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (w *Wait) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (w *Wait) Cancel(ctx core.ExecutionContext) error {
	return ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{
			createPayload(
				getStartTimeFromMetadata(ctx.MetadataContext),
				time.Now().Format(time.RFC3339),
				"cancelled",
				"user_cancel",
				ctx.AuthContext.AuthenticatedUser(),
			),
		},
	)
}
