package runner

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	EnvironmentFromSourceIntegration = "integration"
	EnvironmentFromSourceSecret      = "secret"

	EnvironmentValueSourceLiteral = "literal"
	EnvironmentValueSourceSecret  = "secret"
)

type EnvironmentFromEntry struct {
	Source      string                       `json:"source" mapstructure:"source"`
	Integration configuration.IntegrationRef `json:"integration,omitempty" mapstructure:"integration"`
	Secret      configuration.SecretRef      `json:"secret,omitempty" mapstructure:"secret"`
}

func EnvironmentFromConfigurationField() configuration.Field {
	return environmentFromConfigurationField()
}

func environmentFromConfigurationField() configuration.Field {
	return configuration.Field{
		Name:        "environmentFrom",
		Label:       "Environment from",
		Type:        configuration.FieldTypeList,
		Required:    false,
		Description: "Import environment variables from connected integrations or organization secrets",
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel: "Source",
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeObject,
					Schema: []configuration.Field{
						{
							Name:        "source",
							Label:       "Source",
							Type:        configuration.FieldTypeSelect,
							Required:    true,
							Default:     EnvironmentFromSourceIntegration,
							Description: "Where imported environment variables come from",
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Integration", Value: EnvironmentFromSourceIntegration},
										{Label: "Secret", Value: EnvironmentFromSourceSecret},
									},
								},
							},
						},
						{
							Name:        "integration",
							Label:       "Integration",
							Type:        configuration.FieldTypeIntegration,
							Required:    false,
							Description: "Name of the integration",
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "source", Values: []string{EnvironmentFromSourceIntegration}},
							},
							RequiredConditions: []configuration.RequiredCondition{
								{Field: "source", Values: []string{EnvironmentFromSourceIntegration}},
							},
						},
						{
							Name:        "secret",
							Label:       "Secret",
							Type:        configuration.FieldTypeSecret,
							Required:    false,
							Description: "Organization secret to import all keys from",
							Placeholder: "e.g. deploy-credentials",
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "source", Values: []string{EnvironmentFromSourceSecret}},
							},
							RequiredConditions: []configuration.RequiredCondition{
								{Field: "source", Values: []string{EnvironmentFromSourceSecret}},
							},
						},
					},
				},
			},
		},
	}
}

func ValidateEnvironmentFrom(environmentFrom []EnvironmentFromEntry) error {
	seenIntegrations := make(map[string]struct{}, len(environmentFrom))
	seenSecrets := make(map[string]struct{}, len(environmentFrom))

	for i, entry := range environmentFrom {
		source := strings.TrimSpace(entry.Source)
		if source == "" {
			return fmt.Errorf("environmentFrom[%d].source is required", i)
		}

		switch source {
		case EnvironmentFromSourceIntegration:
			if !entry.Integration.IsSet() {
				return fmt.Errorf("environmentFrom[%d].integration is required", i)
			}

			name := strings.TrimSpace(entry.Integration.Name)
			if _, ok := seenIntegrations[name]; ok {
				return fmt.Errorf("duplicate environmentFrom integration: %s", name)
			}
			seenIntegrations[name] = struct{}{}

		case EnvironmentFromSourceSecret:
			if !entry.Secret.IsSet() {
				return fmt.Errorf("environmentFrom[%d].secret is required", i)
			}

			secretName := strings.TrimSpace(entry.Secret.Secret)
			if _, ok := seenSecrets[secretName]; ok {
				return fmt.Errorf("duplicate environmentFrom secret: %s", secretName)
			}
			seenSecrets[secretName] = struct{}{}

		default:
			return fmt.Errorf("invalid environmentFrom[%d].source: %s", i, entry.Source)
		}
	}

	return nil
}

