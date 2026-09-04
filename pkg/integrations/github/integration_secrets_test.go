package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestGitHub__ResolveSecrets__PAT(t *testing.T) {
	t.Parallel()

	integrationCtx := &contexts.IntegrationContext{
		NewSetupFlow: true,
		CurrentProperties: map[string]any{
			common.PropertyAuthMethod: common.AuthMethodPAT,
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			common.SecretPAT: {Name: common.SecretPAT, Value: []byte("ghp_test_token")},
		},
	}

	secrets, err := (&GitHub{}).ResolveSecrets(core.IntegrationSecretContext{
		HTTP:        &contexts.HTTPContext{},
		Integration: integrationCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("ghp_test_token"), secrets.Values[integrationSecretGitHubToken])
	assert.Contains(t, secrets.Usage, "GITHUB_TOKEN")
	assert.Contains(t, secrets.Usage, "The gh CLI is already installed")
	assert.Contains(t, secrets.Usage, "Do not download or install gh")
	assert.NotContains(t, secrets.Usage, "ghp_test_token")
	assert.Empty(t, secrets.Setup)
}
