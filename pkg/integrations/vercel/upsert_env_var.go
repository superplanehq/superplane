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

type UpsertEnvVar struct{}

type UpsertEnvVarConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Key     string   `json:"key" mapstructure:"key"`
	Value   string   `json:"value" mapstructure:"value"`
	Targets []string `json:"targets" mapstructure:"targets"`
	VarType string   `json:"varType" mapstructure:"varType"`
}

func (c *UpsertEnvVar) Name() string {
	return "vercel.upsertEnvVar"
}

func (c *UpsertEnvVar) Label() string {
	return "Set Environment Variable"
}

func (c *UpsertEnvVar) Description() string {
	return "Create or update an environment variable on a Vercel project"
}

func (c *UpsertEnvVar) Documentation() string {
	return `The Set Environment Variable component creates or updates an environment variable on a Vercel project.

## Use Cases

- **Secret rotation**: Rotate third-party API keys across environments
- **Config propagation**: Point projects at new infrastructure endpoints
- **Environment provisioning**: Set variables right after creating a project

## Configuration

- **Project**: Required. The Vercel project.
- **Key**: Required. The variable name (e.g. ` + "`API_URL`" + `).
- **Value**: Required. The variable value.
- **Environments**: Which environments the variable applies to. Defaults to production only.
- **Type**: How Vercel stores the value. Defaults to encrypted. Use sensitive for values that must never be readable after save.

## Notes

- If a variable with the same key already exists, its value is updated (upsert)
- New values apply to future deployments, not running ones`
}

func (c *UpsertEnvVar) Icon() string {
	return "key"
}

func (c *UpsertEnvVar) Color() string {
	return "gray"
}

func (c *UpsertEnvVar) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpsertEnvVar) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "project"}},
			Description: "Vercel project to set the variable on",
		},
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., API_URL",
			Description: "Name of the environment variable",
		},
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Sensitive:   true,
			Description: "Value of the environment variable",
		},
		{
			Name:     "targets",
			Label:    "Environments",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			Default:  []string{"production"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Production", Value: "production"},
						{Label: "Preview", Value: "preview"},
						{Label: "Development", Value: "development"},
					},
				},
			},
			Description: "Environments this variable applies to",
		},
		{
			Name:        "varType",
			Label:       "Type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "encrypted",
			Description: "How Vercel stores the value",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: envTypes,
				},
			},
		},
	}
}

func decodeUpsertEnvVarConfiguration(config any) (UpsertEnvVarConfiguration, error) {
	spec := UpsertEnvVarConfiguration{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	spec.Key = strings.TrimSpace(spec.Key)
	spec.VarType = strings.TrimSpace(strings.ToLower(spec.VarType))

	targets := make([]string, 0, len(spec.Targets))
	for _, target := range spec.Targets {
		trimmed := strings.TrimSpace(strings.ToLower(target))
		if trimmed == "" || slices.Contains(targets, trimmed) {
			continue
		}
		if !slices.Contains(envTargets, trimmed) {
			return spec, fmt.Errorf("environments must be one of: %s", strings.Join(envTargets, ", "))
		}
		targets = append(targets, trimmed)
	}
	spec.Targets = targets

	if spec.Project == "" {
		return spec, fmt.Errorf("project is required")
	}
	if spec.Key == "" {
		return spec, fmt.Errorf("key is required")
	}
	if len(spec.Targets) == 0 {
		spec.Targets = []string{"production"}
	}
	if spec.VarType == "" {
		spec.VarType = "encrypted"
	}
	if !slices.ContainsFunc(envTypes, func(option configuration.FieldOption) bool {
		return option.Value == spec.VarType
	}) {
		return spec, fmt.Errorf("type must be one of: encrypted, plain, sensitive")
	}

	return spec, nil
}

func (c *UpsertEnvVar) Setup(ctx core.SetupContext) error {
	_, err := decodeUpsertEnvVarConfiguration(ctx.Configuration)
	return err
}

func (c *UpsertEnvVar) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeUpsertEnvVarConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	envVar, err := client.UpsertEnvVar(spec.Project, UpsertEnvVarRequest{
		Key:     spec.Key,
		Value:   spec.Value,
		Type:    spec.VarType,
		Targets: spec.Targets,
	})
	if err != nil {
		return err
	}

	data := map[string]any{
		"projectId": spec.Project,
		"key":       spec.Key,
		"target":    spec.Targets,
	}
	if envVar != nil && envVar.EnvID != "" {
		data["envId"] = envVar.EnvID
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		EnvVarPayloadType,
		[]any{data},
	)
}

func (c *UpsertEnvVar) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpsertEnvVar) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpsertEnvVar) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpsertEnvVar) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpsertEnvVar) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
