package vercel

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListDeployments struct{}

type ListDeploymentsConfiguration struct {
	Project string `json:"project" mapstructure:"project"`
	Target  string `json:"target" mapstructure:"target"`
	State   string `json:"state" mapstructure:"state"`
	Limit   int    `json:"limit" mapstructure:"limit"`
}

func (c *ListDeployments) Name() string {
	return "vercel.listDeployments"
}

func (c *ListDeployments) Label() string {
	return "List Deployments"
}

func (c *ListDeployments) Description() string {
	return "List Vercel deployments, optionally filtered by project, target, and state"
}

func (c *ListDeployments) Documentation() string {
	return `The List Deployments component fetches recent Vercel deployments.

## Use Cases

- **Release audits**: Fetch recent production deployments for reporting
- **Find rollbacks**: Locate the last good deployment before rolling back
- **Stuck build checks**: Look for deployments stuck in a building state

## Configuration

- **Project**: Optional. Leave empty to list deployments across all projects.
- **Target**: Optional. Filter by production or preview deployments.
- **State**: Optional. Filter by deployment state (e.g. READY, ERROR).
- **Limit**: Maximum number of deployments to return. Defaults to 20.

## Output

Emits a ` + "`vercel.deployments`" + ` payload with a ` + "`deployments`" + ` array and the total ` + "`count`" + `.`
}

func (c *ListDeployments) Icon() string {
	return "list"
}

func (c *ListDeployments) Color() string {
	return "gray"
}

func (c *ListDeployments) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func stateOptions() []configuration.FieldOption {
	options := make([]configuration.FieldOption, 0, len(deploymentStateOptions))
	for _, state := range deploymentStateOptions {
		options = append(options, configuration.FieldOption{Label: capitalize(state), Value: state})
	}
	return options
}

// capitalize turns "READY" into "Ready".
func capitalize(value string) string {
	if value == "" {
		return value
	}

	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}

func (c *ListDeployments) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
			Description: "Optional project filter. Leave empty to list deployments for all projects.",
		},
		{
			Name:        "target",
			Label:       "Target",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Optional deployment environment filter",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: deploymentTargetOptions,
				},
			},
		},
		{
			Name:        "state",
			Label:       "State",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Optional deployment state filter",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: stateOptions(),
				},
			},
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     20,
			Description: "Maximum number of deployments to return (1-100)",
		},
	}
}

func decodeListDeploymentsConfiguration(config any) (ListDeploymentsConfiguration, error) {
	spec := ListDeploymentsConfiguration{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	spec.Target = strings.TrimSpace(strings.ToLower(spec.Target))
	spec.State = strings.TrimSpace(strings.ToUpper(spec.State))

	validTarget := slices.ContainsFunc(deploymentTargetOptions, func(option configuration.FieldOption) bool {
		return option.Value == spec.Target
	})
	if spec.Target != "" && !validTarget {
		return spec, fmt.Errorf("target must be one of: production, preview")
	}

	if spec.State != "" && !slices.Contains(deploymentStateOptions, spec.State) {
		return spec, fmt.Errorf("state must be one of: %s", strings.Join(deploymentStateOptions, ", "))
	}

	if spec.Limit == 0 {
		spec.Limit = 20
	}

	if spec.Limit < 0 || spec.Limit > 100 {
		return spec, fmt.Errorf("limit must be between 0 and 100")
	}

	return spec, nil
}

func (c *ListDeployments) Setup(ctx core.SetupContext) error {
	_, err := decodeListDeploymentsConfiguration(ctx.Configuration)
	return err
}

func (c *ListDeployments) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeListDeploymentsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deployments, err := client.ListDeployments(spec.Project, spec.Target, spec.State, spec.Limit)
	if err != nil {
		return fmt.Errorf("failed to list Vercel deployments: %w", err)
	}

	items := make([]map[string]any, 0, len(deployments))
	for _, deployment := range deployments {
		items = append(items, deploymentData(&deployment))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ListDeploymentsPayloadType,
		[]any{map[string]any{"deployments": items, "count": len(items)}},
	)
}

func (c *ListDeployments) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ListDeployments) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListDeployments) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ListDeployments) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ListDeployments) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
