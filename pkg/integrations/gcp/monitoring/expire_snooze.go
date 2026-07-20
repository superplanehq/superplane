package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ExpireSnooze struct{}

type ExpireSnoozeSpec struct {
	Snooze string `mapstructure:"snooze"`
}

func (e *ExpireSnooze) Name() string {
	return "gcp.monitoring.expireSnooze"
}

func (e *ExpireSnooze) Label() string {
	return "Monitoring • Expire Snooze"
}

func (e *ExpireSnooze) Description() string {
	return "End an active Cloud Monitoring snooze now so the snoozed policies can notify again"
}

func (e *ExpireSnooze) Documentation() string {
	return `The Expire Snooze component ends an active snooze immediately, so the policies it covers resume sending notifications. It is the GCP equivalent of expiring an Alertmanager silence.

It moves the snooze's end time to now. Cloud Monitoring requires a snooze window to be at least a minute long, so a snooze that started less than a minute ago is instead cancelled outright (collapsed to a zero-length window) — either way alerting resumes right away.

## Use Cases

- **Early exit**: Maintenance finished ahead of schedule — re-enable alerting now
- **Cleanup**: End a snooze created upstream once a workflow completes

## Configuration

- **Snooze**: Pick from the snoozes in your project, or pass an expression chained from an upstream node (e.g. the ` + "`name`" + ` emitted by ` + "`gcp.monitoring.createSnooze`" + `).

## Output

Returns the updated snooze: **name**, **id**, **displayName**, **policies**, **startTime**, **endTime** (now).

## Important Notes

- Cloud Monitoring has no delete-snooze operation; ending a snooze means setting its end time to the current time
- Requires the ` + "`roles/monitoring.editor`" + ` (or ` + "`roles/monitoring.snoozeEditor`" + `) IAM role`
}

func (e *ExpireSnooze) Icon() string {
	return "bell-off"
}

func (e *ExpireSnooze) Color() string {
	return "red"
}

func (e *ExpireSnooze) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (e *ExpireSnooze) Configuration() []configuration.Field {
	return []configuration.Field{snoozeSelectorField()}
}

func (e *ExpireSnooze) Setup(ctx core.SetupContext) error {
	spec := ExpireSnoozeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if err := validateSnoozeSelection(spec.Snooze); err != nil {
		return err
	}
	return resolveSnoozeMetadata(ctx, spec.Snooze)
}

func (e *ExpireSnooze) Execute(ctx core.ExecutionContext) error {
	spec := ExpireSnoozeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	name, err := resolveSnoozeName(spec.Snooze, client.ProjectID())
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	// Read the snooze first so the new end time respects Cloud Monitoring's rule
	// that a patched interval be either at least one minute long or exactly
	// length 0 (a cancellation). Setting the end time to "now" on a snooze that
	// started less than a minute ago would produce a sub-minute interval, which
	// the API rejects with HTTP 400.
	current, err := client.GetURL(context.Background(), fmt.Sprintf("%s/%s", monitoringBaseURL, name))
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to read snooze", roleHintRead, err))
	}
	var existing snooze
	if err := json.Unmarshal(current, &existing); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse snooze response: %v", err))
	}

	body := map[string]any{
		"interval": map[string]any{"endTime": expireEndTime(&existing, time.Now().UTC())},
	}
	q := url.Values{}
	q.Set("updateMask", "interval.endTime")
	endpoint := fmt.Sprintf("%s/%s?%s", monitoringBaseURL, name, q.Encode())

	respBody, err := client.PatchURL(context.Background(), endpoint, body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to expire snooze", roleHintWrite, err))
	}

	var updated snooze
	if err := json.Unmarshal(respBody, &updated); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse snooze response: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.monitoring.snooze.expired",
		[]any{snoozePayload(&updated)},
	)
}

// expireEndTime picks the interval end time that stops a snooze now while
// satisfying Cloud Monitoring's constraint that a patched interval be at least a
// minute long or exactly length 0. When the snooze has already been active for a
// minute or more it is ended at "now"; otherwise it is collapsed to a
// zero-length interval (endTime == startTime), which cancels it outright so its
// policies can notify again immediately.
func expireEndTime(s *snooze, now time.Time) string {
	if s.Interval != nil && s.Interval.StartTime != "" {
		// End at "now" only when the snooze has already been active for a minute or
		// more. Otherwise — including when the start time cannot be parsed —
		// collapse to a zero-length interval (endTime == startTime), which Cloud
		// Monitoring always accepts as a cancellation. Falling back to "now" here
		// could yield a sub-minute interval that the API rejects with HTTP 400.
		if start, err := time.Parse(time.RFC3339, s.Interval.StartTime); err == nil && now.Sub(start) >= time.Minute {
			return now.Format(time.RFC3339)
		}
		return s.Interval.StartTime
	}
	return now.Format(time.RFC3339)
}

func (e *ExpireSnooze) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (e *ExpireSnooze) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (e *ExpireSnooze) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (e *ExpireSnooze) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (e *ExpireSnooze) Hooks() []core.Hook {
	return []core.Hook{}
}

func (e *ExpireSnooze) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
