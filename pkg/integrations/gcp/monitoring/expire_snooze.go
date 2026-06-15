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
	return `The Expire Snooze component ends an active snooze immediately by moving its end time to now, so the policies it covers resume sending notifications. It is the GCP equivalent of expiring an Alertmanager silence.

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
	return validateSnoozeSelection(spec.Snooze)
}

func (e *ExpireSnooze) Execute(ctx core.ExecutionContext) error {
	spec := ExpireSnoozeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	name, err := parseSnoozeName(spec.Snooze)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	body := map[string]any{
		"interval": map[string]any{"endTime": time.Now().UTC().Format(time.RFC3339)},
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
