package runner

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

type AgentCredentialsOptions struct {
	SecretLabel       string
	IntegrationName   string
	IntegrationLabel  string
	AllowHosted       bool
	HostedDescription string
}

func AgentMachineTypeField() configuration.Field {
	return configuration.Field{
		Name:     "machineType",
		Label:    "Machine type",
		Type:     configuration.FieldTypeSelect,
		Required: true,
		TypeOptions: &configuration.TypeOptions{
			Select: &configuration.SelectTypeOptions{
				Options: MachineTypeOptions(),
			},
		},
	}
}

func AgentCredentialsField(opts AgentCredentialsOptions) configuration.Field {
	options := []configuration.FieldOption{
		{Label: "Secret", Value: CredentialsSourceSecret},
	}
	if strings.TrimSpace(opts.IntegrationName) != "" {
		label := opts.IntegrationLabel
		if label == "" {
			label = "Integration"
		}
		options = append(options, configuration.FieldOption{Label: label, Value: CredentialsSourceIntegration})
	}
	if opts.AllowHosted {
		options = append(options, configuration.FieldOption{Label: "SuperPlane hosted", Value: CredentialsSourceHosted})
	}

	schema := []configuration.Field{
		{
			Name:     "source",
			Label:    "Source",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: options},
			},
		},
		{
			Name:  "secret",
			Label: opts.SecretLabel,
			Type:  configuration.FieldTypeSecretKey,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "source", Values: []string{CredentialsSourceSecret}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "source", Values: []string{CredentialsSourceSecret}},
			},
		},
	}
	if strings.TrimSpace(opts.IntegrationName) != "" {
		schema = append(schema, configuration.Field{
			Name:  "integration",
			Label: "Integration",
			Type:  configuration.FieldTypeIntegration,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "source", Values: []string{CredentialsSourceIntegration}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "source", Values: []string{CredentialsSourceIntegration}},
			},
			TypeOptions: &configuration.TypeOptions{
				Integration: &configuration.IntegrationTypeOptions{
					Integration: opts.IntegrationName,
				},
			},
		})
	}

	description := "API key or integration to use."
	if opts.AllowHosted {
		description = "Secret, integration, or SuperPlane-hosted credentials."
		if opts.HostedDescription != "" {
			description = opts.HostedDescription
		}
	}

	return configuration.Field{
		Name:        "credentials",
		Label:       "Credentials",
		Type:        configuration.FieldTypeObject,
		Required:    true,
		Description: description,
		TypeOptions: &configuration.TypeOptions{
			Object: &configuration.ObjectTypeOptions{Schema: schema},
		},
	}
}

func AgentModelField(provider, description, placeholder string) configuration.Field {
	return configuration.Field{
		Name:        "model",
		Label:       "Model",
		Type:        configuration.FieldTypeHostedModel,
		Required:    false,
		Description: description,
		Placeholder: placeholder,
		TypeOptions: &configuration.TypeOptions{
			HostedModel: &configuration.HostedModelTypeOptions{Provider: provider},
		},
	}
}

func AgentStepsField(description, promptPlaceholder, commandPlaceholder string) configuration.Field {
	return configuration.Field{
		Name:        "steps",
		Label:       "Steps",
		Type:        configuration.FieldTypeList,
		Required:    true,
		Default:     DefaultAgentSteps(),
		Description: description,
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel:   "Step",
				Accordion:   true,
				Reorderable: true,
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeObject,
					Schema: []configuration.Field{
						{
							Name:        "name",
							Label:       "Name",
							Type:        configuration.FieldTypeString,
							Required:    true,
							Placeholder: "e.g. Clone repo",
						},
						{
							Name:     "type",
							Label:    "Type",
							Type:     configuration.FieldTypeSelect,
							Required: true,
							Default:  AgentStepPrompt,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Prompt", Value: AgentStepPrompt, Description: "Run one agent turn"},
										{Label: "Bash", Value: AgentStepBash, Description: "Run shell commands on the runner"},
									},
								},
							},
						},
						{
							Name:        "prompt",
							Label:       "Prompt",
							Type:        configuration.FieldTypeText,
							Required:    false,
							Placeholder: promptPlaceholder,
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "type", Values: []string{AgentStepPrompt}},
							},
							RequiredConditions: []configuration.RequiredCondition{
								{Field: "type", Values: []string{AgentStepPrompt}},
							},
						},
						{
							Name:        "command",
							Label:       "Command",
							Type:        configuration.FieldTypeText,
							Required:    false,
							Placeholder: commandPlaceholder,
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "type", Values: []string{AgentStepBash}},
							},
							RequiredConditions: []configuration.RequiredCondition{
								{Field: "type", Values: []string{AgentStepBash}},
							},
							TypeOptions: &configuration.TypeOptions{
								Text: &configuration.TextTypeOptions{Language: "shell"},
							},
						},
					},
				},
			},
		},
	}
}

