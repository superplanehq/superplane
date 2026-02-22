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

const GetFlagPayloadType = "launchdarkly.getFlag.result"

type GetFlag struct{}

type GetFlagSpec struct {
	ProjectKey string `json:"projectKey" mapstructure:"projectKey"`
	FlagKey    string `json:"flagKey" mapstructure:"flagKey"`
}

type GetFlagOutput struct {
	Found   bool           `json:"found"`
	Flag    map[string]any `json:"flag,omitempty"`
	Message string         `json:"message"`
}

func (g *GetFlag) Name() string {
	return "launchdarkly.getFlag"
}

func (g *GetFlag) Label() string {
	return "Get Feature Flag"
}

func (g *GetFlag) Description() string {
	return "Retrieve a LaunchDarkly feature flag by project key and flag key"
}

func (g *GetFlag) Documentation() string {
	return `Fetches details of a LaunchDarkly feature flag.

## Inputs
- **Project Key**: LaunchDarkly project key (for example: default)
- **Flag Key**: Feature flag key

## Output
Returns:
- found: true when flag exists
- flag: full LaunchDarkly flag payload
- message: human-readable status

A 404 response is treated as a valid "not found" result.`
}

func (g *GetFlag) Icon() string {
	return "search"
}

func (g *GetFlag) Color() string {
	return "#2B7FFF"
}

func (g *GetFlag) ExampleOutput() map[string]any {
	return getFlagExampleOutput()
}

func (g *GetFlag) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetFlag) Configuration() []configuration.Field {
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

func (g *GetFlag) Setup(ctx core.SetupContext) error {
	spec := GetFlagSpec{}
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

func (g *GetFlag) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetFlag) Execute(ctx core.ExecutionContext) error {
	spec := GetFlagSpec{}
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

	statusCode, flag, _, err := client.GetFlag(projectKey, flagKey)
	if err != nil {
		if statusCode == http.StatusNotFound {
			output := GetFlagOutput{
				Found:   false,
				Message: fmt.Sprintf("Feature flag %q was not found in project %q.", flagKey, projectKey),
			}
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetFlagPayloadType, []any{output})
		}
		return fmt.Errorf("failed to get flag: %w", err)
	}

	output := GetFlagOutput{
		Found:   true,
		Flag:    flag,
		Message: fmt.Sprintf("Successfully retrieved feature flag %q.", flagKey),
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetFlagPayloadType, []any{output})
}

func (g *GetFlag) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetFlag) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetFlag) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (g *GetFlag) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetFlag) Cleanup(ctx core.SetupContext) error {
	return nil
}
