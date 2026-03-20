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

type DeleteAlertPolicy struct{}

type DeleteAlertPolicySpec struct {
	AlertPolicy string `json:"alertPolicy" mapstructure:"alertPolicy"`
}

func (d *DeleteAlertPolicy) Name() string {
	return "digitalocean.deleteAlertPolicy"
}

func (d *DeleteAlertPolicy) Label() string {
	return "Delete Alert Policy"
}

func (d *DeleteAlertPolicy) Description() string {
	return "Delete a DigitalOcean monitoring alert policy"
}

func (d *DeleteAlertPolicy) Documentation() string {
	return `The Delete Alert Policy component permanently removes a monitoring alert policy from your DigitalOcean account.

## Use Cases

- **Cleanup**: Remove alert policies that are no longer needed
- **Policy rotation**: Delete old policies as part of a replace workflow
- **Automated teardown**: Remove monitoring policies when decommissioning environments

## Configuration

- **Alert Policy**: The alert policy to delete (required, supports expressions)

## Output

Returns information about the deleted policy:
- **alertPolicyUuid**: The UUID of the alert policy that was deleted

## Important Notes

- This operation is **permanent** and cannot be undone
- If the policy does not exist (already deleted), the component completes successfully (idempotent)`
}

func (d *DeleteAlertPolicy) Icon() string {
	return "trash-2"
}

func (d *DeleteAlertPolicy) Color() string {
	return "red"
}

func (d *DeleteAlertPolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteAlertPolicy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alertPolicy",
			Label:       "Alert Policy",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The alert policy to delete",
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

func (d *DeleteAlertPolicy) Setup(ctx core.SetupContext) error {
	spec := DeleteAlertPolicySpec{}
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

func (d *DeleteAlertPolicy) Execute(ctx core.ExecutionContext) error {
	spec := DeleteAlertPolicySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DeleteAlertPolicy(spec.AlertPolicy)
	if err != nil {
		if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
			// Policy already deleted, emit success
			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"digitalocean.alertpolicy.deleted",
				[]any{map[string]any{"alertPolicyUuid": spec.AlertPolicy}},
			)
		}
		return fmt.Errorf("failed to delete alert policy: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.alertpolicy.deleted",
		[]any{map[string]any{"alertPolicyUuid": spec.AlertPolicy}},
	)
}

func (d *DeleteAlertPolicy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteAlertPolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteAlertPolicy) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteAlertPolicy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteAlertPolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteAlertPolicy) Cleanup(ctx core.SetupContext) error {
	return nil
}
