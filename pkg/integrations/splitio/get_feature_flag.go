package splitio

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
	WorkspaceID   string `json:"workspaceId" mapstructure:"workspaceId"`
	EnvironmentID string `json:"environmentId" mapstructure:"environmentId"`
	FlagName      string `json:"flagName" mapstructure:"flagName"`
}

func (c *GetFeatureFlag) Name() string {
	return "splitio.getFeatureFlag"
}

func (c *GetFeatureFlag) Label() string {
	return "Get Feature Flag"
}

func (c *GetFeatureFlag) Description() string {
	return "Get a feature flag definition from Split.io"
}

func (c *GetFeatureFlag) Documentation() string {
	return `The Get Feature Flag component retrieves a specific feature flag definition from Split.io for a given workspace and environment.

## Use Cases

- **Flag lookup**: Fetch flag details for processing or display
- **Workflow automation**: Get flag information to make decisions in workflows
- **Status checking**: Check if a flag is killed or active before performing actions
- **Audit and monitoring**: Retrieve flag data for compliance workflows

## Configuration

- **Workspace**: The Split.io workspace containing the flag
- **Environment**: The environment to get the flag definition from
- **Feature Flag**: The name of the feature flag to retrieve (supports expressions)

## Output

Returns the complete feature flag definition including:
- Flag name and description
- Treatments and their configurations
- Targeting rules
- Default rule and treatment
- Kill status and traffic allocation`
}

func (c *GetFeatureFlag) Icon() string {
	return "splitio"
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
			Name:        "workspaceId",
			Label:       "Workspace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Split.io workspace",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "workspace",
				},
			},
		},
		{
			Name:        "environmentId",
			Label:       "Environment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The environment to get the flag definition from",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "environment",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "workspaceId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "workspaceId"},
						},
					},
				},
			},
		},
		{
			Name:        "flagName",
			Label:       "Feature Flag",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The feature flag to retrieve",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "split",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "workspaceId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "workspaceId"},
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

	if strings.TrimSpace(spec.WorkspaceID) == "" {
		return errors.New("workspace is required")
	}

	if strings.TrimSpace(spec.EnvironmentID) == "" {
		return errors.New("environment is required")
	}

	if strings.TrimSpace(spec.FlagName) == "" {
		return errors.New("feature flag name is required")
	}

	return nil
}

func (c *GetFeatureFlag) Execute(ctx core.ExecutionContext) error {
	spec := GetFeatureFlagSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.WorkspaceID) == "" {
		return errors.New("workspace is required")
	}

	if strings.TrimSpace(spec.EnvironmentID) == "" {
		return errors.New("environment is required")
	}

	if strings.TrimSpace(spec.FlagName) == "" {
		return errors.New("feature flag name is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Split.io client: %w", err)
	}

	flag, err := client.GetSplitDefinition(spec.WorkspaceID, spec.FlagName, spec.EnvironmentID)
	if err != nil {
		return fmt.Errorf("failed to get feature flag: %w", err)
	}

	flag["workspaceId"] = spec.WorkspaceID
	flag["environmentId"] = spec.EnvironmentID

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"splitio.flag",
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
