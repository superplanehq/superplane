package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateHTTPSyntheticCheck struct{}

func (c *UpdateHTTPSyntheticCheck) Name() string {
	return "grafana.updateHttpSyntheticCheck"
}

func (c *UpdateHTTPSyntheticCheck) Label() string {
	return "Update HTTP Synthetic Check"
}

func (c *UpdateHTTPSyntheticCheck) Description() string {
	return "Update a Grafana Synthetic Monitoring HTTP check"
}

func (c *UpdateHTTPSyntheticCheck) Documentation() string {
	return `The Update HTTP Synthetic Check component updates an existing Grafana Synthetic Monitoring HTTP check.

## Configuration

- **Synthetic Check**: The synthetic check to update (required)
- **Job**, **Labels**, **Request**, **Schedule**, **Response validation**, and **Per-Check Alerts** are **togglable**. Enable a section only when you want to change it; disabled sections keep the values currently stored in Grafana.

## Output

Returns the updated Grafana synthetic check.`
}

func (c *UpdateHTTPSyntheticCheck) Icon() string {
	return "activity"
}

func (c *UpdateHTTPSyntheticCheck) Color() string {
	return "blue"
}

func (c *UpdateHTTPSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateHTTPSyntheticCheck) Configuration() []configuration.Field {
	return append([]configuration.Field{
		{
			Name:        "syntheticCheck",
			Label:       "Synthetic Check",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The synthetic check to update",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeSyntheticCheck,
				},
			},
		},
	}, syntheticCheckUpdateSharedFields()...)
}

func (c *UpdateHTTPSyntheticCheck) Setup(ctx core.SetupContext) error {
	merged, id, _, existing, client, err := prepareSyntheticCheckUpdate(ctx.HTTP, ctx.Integration, ctx.Configuration, true)
	if err != nil {
		return err
	}
	if err := resolveSyntheticCheckNodeMetadata(ctx, id, existing); err != nil {
		return err
	}
	return resolveSyntheticProbeSummaryMetadata(ctx, merged.Probes, client)
}

func (c *UpdateHTTPSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	merged, id, raw, existing, client, err := prepareSyntheticCheckUpdate(ctx.HTTP, ctx.Integration, ctx.Configuration, false)
	if err != nil {
		return err
	}

	payload, err := buildSyntheticCheckPayload(merged)
	if err != nil {
		return err
	}
	if existing != nil {
		payload.ID = existing.ID
		payload.TenantID = existing.TenantID
		if existing.Settings.HTTP != nil && existing.Settings.HTTP.TLSConfig != nil && payload.Settings.HTTP != nil {
			payload.Settings.HTTP.TLSConfig = existing.Settings.HTTP.TLSConfig
		}
	}

	updated, err := client.UpdateCheck(payload)
	if err != nil {
		return fmt.Errorf("error updating synthetic check: %w", err)
	}

	var alerts []SyntheticCheckAlert
	if syntheticCheckUpdateSectionPresent(raw, "alerts") {
		alerts = buildSyntheticAlertDrafts(merged.Alerts)
		if err := client.UpdateCheckAlerts(id, alerts); err != nil {
			return fmt.Errorf("error configuring synthetic check alerts: %w", err)
		}
	} else if existing != nil {
		alerts, err = client.ListCheckAlerts(id)
		if err != nil {
			return fmt.Errorf("error loading synthetic check alerts: %w", err)
		}
	}
	updated.Alerts = alerts

	output := map[string]any{
		"check":    updated,
		"checkUrl": buildSyntheticCheckWebURL(ctx.Integration, updated.ID),
		"alerts":   alerts,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.syntheticCheck.updated",
		[]any{output},
	)
}

func (c *UpdateHTTPSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateHTTPSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateHTTPSyntheticCheck) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateHTTPSyntheticCheck) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *UpdateHTTPSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateHTTPSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
