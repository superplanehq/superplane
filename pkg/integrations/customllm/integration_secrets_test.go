package customllm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestIntegrationNameMatchesBYOKApp(t *testing.T) {
	assert.Equal(t, models.CustomLLMAppName, (&CustomLLM{}).Name())
}

func TestResolveSecrets(t *testing.T) {
	secrets, err := (&CustomLLM{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":  "sk-live-secret",
				"baseURL": "https://models.example/v1/",
				"apiType": "openai-compatible",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("sk-live-secret"), secrets.Values[secretAPIKey])
	assert.Equal(t, []byte("https://models.example/v1"), secrets.Values[secretBaseURL])
	assert.Equal(t, []byte(APITypeOpenAICompatible), secrets.Values[secretAPIType])
	assert.NotContains(t, secrets.Usage, "sk-live-secret")
}
