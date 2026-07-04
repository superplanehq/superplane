package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetSnooze struct{}

type GetSnoozeSpec struct {
	Snooze string `mapstructure:"snooze"`
}

func (g *GetSnooze) Name() string {
	return "gcp.monitoring.getSnooze"
}

func (g *GetSnooze) Label() string {
	return "Monitoring • Get Snooze"
}

func (g *GetSnooze) Description() string {
	return "Read a Cloud Monitoring snooze, including the policies it covers and its time window"
}

func (g *GetSnooze) Documentation() string {
	return `The Get Snooze component reads a Cloud Monitoring snooze.

## Use Cases

- **Auditing**: Inspect which policies are snoozed and until when
- **Chaining**: Read a snooze created upstream before expiring it

## Configuration

- **Snooze**: Pick from the snoozes in your project, or pass an expression chained from an upstream node (e.g. the ` + "`name`" + ` emitted by ` + "`gcp.monitoring.createSnooze`" + `).

## Output

Returns the snooze: **name**, **id**, **displayName**, **policies**, **policiesCount**, **startTime**, **endTime**.

## Important Notes

- Requires the ` + "`roles/monitoring.viewer`" + ` IAM role on the integration's service account`
}

func (g *GetSnooze) Icon() string {
	return "bell-off"
}

func (g *GetSnooze) Color() string {
	return "blue"
}

func (g *GetSnooze) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetSnooze) Configuration() []configuration.Field {
	return []configuration.Field{snoozeSelectorField()}
}

func (g *GetSnooze) Setup(ctx core.SetupContext) error {
	spec := GetSnoozeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if err := validateSnoozeSelection(spec.Snooze); err != nil {
		return err
	}
	return resolveSnoozeMetadata(ctx, spec.Snooze)
}

func (g *GetSnooze) Execute(ctx core.ExecutionContext) error {
	spec := GetSnoozeSpec{}
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

	body, err := client.GetURL(context.Background(), fmt.Sprintf("%s/%s", monitoringBaseURL, name))
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to get snooze", roleHintRead, err))
	}

	var s snooze
	if err := json.Unmarshal(body, &s); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse snooze response: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.monitoring.snooze.fetched",
		[]any{snoozePayload(&s)},
	)
}

func (g *GetSnooze) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetSnooze) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetSnooze) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetSnooze) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetSnooze) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetSnooze) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
