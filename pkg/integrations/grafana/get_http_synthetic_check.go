package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetHTTPSyntheticCheck struct{}

func (g *GetHTTPSyntheticCheck) Name() string {
	return "grafana.getHttpSyntheticCheck"
}

func (g *GetHTTPSyntheticCheck) Label() string {
	return "Get HTTP Synthetic Check"
}

func (g *GetHTTPSyntheticCheck) Description() string {
	return "Retrieve a Grafana HTTP synthetic check configuration and best-effort operational metrics"
}

func (g *GetHTTPSyntheticCheck) Documentation() string {
	return `The Get HTTP Synthetic Check component fetches a Grafana synthetic check and enriches it with best-effort operational metrics.

## Use Cases

- **Operational inspection**: fetch the current HTTP check configuration
- **Workflow enrichment**: branch using recent synthetic check health data
- **Troubleshooting**: pull current latency and run totals into incident workflows

## Configuration

- **Synthetic Check**: The synthetic check to retrieve

## Output Channels

- **Up**: All probe locations are passing
- **Partial**: Some probe locations are passing and some are failing
- **Down**: All probe locations are failing

## Output

Returns a combined payload containing:

- **configuration**: the Grafana synthetic check definition
- **alerts**: the configured per-check synthetic alerts when available
- **metrics**: best-effort operational metrics derived from Grafana synthetic monitoring metrics; when present, **lastOutcome** is one of **Up**, **Partial**, or **Down**, matching the output channels`
}

func (g *GetHTTPSyntheticCheck) Icon() string {
	return "activity"
}

func (g *GetHTTPSyntheticCheck) Color() string {
	return "blue"
}

const (
	channelNameUp      = "up"
	channelNamePartial = "partial"
	channelNameDown    = "down"
)

func (g *GetHTTPSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: channelNameUp, Label: "Up", Description: "All probe locations are passing"},
		{Name: channelNamePartial, Label: "Partial", Description: "Some probe locations are passing and some are failing"},
		{Name: channelNameDown, Label: "Down", Description: "All probe locations are failing"},
	}
}

func (g *GetHTTPSyntheticCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "syntheticCheck",
			Label:       "Synthetic Check",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The synthetic check to retrieve",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeSyntheticCheck,
				},
			},
		},
	}
}

func (g *GetHTTPSyntheticCheck) Setup(ctx core.SetupContext) error {
	spec := SyntheticCheckSelectionSpec{}
	if err := decodeSyntheticCheckSpec(ctx.Configuration, &spec); err != nil {
		return err
	}
	if err := validateSyntheticCheckSelection(spec); err != nil {
		return err
	}
	return resolveSyntheticCheckNodeMetadata(ctx, spec.SyntheticCheck, nil)
}

func (g *GetHTTPSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	spec := SyntheticCheckSelectionSpec{}
	if err := decodeSyntheticCheckSpec(ctx.Configuration, &spec); err != nil {
		return err
	}
	if err := validateSyntheticCheckSelection(spec); err != nil {
		return err
	}

	client, err := NewSyntheticsClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating grafana synthetics client: %w", err)
	}

	check, err := client.GetCheck(spec.SyntheticCheck)
	if err != nil {
		return fmt.Errorf("error getting synthetic check: %w", err)
	}

	alerts, err := client.ListCheckAlerts(spec.SyntheticCheck)
	if err == nil {
		check.Alerts = alerts
	}

	metrics := fetchSyntheticCheckMetrics(ctx, check, client.MetricsDataSourceUID)

	output := map[string]any{
		"configuration": check,
		"metrics":       metrics,
		"checkUrl":      buildSyntheticCheckWebURL(ctx.Integration, check.ID),
		"alerts":        check.Alerts,
	}

	channel := ""
	if metrics != nil && metrics.LastOutcome != nil {
		if routed := syntheticCheckOutcomeToChannel(*metrics.LastOutcome); routed != "" {
			channel = routed
		}
	}

	return ctx.ExecutionState.Emit(
		channel,
		"grafana.syntheticCheck",
		[]any{output},
	)
}

func syntheticCheckOutcomeToChannel(outcome string) string {
	switch outcome {
	case "Up":
		return channelNameUp
	case "Partial":
		return channelNamePartial
	case "Down":
		return channelNameDown
	default:
		return ""
	}
}

func (g *GetHTTPSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetHTTPSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetHTTPSyntheticCheck) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetHTTPSyntheticCheck) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (g *GetHTTPSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetHTTPSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
