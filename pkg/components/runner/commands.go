package runner

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	EnvironmentValueSourceLiteral = "literal"
	EnvironmentValueSourceSecret  = "secret"
)

var environmentVariableNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func normalizeCommands(commands string) []string {
	lines := strings.Split(commands, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		out = append(out, l)
	}
	return out
}

func normalizeSetupCommands(commands string) []string {
	commands = strings.TrimSpace(commands)
	if commands == "" {
		return nil
	}

	return []string{commands}
}

func validateCommands(commands string) error {
	lines := normalizeCommands(commands)
	if len(lines) == 0 {
		return errors.New("at least one command is required")
	}
	return nil
}

func validateEnvironment(environment []EnvironmentVariable) error {
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

func resolveEnvironment(secrets core.SecretsContext, environment []EnvironmentVariable) ([]BrokerEnvironmentVariable, error) {
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
