package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestResolveSecrets(t *testing.T) {
	t.Parallel()

	secrets, err := (&Claude{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "sk-ant-test",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("sk-ant-test"), secrets.Values[integrationSecretAnthropicAPIKey])
	assert.Contains(t, secrets.Usage, "ANTHROPIC_API_KEY")
	assert.NotContains(t, secrets.Usage, "sk-ant-test")
	assert.Empty(t, secrets.Setup)
}
