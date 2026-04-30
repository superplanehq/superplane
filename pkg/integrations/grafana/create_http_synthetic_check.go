package grafana

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateHTTPSyntheticCheck struct{}

type CreateHTTPSyntheticCheckSpec struct {
	SyntheticCheckSpecBase `mapstructure:",squash"`
}

func (c *CreateHTTPSyntheticCheck) Name() string {
	return "grafana.createHttpSyntheticCheck"
}

func (c *CreateHTTPSyntheticCheck) Label() string {
	return "Create HTTP Synthetic Check"
}

func (c *CreateHTTPSyntheticCheck) Description() string {
	return "Create a Grafana Synthetic Monitoring HTTP check"
}

func (c *CreateHTTPSyntheticCheck) Documentation() string {
	return `The Create HTTP Synthetic Check component creates an HTTP synthetic check in Grafana Synthetic Monitoring.

## Use Cases

- **Availability monitoring**: create checks for API and website uptime
- **Deployment verification**: validate a service immediately after deployment
- **Operational automation**: provision consistent HTTP checks from workflows

## Configuration

Fields are grouped like other synthetic check components:

- **Job** and **Labels**: Check display name and optional key/value labels
- **Request**: URL, HTTP method, headers, body, redirects, basic auth, and bearer token
- **Schedule**: Whether the check is enabled, frequency (seconds), timeout (ms), and probe locations
- **Response validation**: SSL expectations, accepted status codes, and body/header regex rules (optional)
- **Per-Check Alerts**: Optional Grafana synthetic monitoring alerts configured after check creation

## Output

Returns the created Grafana synthetic check, including its ID and HTTP configuration.`
}

func (c *CreateHTTPSyntheticCheck) Icon() string {
	return "activity"
}

func (c *CreateHTTPSyntheticCheck) Color() string {
	return "green"
}

func (c *CreateHTTPSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateHTTPSyntheticCheck) Configuration() []configuration.Field {
	return syntheticCheckSharedFields()
}

func (c *CreateHTTPSyntheticCheck) Setup(ctx core.SetupContext) error {
	spec := CreateHTTPSyntheticCheckSpec{}
	if err := decodeSyntheticCheckSpec(ctx.Configuration, &spec); err != nil {
		return err
	}
	if err := validateSyntheticCheckBase(spec.SyntheticCheckSpecBase); err != nil {
		return err
	}
	return resolveSyntheticProbeSummaryMetadata(ctx, spec.Probes, nil)
}

func (c *CreateHTTPSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	spec := CreateHTTPSyntheticCheckSpec{}
	if err := decodeSyntheticCheckSpec(ctx.Configuration, &spec); err != nil {
		return err
	}
	if err := validateSyntheticCheckBase(spec.SyntheticCheckSpecBase); err != nil {
		return err
	}

	client, err := NewSyntheticsClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating grafana synthetics client: %w", err)
	}

	payload, err := buildSyntheticCheckPayload(spec.SyntheticCheckSpecBase)
	if err != nil {
		return err
	}

	created, err := client.CreateCheck(payload)
	if err != nil {
		return fmt.Errorf("error creating synthetic check: %w", err)
	}

	alerts := buildSyntheticAlertDrafts(spec.Alerts)
	if len(alerts) > 0 {
		if err := client.UpdateCheckAlerts(created.IDString(), alerts); err != nil {
			if _, cleanupErr := client.DeleteCheck(created.IDString()); cleanupErr != nil {
				return errors.Join(
					fmt.Errorf("error configuring synthetic check alerts: %w", err),
					fmt.Errorf("error deleting synthetic check after alert configuration failure: %w", cleanupErr),
				)
			}
			return fmt.Errorf("error configuring synthetic check alerts: %w", err)
		}
		created.Alerts = alerts
	}

	output := map[string]any{
		"check":    created,
		"checkUrl": buildSyntheticCheckWebURL(ctx.Integration, created.ID),
		"alerts":   alerts,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.syntheticCheck.created",
		[]any{output},
	)
}

func (c *CreateHTTPSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateHTTPSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateHTTPSyntheticCheck) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateHTTPSyntheticCheck) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *CreateHTTPSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateHTTPSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
