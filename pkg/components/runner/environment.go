package runner

import (
	"fmt"
	"slices"
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

// Runner tasks execute with a terminal attached, so commands like `git log`
// send their output to a pager that then waits for a key press and blocks the
// task until it times out. Task environments disable paging by default.
var pagerDefaults = []BrokerEnvironmentVariable{
	{Name: "GIT_PAGER", Value: "cat"},
	{Name: "PAGER", Value: "cat"},
}

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

type IntegrationSetup struct {
	Name   string
	Script string
}

type ResolvedEnvironment struct {
	Variables []BrokerEnvironmentVariable
	Usage     string
	Setups    []IntegrationSetup
}

func ResolveEnvironment(
	secrets core.SecretsContext,
	environmentFrom []EnvironmentFromEntry,
	environment []EnvironmentVariable,
) (ResolvedEnvironment, error) {
	resolved := make([]BrokerEnvironmentVariable, 0)
	seen := make(map[string]struct{})
	usages := make([]string, 0)
	setups := make([]IntegrationSetup, 0)

	for _, entry := range environmentFrom {
		switch strings.TrimSpace(entry.Source) {
		case EnvironmentFromSourceIntegration:
			if secrets == nil {
				return ResolvedEnvironment{}, fmt.Errorf("failed to resolve environmentFrom integration secrets: secrets context is unavailable")
			}

			name := strings.TrimSpace(entry.Integration.Name)
			imported, err := secrets.GetIntegrationSecrets(name)
			if err != nil {
				return ResolvedEnvironment{}, fmt.Errorf("failed to resolve environmentFrom integration secrets: %w", err)
			}

			if err := appendImportedEnvironmentVariables(&resolved, seen, imported.Values); err != nil {
				return ResolvedEnvironment{}, err
			}

			if usage := strings.TrimSpace(imported.Usage); usage != "" {
				usages = append(usages, usage)
			}
			if setup := strings.TrimSpace(imported.Setup); setup != "" {
				setupName := strings.TrimSpace(imported.SetupName)
				if setupName == "" {
					setupName = "Set up " + name
				}
				setups = append(setups, IntegrationSetup{Name: setupName, Script: setup})
			}

		case EnvironmentFromSourceSecret:
			if secrets == nil {
				return ResolvedEnvironment{}, fmt.Errorf("failed to resolve environmentFrom secret keys: secrets context is unavailable")
			}

			keys, err := secrets.GetSecretKeys(entry.Secret.Secret)
			if err != nil {
				return ResolvedEnvironment{}, fmt.Errorf("failed to resolve environmentFrom secret keys: %w", err)
			}

			if err := appendImportedEnvironmentVariables(&resolved, seen, keys); err != nil {
				return ResolvedEnvironment{}, err
			}

		default:
			return ResolvedEnvironment{}, fmt.Errorf("invalid environmentFrom source: %s", entry.Source)
		}
	}

	explicit, err := resolveExplicitEnvironment(secrets, environment)
	if err != nil {
		return ResolvedEnvironment{}, err
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

	return ResolvedEnvironment{
		Variables: prependPagerDefaults(resolved),
		Usage:     strings.Join(usages, "\n\n"),
		Setups:    setups,
	}, nil
}

// prependPagerDefaults keeps the configured environment authoritative: a
// variable set by the node, an integration, or a secret is never replaced.
func prependPagerDefaults(environment []BrokerEnvironmentVariable) []BrokerEnvironmentVariable {
	defaults := make([]BrokerEnvironmentVariable, 0, len(pagerDefaults))
	for _, variable := range pagerDefaults {
		configured := slices.ContainsFunc(environment, func(existing BrokerEnvironmentVariable) bool {
			return existing.Name == variable.Name
		})
		if configured {
			continue
		}

		defaults = append(defaults, variable)
	}

	return append(defaults, environment...)
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
