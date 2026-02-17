package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeleteFlagPayloadType = "launchdarkly.deleteFlag.result"

type DeleteFlag struct{}

type DeleteFlagSpec struct {
	ProjectKey string `json:"projectKey" mapstructure:"projectKey"`
	FlagKey    string `json:"flagKey" mapstructure:"flagKey"`
}

type DeleteFlagOutput struct {
	Deleted    bool   `json:"deleted"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

func (d *DeleteFlag) Name() string {
	return "launchdarkly.deleteFlag"
}

func (d *DeleteFlag) Label() string {
	return "Delete Feature Flag"
}

func (d *DeleteFlag) Description() string {
	return "Permanently delete a LaunchDarkly feature flag"
}

func (d *DeleteFlag) Documentation() string {
	return `Deletes a LaunchDarkly feature flag from all environments.

## Warning
This operation is destructive and cannot be undone.

## Inputs
- Project Key
- Flag Key

## Output
Returns whether the flag was deleted, HTTP status code, and a status message.

A 404 response is handled as an expected "already deleted/not found" result.`
}

func (d *DeleteFlag) Icon() string {
	return "trash"
}

func (d *DeleteFlag) Color() string {
	return "#DC2626"
}

func (d *DeleteFlag) ExampleOutput() map[string]any {
	return deleteFlagExampleOutput()
}

func (d *DeleteFlag) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteFlag) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectKey",
			Label:       "Project Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Key of the LaunchDarkly project that owns the flag",
			Placeholder: "default",
		},
		{
			Name:        "flagKey",
			Label:       "Flag Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Unique key of the LaunchDarkly feature flag",
			Placeholder: "my-feature-flag",
		},
	}
}

func (d *DeleteFlag) Setup(ctx core.SetupContext) error {
	spec := DeleteFlagSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.ProjectKey) == "" {
		return fmt.Errorf("projectKey is required")
	}
	if strings.TrimSpace(spec.FlagKey) == "" {
		return fmt.Errorf("flagKey is required")
	}

	return nil
}

func (d *DeleteFlag) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteFlag) Execute(ctx core.ExecutionContext) error {
	spec := DeleteFlagSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	projectKey := strings.TrimSpace(spec.ProjectKey)
	flagKey := strings.TrimSpace(spec.FlagKey)

	if projectKey == "" {
		return fmt.Errorf("projectKey is required")
	}
	if flagKey == "" {
		return fmt.Errorf("flagKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create LaunchDarkly client: %w", err)
	}

	statusCode, _, err := client.DeleteFlag(projectKey, flagKey)
	if err != nil {
		if statusCode == http.StatusNotFound {
			output := DeleteFlagOutput{
				Deleted:    false,
				StatusCode: http.StatusNotFound,
				Message:    fmt.Sprintf("Feature flag %q was not found in project %q.", flagKey, projectKey),
			}
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteFlagPayloadType, []any{output})
		}
		return fmt.Errorf("failed to delete flag: %w", err)
	}

	output := DeleteFlagOutput{
		Deleted:    true,
		StatusCode: statusCode,
		Message:    fmt.Sprintf("Feature flag %q was successfully deleted from project %q.", flagKey, projectKey),
	}

	if statusCode != http.StatusNoContent && statusCode != http.StatusOK {
		output.Message = fmt.Sprintf("Delete request for feature flag %q returned status %d.", flagKey, statusCode)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteFlagPayloadType, []any{output})
}

func (d *DeleteFlag) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteFlag) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteFlag) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (d *DeleteFlag) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteFlag) Cleanup(ctx core.SetupContext) error {
	return nil
}
