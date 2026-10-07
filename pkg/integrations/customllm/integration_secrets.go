package customllm

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	secretAPIKey  = "CUSTOM_LLM_API_KEY"
	secretBaseURL = "CUSTOM_LLM_BASE_URL"
	secretAPIType = "CUSTOM_LLM_API_TYPE"
	secretUsage   = `A provider token is available in the CUSTOM_LLM_API_KEY environment variable.
The API URL is in CUSTOM_LLM_BASE_URL. The API type is in CUSTOM_LLM_API_TYPE.
Do not print the token.`
)

func (c *CustomLLM) ResolveSecrets(ctx core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	apiKey, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return core.IntegrationSecrets{}, fmt.Errorf("failed to get API token: %w", err)
	}
	baseURL, err := ctx.Integration.GetConfig("baseURL")
	if err != nil {
		return core.IntegrationSecrets{}, fmt.Errorf("failed to get API URL: %w", err)
	}
	apiType, err := ctx.Integration.GetConfig("apiType")
	if err != nil {
		return core.IntegrationSecrets{}, fmt.Errorf("failed to get API type: %w", err)
	}
	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return core.IntegrationSecrets{}, fmt.Errorf("apiKey is required")
	}
	url := strings.TrimRight(strings.TrimSpace(string(baseURL)), "/")
	if url == "" {
		return core.IntegrationSecrets{}, fmt.Errorf("baseURL is required")
	}
	normalizedType, err := NormalizeAPIType(string(apiType))
	if err != nil {
		return core.IntegrationSecrets{}, err
	}
	return core.IntegrationSecrets{
		Values: map[string][]byte{
			secretAPIKey:  []byte(key),
			secretBaseURL: []byte(url),
			secretAPIType: []byte(normalizedType),
		},
		Usage: secretUsage,
	}, nil
}
