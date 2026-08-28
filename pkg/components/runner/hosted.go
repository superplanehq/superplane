package runner

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/llm"
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
	if strings.TrimSpace(model) == "" {
		return core.HostedLLMAccess{}, fmt.Errorf("model is required for SuperPlane-hosted credentials")
	}
	if err := ctx.HostedLLM.AssertModelSelectable(provider, "hosted", model); err != nil {
		return core.HostedLLMAccess{}, err
	}
	access, err := ctx.HostedLLM.Resolve(provider)
	if err != nil {
		return core.HostedLLMAccess{}, err
	}
	if !access.AllowsModel(model) {
		return core.HostedLLMAccess{}, fmt.Errorf("model %s is not on the SuperPlane-hosted allowlist", model)
	}
	if err := llm.ValidateBaseURL(access.BaseURL); err != nil {
		return core.HostedLLMAccess{}, err
	}
	return access, nil
}

func InjectHostedAPIKey(environment []BrokerEnvironmentVariable, envName, apiKey string, extra ...BrokerEnvironmentVariable) []BrokerEnvironmentVariable {
	environment = append(environment, BrokerEnvironmentVariable{Name: envName, Value: apiKey})
	return append(environment, extra...)
}

func InjectHostedCredentials(environment []BrokerEnvironmentVariable, apiKeyEnv, apiKey, baseURLEnv, baseURL string) []BrokerEnvironmentVariable {
	environment = dropEnvironmentNames(environment, apiKeyEnv, baseURLEnv)
	extra := []BrokerEnvironmentVariable{}
	if trimmed := strings.TrimSpace(baseURL); trimmed != "" {
		extra = append(extra, BrokerEnvironmentVariable{Name: baseURLEnv, Value: strings.TrimRight(trimmed, "/")})
	}
	return InjectHostedAPIKey(environment, apiKeyEnv, apiKey, extra...)
}

// ValidateHostedAgentSpec rejects hosted nodes that omit the model or try to
// override reserved provider env vars. environmentFrom can still import those
// names from a secret; InjectHostedCredentials strips them at execute time.
func ValidateHostedAgentSpec(credentials AgentCredentials, model string, environment []EnvironmentVariable, reservedEnvNames ...string) error {
	if !IsHostedCredentials(credentials.Source) {
		return nil
	}
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required for SuperPlane-hosted credentials")
	}
	return ValidateReservedEnvironmentNames(environment, reservedEnvNames...)
}

func dropEnvironmentNames(environment []BrokerEnvironmentVariable, names ...string) []BrokerEnvironmentVariable {
	deny := make(map[string]struct{}, len(names))
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			deny[trimmed] = struct{}{}
		}
	}
	if len(deny) == 0 {
		return environment
	}

	kept := make([]BrokerEnvironmentVariable, 0, len(environment))
	for _, variable := range environment {
		if _, drop := deny[variable.Name]; drop {
			continue
		}
		kept = append(kept, variable)
	}
	return kept
}