func AgentWorkingDirectoryField() configuration.Field {
	return configuration.Field{
		Name:        "workingDirectory",
		Label:       "Working directory",
		Type:        configuration.FieldTypeString,
		Required:    false,
		Description: "Optional starting directory.",
		Placeholder: "/tmp/repo",
	}
}

func AgentEnvironmentField(reservedEnv string) configuration.Field {
	description := "Optional key/value pairs passed into the agent environment"
	if reservedEnv != "" {
		description = fmt.Sprintf("Optional key/value pairs passed into the agent environment (in addition to %s)", reservedEnv)
	}
	return configuration.Field{
		Name:        "environment",
		Label:       "Environment variables",
		Type:        configuration.FieldTypeList,
		Required:    false,
		Description: description,
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel: "Variable",
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeObject,
					Schema: []configuration.Field{
						{
							Name:        "name",
							Label:       "Name",
							Type:        configuration.FieldTypeString,
							Description: "Environment variable name (letters, numbers, underscore)",
							Placeholder: "e.g. GITHUB_TOKEN",
							Required:    true,
						},
						{
							Name:        "valueSource",
							Label:       "Value source",
							Type:        configuration.FieldTypeSelect,
							Description: "Where this variable value comes from",
							Required:    true,
							Default:     EnvironmentValueSourceLiteral,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Literal value", Value: EnvironmentValueSourceLiteral},
										{Label: "Secret key", Value: EnvironmentValueSourceSecret},
									},
								},
							},
						},
						{
							Name:                 "value",
							Label:                "Value",
							Type:                 configuration.FieldTypeString,
							Description:          "Literal value. Supports expressions such as {{ previous().data.author.email }}",
							Placeholder:          "e.g. production",
							Required:             false,
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "valueSource", Values: []string{EnvironmentValueSourceLiteral}}},
							RequiredConditions:   []configuration.RequiredCondition{{Field: "valueSource", Values: []string{EnvironmentValueSourceLiteral}}},
						},
						{
							Name:                 "secret",
							Label:                "Secret key",
							Type:                 configuration.FieldTypeSecretKey,
							Description:          "Stored credential key to use as the variable value",
							Required:             false,
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "valueSource", Values: []string{EnvironmentValueSourceSecret}}},
							RequiredConditions:   []configuration.RequiredCondition{{Field: "valueSource", Values: []string{EnvironmentValueSourceSecret}}},
						},
					},
				},
			},
		},
	}
}

func AgentTimeoutField() configuration.Field {
	return configuration.Field{
		Name:        "executionTimeoutSeconds",
		Label:       "Execution timeout (seconds)",
		Type:        configuration.FieldTypeNumber,
		Required:    false,
		Default:     DefaultExecutionTimeoutSeconds,
		Description: "Hard time limit for the whole task, including all steps. Defaults to 3600 seconds (1 hour).",
		TypeOptions: &configuration.TypeOptions{
			Number: &configuration.NumberTypeOptions{
				Min: IntPtr(0),
				Max: IntPtr(MaxExecutionTimeoutSecondsRequest),
			},
		},
	}
}

func ValidateReservedEnvironmentName(environment []EnvironmentVariable, reserved string) error {
	for i, variable := range environment {
		if strings.TrimSpace(variable.Name) == reserved {
			return fmt.Errorf("environment[%d].name cannot be %s; use the credentials field", i, reserved)
		}
	}
	return nil
}
