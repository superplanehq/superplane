package github

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	githubmocks "github.com/superplanehq/superplane/test/support/mocks/github"
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
	assert.Contains(t, secrets.Usage, "https://github.com/<owner>/<repo>.git")
	assert.NotContains(t, secrets.Usage, "x-access-token")
	assert.NotContains(t, secrets.Usage, "ghp_test_token")
	assert.Equal(t, githubSetupName, secrets.SetupName)
	assert.Contains(t, secrets.Setup, "gh auth setup-git --hostname github.com --force")
	assert.NotContains(t, secrets.Setup, "x-access-token")
	assert.NotContains(t, secrets.Setup, "ghp_test_token")
}

func TestGitHub__ResolveSecrets__ScopesHostedToken(t *testing.T) {
	integrationCtx := githubmocks.IntegrationContextForLegacySetupFlow(githubPrivateKeyPEM(t))
	integrationCtx.Metadata = common.Metadata{
		InstallationID:       "67890",
		Repositories:         []common.Repository{{ID: 101, Name: "acme/api"}},
		SelectedRepositories: []common.Repository{{ID: 101, Name: "acme/api"}},
		RepositoryScoped:     true,
		GitHubApp:            common.GitHubAppMetadata{ID: 12345},
	}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		githubmocks.GitHubResponse(http.StatusCreated, `{"token":"scoped-token","expires_at":"2099-01-01T00:00:00Z"}`),
	}}

	secrets, err := (&GitHub{}).ResolveSecrets(core.IntegrationSecretContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, []byte("scoped-token"), secrets.Values[integrationSecretGitHubToken])
	require.Len(t, httpCtx.Requests, 1)
	requestBody, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	var request struct {
		RepositoryIDs []int64 `json:"repository_ids"`
	}
	require.NoError(t, json.Unmarshal(requestBody, &request))
	assert.Equal(t, []int64{101}, request.RepositoryIDs)
}