func ResolveEnvironment(
	secrets core.SecretsContext,
	environmentFrom []EnvironmentFromEntry,
	environment []EnvironmentVariable,
) ([]BrokerEnvironmentVariable, error) {
	resolved := make([]BrokerEnvironmentVariable, 0)
	seen := make(map[string]struct{})

	for _, entry := range environmentFrom {
		switch strings.TrimSpace(entry.Source) {
		case EnvironmentFromSourceIntegration:
			if secrets == nil {
				return nil, fmt.Errorf("failed to resolve environmentFrom integration secrets: secrets context is unavailable")
			}

			keys, err := secrets.GetIntegrationKeys(strings.TrimSpace(entry.Integration.Name))
			if err != nil {
				return nil, fmt.Errorf("failed to resolve environmentFrom integration secrets: %w", err)
			}

			if err := appendImportedEnvironmentVariables(&resolved, seen, keys); err != nil {
				return nil, err
			}

		case EnvironmentFromSourceSecret:
			if secrets == nil {
				return nil, fmt.Errorf("failed to resolve environmentFrom secret keys: secrets context is unavailable")
			}

			keys, err := secrets.GetSecretKeys(entry.Secret.Secret)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve environmentFrom secret keys: %w", err)
			}

			if err := appendImportedEnvironmentVariables(&resolved, seen, keys); err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("invalid environmentFrom source: %s", entry.Source)
		}
	}

	explicit, err := resolveExplicitEnvironment(secrets, environment)
	if err != nil {
		return nil, err
	}

	for _, variable := range explicit {
		if _, ok := seen[variable.Name]; ok {
			for i := range resolved {
				if resolved[i].Name == variable.Name {
					resolved[i] = variable
					break
				}
			}
			continue
		}

		seen[variable.Name] = struct{}{}
		resolved = append(resolved, variable)
	}

	return resolved, nil
}

func appendImportedEnvironmentVariables(
	resolved *[]BrokerEnvironmentVariable,
	seen map[string]struct{},
	keys map[string][]byte,
) error {
	for name, value := range keys {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		if !environmentVariableNameRegex.MatchString(name) {
			return fmt.Errorf("invalid environment variable name: %s", name)
		}

		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate environment variable name: %s", name)
		}

		seen[name] = struct{}{}
		*resolved = append(*resolved, BrokerEnvironmentVariable{
			Name:  name,
			Value: string(value),
		})
	}

	return nil
}

func ValidateEnvironment(environment []EnvironmentVariable) error {
	seen := make(map[string]struct{}, len(environment))

	for i, variable := range environment {
		name := strings.TrimSpace(variable.Name)
		if name == "" {
			return fmt.Errorf("environment[%d].name is required", i)
		}

		if !environmentVariableNameRegex.MatchString(name) {
			return fmt.Errorf("invalid environment variable name: %s", variable.Name)
		}

		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate environment variable name: %s", name)
		}
		seen[name] = struct{}{}

		switch strings.TrimSpace(variable.ValueSource) {
		case EnvironmentValueSourceLiteral:
			if variable.Value == nil {
				return fmt.Errorf("environment[%d].value is required for literal environment variables", i)
			}

		case EnvironmentValueSourceSecret:
			if !variable.Secret.IsSet() {
				return fmt.Errorf("environment[%d].secret.secret and environment[%d].secret.key are required", i, i)
			}

		case "":
			return fmt.Errorf("environment[%d].valueSource is required", i)

		default:
			return fmt.Errorf("invalid environment variable value source: %s", variable.ValueSource)
		}
	}

	return nil
}

func resolveExplicitEnvironment(secrets core.SecretsContext, environment []EnvironmentVariable) ([]BrokerEnvironmentVariable, error) {
	if len(environment) == 0 {
		return nil, nil
	}

	resolved := make([]BrokerEnvironmentVariable, 0, len(environment))
	for _, variable := range environment {
		name := strings.TrimSpace(variable.Name)

		switch strings.TrimSpace(variable.ValueSource) {
		case EnvironmentValueSourceLiteral:
			resolved = append(resolved, BrokerEnvironmentVariable{
				Name:  name,
				Value: *variable.Value,
			})

		case EnvironmentValueSourceSecret:
			if secrets == nil {
				return nil, fmt.Errorf("failed to resolve environment variable %s: secrets context is unavailable", name)
			}

			value, err := secrets.GetKey(variable.Secret.Secret, variable.Secret.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve environment variable %s secret %s/%s: %w", name, variable.Secret.Secret, variable.Secret.Key, err)
			}

			resolved = append(resolved, BrokerEnvironmentVariable{
				Name:  name,
				Value: string(value),
			})
		}
	}

	return resolved, nil
}
