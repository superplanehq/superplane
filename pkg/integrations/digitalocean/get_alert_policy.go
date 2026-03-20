package digitalocean

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetAlertPolicy struct{}

type GetAlertPolicySpec struct {
	AlertPolicy string `json:"alertPolicy" mapstructure:"alertPolicy"`
}

func (g *GetAlertPolicy) Name() string {
	return "digitalocean.getAlertPolicy"
}

func (g *GetAlertPolicy) Label() string {
	return "Get Alert Policy"
}

func (g *GetAlertPolicy) Description() string {
	return "Fetch details of a DigitalOcean monitoring alert policy"
}

func (g *GetAlertPolicy) Documentation() string {
	return `The Get Alert Policy component retrieves the full details of a monitoring alert policy.

## Use Cases

- **Policy inspection**: Verify the current configuration of an alert policy
- **Conditional logic**: Check whether a policy is enabled before modifying it downstream
- **Audit workflows**: Retrieve alert policy details as part of a compliance or reporting pipeline

## Configuration

- **Alert Policy**: The alert policy to retrieve (required, supports expressions)

## Output

Returns the alert policy object including:
- **uuid**: Alert policy UUID
- **description**: Human-readable description
- **type**: Metric type being monitored (e.g. v1/insights/droplet/cpu)
- **compare**: Comparison operator (GreaterThan/LessThan)
- **value**: Threshold value
- **window**: Evaluation window (5m, 10m, 30m, 1h)
- **entities**: Scoped droplet IDs
- **tags**: Scoped droplet tags
- **enabled**: Whether the policy is active
- **alerts**: Configured notification channels`
}

func (g *GetAlertPolicy) Icon() string {
	return "bell"
}

func (g *GetAlertPolicy) Color() string {
	return "gray"
}

func (g *GetAlertPolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetAlertPolicy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alertPolicy",
			Label:       "Alert Policy",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The alert policy to retrieve",
			Placeholder: "Select alert policy",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "alert_policy",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (g *GetAlertPolicy) Setup(ctx core.SetupContext) error {
	spec := GetAlertPolicySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.AlertPolicy == "" {
		return errors.New("alertPolicy is required")
	}

	if err := resolveAlertPolicyMetadata(ctx, spec.AlertPolicy); err != nil {
		return fmt.Errorf("error resolving alert policy metadata: %v", err)
	}

	return nil
}

func (g *GetAlertPolicy) Execute(ctx core.ExecutionContext) error {
	spec := GetAlertPolicySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	policy, err := client.GetAlertPolicy(spec.AlertPolicy)
	if err != nil {
		return fmt.Errorf("failed to get alert policy: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.alertpolicy.fetched",
		[]any{policy},
	)
}

func (g *GetAlertPolicy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetAlertPolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetAlertPolicy) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetAlertPolicy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetAlertPolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetAlertPolicy) Cleanup(ctx core.SetupContext) error {
	return nil
}
