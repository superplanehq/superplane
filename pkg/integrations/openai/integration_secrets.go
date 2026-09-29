package openai

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	integrationSecretOpenAIAPIKey = "OPENAI_API_KEY"
	openAISecretUsage             = `An OpenAI API key is available in the OPENAI_API_KEY environment variable.
Do not print the key.`
)

func (o *OpenAI) ResolveSecrets(ctx core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	apiKey, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return core.IntegrationSecrets{}, fmt.Errorf("failed to get API key: %w", err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return core.IntegrationSecrets{}, fmt.Errorf("apiKey is required")
	}

	return core.IntegrationSecrets{
		Values: map[string][]byte{
			integrationSecretOpenAIAPIKey: []byte(key),
		},
		Usage: openAISecretUsage,
	}, nil
}
