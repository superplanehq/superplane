package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateSnooze struct{}

type CreateSnoozeSpec struct {
	DisplayName string   `mapstructure:"displayName"`
	Policies    []string `mapstructure:"policies"`
	Duration    string   `mapstructure:"duration"`
}

func (c *CreateSnooze) Name() string {
	return "gcp.monitoring.createSnooze"
}

func (c *CreateSnooze) Label() string {
	return "Monitoring • Create Snooze"
}

func (c *CreateSnooze) Description() string {
	return "Create a Cloud Monitoring snooze that suppresses alert notifications for selected policies over a time window"
}

func (c *CreateSnooze) Documentation() string {
	return `The Create Snooze component creates a Cloud Monitoring **snooze** that prevents the selected alerting policies from sending notifications for a window of time, starting now. It is the GCP equivalent of an Alertmanager silence.

## Use Cases

- **Maintenance windows**: Mute alerts on policies while you deploy or patch
- **Noise control**: Temporarily quiet a flapping policy while you fix the root cause
- **Planned work**: Suppress notifications for a known, expected disruption

## Configuration

- **Display Name**: Human-readable name for the snooze (required)
- **Alerting Policies**: One or more policies to snooze (required, up to 16)
- **Duration**: How long to snooze, starting now (required)

## Output

Returns the created snooze: **name**, **id**, **displayName**, **policies**, **policiesCount**, **startTime**, **endTime**.

## Important Notes

- The snooze starts immediately and ends after the chosen duration
- A snooze suppresses notifications but does not stop incidents from opening
- Requires the ` + "`roles/monitoring.editor`" + ` (or ` + "`roles/monitoring.snoozeEditor`" + `) IAM role`
}

func (c *CreateSnooze) Icon() string {
	return "bell-off"
}

func (c *CreateSnooze) Color() string {
	return "orange"
}

func (c *CreateSnooze) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSnooze) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Human-readable name for the snooze.",
			Placeholder: "e.g. Deploy maintenance window",
		},
		{
			Name:        "policies",
			Label:       "Alerting Policies",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The alerting policies to snooze (up to 16).",
			Placeholder: "Select alerting policies",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeAlertPolicy, Multi: true},
			},
		},
		{
			Name:        "duration",
			Label:       "Duration",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "How long to snooze, starting now.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: snoozeDurationOptions}},
		},
	}
}

func validateCreateSnoozeSpec(spec CreateSnoozeSpec) error {
	if strings.TrimSpace(spec.DisplayName) == "" {
		return errors.New("displayName is required")
	}
	policies := nonEmpty(spec.Policies)
	if len(policies) == 0 {
		return errors.New("at least one alerting policy is required")
	}
	if len(policies) > maxSnoozePolicies {
		return fmt.Errorf("at most %d policies can be snoozed at once", maxSnoozePolicies)
	}
	if !isValidSnoozeDuration(spec.Duration) {
		return errors.New("invalid or missing duration")
	}
	return nil
}

func (c *CreateSnooze) Setup(ctx core.SetupContext) error {
	spec := CreateSnoozeSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	return validateCreateSnoozeSpec(spec)
}

func (c *CreateSnooze) Execute(ctx core.ExecutionContext) error {
	spec := CreateSnoozeSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	if err := validateCreateSnoozeSpec(spec); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	dur, err := time.ParseDuration(spec.Duration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("invalid duration: %v", err))
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	now := time.Now().UTC()
	body := map[string]any{
		"displayName": strings.TrimSpace(spec.DisplayName),
		"criteria":    map[string]any{"policies": nonEmpty(spec.Policies)},
		"interval": map[string]any{
			"startTime": now.Format(time.RFC3339),
			"endTime":   now.Add(dur).Format(time.RFC3339),
		},
	}

	endpoint := fmt.Sprintf("%s/projects/%s/snoozes", monitoringBaseURL, client.ProjectID())
	respBody, err := client.PostURL(context.Background(), endpoint, body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to create snooze", roleHintWrite, err))
	}

	var created snooze
	if err := json.Unmarshal(respBody, &created); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse snooze response: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.monitoring.snooze.created",
		[]any{snoozePayload(&created)},
	)
}

func (c *CreateSnooze) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateSnooze) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSnooze) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateSnooze) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateSnooze) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateSnooze) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// nonEmpty trims and drops blank entries from a string slice.
func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
