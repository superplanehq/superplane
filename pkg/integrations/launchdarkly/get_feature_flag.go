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

type GetFeatureFlag struct{}

type GetFeatureFlagSpec struct {
	ProjectKey string `json:"projectKey" mapstructure:"projectKey"`
	FlagKey    string `json:"flagKey" mapstructure:"flagKey"`
}

func (c *GetFeatureFlag) Name() string {
	return "launchdarkly.getFeatureFlag"
}

func (c *GetFeatureFlag) Label() string {
	return "Get Feature Flag"
}

func (c *GetFeatureFlag) Description() string {
	return "Get a feature flag from LaunchDarkly"
}

func (c *GetFeatureFlag) Documentation() string {
	return `The Get Feature Flag component retrieves a specific feature flag from a LaunchDarkly project.

## Use Cases

- **Flag lookup**: Fetch flag details for processing or display
- **Workflow automation**: Get flag information to make decisions in workflows
- **Status checking**: Check flag status before performing actions
- **Audit and monitoring**: Retrieve flag data for compliance workflows

## Configuration

- **Project Key**: The key of the LaunchDarkly project containing the flag
- **Flag Key**: The key of the feature flag to retrieve (supports expressions)

## Output

Returns the complete feature flag object including:
- Flag key, name, and description
- Kind (boolean, multivariate)
- Creation date
- Archived and temporary status
- Variations, environments, and targeting rules`
}

func (c *GetFeatureFlag) Icon() string {
	return "launchdarkly"
}

func (c *GetFeatureFlag) Color() string {
	return "gray"
}

func (c *GetFeatureFlag) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetFeatureFlag) Configuration() []configuration.Field {
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
			Description: "The feature flag to retrieve",
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

func (c *GetFeatureFlag) Setup(ctx core.SetupContext) error {
	spec := GetFeatureFlagSpec{}
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

func (c *GetFeatureFlag) Execute(ctx core.ExecutionContext) error {
	spec := GetFeatureFlagSpec{}
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

	flag, err := client.GetFeatureFlag(spec.ProjectKey, spec.FlagKey)
	if err != nil {
		return fmt.Errorf("failed to get feature flag: %w", err)
	}

	flag["projectKey"] = spec.ProjectKey

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"launchdarkly.flag",
		[]any{flag},
	)
}

func (c *GetFeatureFlag) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetFeatureFlag) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetFeatureFlag) Actions() []core.Action {
	return nil
}

func (c *GetFeatureFlag) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetFeatureFlag) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetFeatureFlag) Cleanup(ctx core.SetupContext) error {
	return nil
}
