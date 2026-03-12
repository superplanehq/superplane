package launchdarkly

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteFeatureFlag struct{}

type DeleteFeatureFlagSpec struct {
	ProjectKey string `json:"projectKey" mapstructure:"projectKey"`
	FlagKey    string `json:"flagKey" mapstructure:"flagKey"`
}

func (c *DeleteFeatureFlag) Name() string {
	return "launchdarkly.deleteFeatureFlag"
}

func (c *DeleteFeatureFlag) Label() string {
	return "Delete Feature Flag"
}

func (c *DeleteFeatureFlag) Description() string {
	return "Delete a feature flag from LaunchDarkly"
}

func (c *DeleteFeatureFlag) Documentation() string {
	return `The Delete Feature Flag component permanently deletes a feature flag from a LaunchDarkly project.

## Use Cases

- **Flag cleanup**: Remove stale or temporary flags after rollout is complete
- **Automated lifecycle**: Delete flags as part of a release workflow
- **Maintenance workflows**: Clean up archived flags that are no longer needed

## Configuration

- **Project Key**: The key of the LaunchDarkly project containing the flag
- **Flag Key**: The key of the feature flag to delete (supports expressions)

## Output

Returns a confirmation payload with the deleted flag's project and flag keys.

**Warning**: This action is irreversible. Once deleted, the flag and all its targeting rules are permanently removed.`
}

func (c *DeleteFeatureFlag) Icon() string {
	return "launchdarkly"
}

func (c *DeleteFeatureFlag) Color() string {
	return "gray"
}

func (c *DeleteFeatureFlag) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteFeatureFlag) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectKey",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The LaunchDarkly project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "flagKey",
			Label:       "Feature Flag",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The feature flag to delete",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "flag",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "projectKey",
							ValueFrom: &configuration.ParameterValueFrom{Field: "projectKey"},
						},
					},
				},
			},
		},
	}
}

func (c *DeleteFeatureFlag) Setup(ctx core.SetupContext) error {
	spec := DeleteFeatureFlagSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.ProjectKey) == "" {
		return errors.New("project key is required")
	}

	if strings.TrimSpace(spec.FlagKey) == "" {
		return errors.New("flag key is required")
	}

	return nil
}

func (c *DeleteFeatureFlag) Execute(ctx core.ExecutionContext) error {
	spec := DeleteFeatureFlagSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.ProjectKey) == "" {
		return errors.New("project key is required")
	}

	if strings.TrimSpace(spec.FlagKey) == "" {
		return errors.New("flag key is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create LaunchDarkly client: %w", err)
	}

	if err := client.DeleteFeatureFlag(spec.ProjectKey, spec.FlagKey); err != nil {
		return fmt.Errorf("failed to delete feature flag: %w", err)
	}

	result := map[string]any{
		"projectKey": spec.ProjectKey,
		"flagKey":    spec.FlagKey,
		"deleted":    true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"launchdarkly.flag.deleted",
		[]any{result},
	)
}

func (c *DeleteFeatureFlag) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteFeatureFlag) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteFeatureFlag) Actions() []core.Action {
	return nil
}

func (c *DeleteFeatureFlag) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteFeatureFlag) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteFeatureFlag) Cleanup(ctx core.SetupContext) error {
	return nil
}
