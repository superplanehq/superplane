package runner

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type AgentCredentials struct {
	Source      string                       `mapstructure:"source"`
	Secret      configuration.SecretKeyRef   `mapstructure:"secret"`
	Integration configuration.IntegrationRef `mapstructure:"integration"`
}

func ValidateAgentCredentials(credentials AgentCredentials, integrationRequired bool) error {
	switch credentials.Source {
	case CredentialsSourceSecret:
		if !credentials.Secret.IsSet() {
			return fmt.Errorf("API key is required")
		}
		return nil
	case CredentialsSourceIntegration:
		if !integrationRequired {
			return fmt.Errorf("invalid credentials source: %s", credentials.Source)
		}
		if !credentials.Integration.IsSet() {
			return fmt.Errorf("integration is required")
		}
		return nil
	case CredentialsSourceHosted:
		return nil
	default:
		return fmt.Errorf("invalid credentials source: %s", credentials.Source)
	}
}

func InjectSecretAPIKey(ctx core.ExecutionContext, environment []BrokerEnvironmentVariable, envName string, secret configuration.SecretKeyRef) ([]BrokerEnvironmentVariable, error) {
	apiKey, err := ctx.Secrets.GetKey(secret.Secret, secret.Key)
	if err != nil {
		return nil, fmt.Errorf("resolve API key: %w", err)
	}
	return append(environment, BrokerEnvironmentVariable{Name: envName, Value: string(apiKey)}), nil
}

func InjectIntegrationKeys(ctx core.ExecutionContext, environment []BrokerEnvironmentVariable, integration configuration.IntegrationRef) ([]BrokerEnvironmentVariable, error) {
	keys, err := ctx.Secrets.GetIntegrationKeys(integration.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve integration: %w", err)
	}
	for name, value := range keys {
		environment = append(environment, BrokerEnvironmentVariable{Name: name, Value: string(value)})
	}
	return environment, nil
}

func PrepareHostedRun(ctx core.ExecutionContext, provider, model string) (core.HostedLLMAccess, error) {
	if ctx.HostedLLM == nil {
		return core.HostedLLMAccess{}, fmt.Errorf("hosted credentials are not available")
	}
	if err := ctx.HostedLLM.AssertCreditAvailable(); err != nil {
		return core.HostedLLMAccess{}, err
	}
	access, err := ctx.HostedLLM.Resolve(provider)
	if err != nil {
		return core.HostedLLMAccess{}, err
	}
	if strings.TrimSpace(model) == "" {
		return core.HostedLLMAccess{}, fmt.Errorf("model is required for SuperPlane-hosted credentials")
	}
	if !access.AllowsModel(model) {
		return core.HostedLLMAccess{}, fmt.Errorf("model %s is not on the SuperPlane-hosted allowlist", model)
	}
	return access, nil
}

func InjectHostedAPIKey(environment []BrokerEnvironmentVariable, envName, apiKey string, extra ...BrokerEnvironmentVariable) []BrokerEnvironmentVariable {
	environment = append(environment, BrokerEnvironmentVariable{Name: envName, Value: apiKey})
	return append(environment, extra...)
}
