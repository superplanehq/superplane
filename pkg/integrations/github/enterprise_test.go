package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/test/support/contexts"
	"google.golang.org/grpc/codes"
)

func TestHostAllowed(t *testing.T) {
	host, err := common.ParseHost("acme.ghe.com")
	require.NoError(t, err)

	t.Run("empty allowlist accepts any host", func(t *testing.T) {
		t.Setenv(config.EnvGitHubEnterpriseAllowedHosts, "")
		assert.True(t, hostAllowed(host))
		assert.True(t, hostAllowed(common.DotCom()))
	})

	t.Run("exact hostname", func(t *testing.T) {
		t.Setenv(config.EnvGitHubEnterpriseAllowedHosts, "acme.ghe.com,other.ghe.com")
		assert.True(t, hostAllowed(host))
		other, err := common.ParseHost("blocked.ghe.com")
		require.NoError(t, err)
		assert.False(t, hostAllowed(other))
		assert.True(t, hostAllowed(common.DotCom()))
	})

	t.Run("wildcard", func(t *testing.T) {
		t.Setenv(config.EnvGitHubEnterpriseAllowedHosts, "*.ghe.com")
		assert.True(t, hostAllowed(host))
		assert.False(t, hostnameMatchesAllowlist("ghe.com", "*.ghe.com"))
	})
}

func TestProbeGitHubHost(t *testing.T) {
	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	host, err := common.ParseHost("acme.ghe.com")
	require.NoError(t, err)

	t.Run("ghe.com meta without installed_version", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"verifiable_password_authentication":true}`))},
		}}
		require.NoError(t, ProbeGitHubHost(httpCtx, host, orgID))
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.acme.ghe.com/meta", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Empty(t, httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("ghes installed_version without server flag", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"installed_version":"3.16.0"}`))},
		}}
		err := ProbeGitHubHost(httpCtx, host, orgID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub Enterprise Server is not available")
	})

	t.Run("ghes installed_version with server flag", func(t *testing.T) {
		restore := withGitHubEnterpriseServerForTest(func(uuid.UUID) bool { return true })
		t.Cleanup(restore)
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"installed_version":"3.16.0"}`))},
		}}
		require.NoError(t, ProbeGitHubHost(httpCtx, host, orgID))
	})

	t.Run("404", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("not found"))},
		}}
		err := ProbeGitHubHost(httpCtx, host, orgID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "this host is not a GitHub API")
	})

	t.Run("ghe.com 404 with enterprise version header", func(t *testing.T) {
		header := make(http.Header)
		header.Set("X-GitHub-Enterprise-Version", "ghe.com")
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{
				StatusCode: http.StatusNotFound,
				Header:     header,
				Body: io.NopCloser(strings.NewReader(
					`{"message":"Not Found","documentation_url":"https://docs.github.com/rest/meta/meta#get-apiname-meta-information","status":"404"}`,
				)),
			},
		}}
		require.NoError(t, ProbeGitHubHost(httpCtx, host, orgID))
	})

	t.Run("403 ip allow list", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader("forbidden"))},
		}}
		err := ProbeGitHubHost(httpCtx, host, orgID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "IP allow list")
	})
}

func TestWizardCreateConfigurationEnterprise(t *testing.T) {
	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	t.Run("empty host is github.com without probe", func(t *testing.T) {
		got, err := WizardCreateConfiguration(orgID, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{common.ConfigHost: ""}, got)
	})

	t.Run("persists privateApp and empty host", func(t *testing.T) {
		got, err := WizardCreateConfiguration(orgID, map[string]any{"privateApp": true, "host": ""}, nil)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{common.ConfigHost: "", "privateApp": true}, got)
	})

	t.Run("persists factoryOnboarding", func(t *testing.T) {
		got, err := WizardCreateConfiguration(orgID, map[string]any{"privateApp": true, "factoryOnboarding": true}, nil)
		require.NoError(t, err)
		assert.Equal(t, true, got["factoryOnboarding"])
	})

	t.Run("normalizes github.com to empty host", func(t *testing.T) {
		got, err := WizardCreateConfiguration(orgID, map[string]any{"host": "github.com"}, nil)
		require.NoError(t, err)
		assert.Equal(t, "", got[common.ConfigHost])
	})

	t.Run("rejects GitHub Enterprise hosts in Community", func(t *testing.T) {
		t.Cleanup(withGitHubEnterpriseForTest(communityGitHubEnterprise()))
		_, err := WizardCreateConfiguration(orgID, map[string]any{"host": "acme.ghe.com"}, nil)
		require.Error(t, err)
		assert.Equal(t, licensing.ErrNotLicensed.Error(), grpcerrors.StatusMessage(err))
		assert.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	})

	t.Run("accepts github.com in Community", func(t *testing.T) {
		t.Cleanup(withGitHubEnterpriseForTest(communityGitHubEnterprise()))
		got, err := WizardCreateConfiguration(orgID, map[string]any{"host": "github.com"}, nil)
		require.NoError(t, err)
		assert.Equal(t, "", got[common.ConfigHost])
	})

	t.Run("accepts ghe.com when the license grants GitHub Enterprise", func(t *testing.T) {
		withLicensedGitHubEnterprise(t)
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
		}}
		got, err := WizardCreateConfiguration(orgID, map[string]any{"host": "https://acme.ghe.com"}, httpCtx)
		require.NoError(t, err)
		assert.Equal(t, "acme.ghe.com", got[common.ConfigHost])
		assert.Equal(t, true, got[configPrivateApp])
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.acme.ghe.com/meta", httpCtx.Requests[0].URL.String())
	})

	t.Run("rejects a host outside the allowlist", func(t *testing.T) {
		withLicensedGitHubEnterprise(t)
		t.Setenv(config.EnvGitHubEnterpriseAllowedHosts, "allowed.ghe.com")
		_, err := WizardCreateConfiguration(orgID, map[string]any{"host": "acme.ghe.com"}, &contexts.HTTPContext{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not allowed")
	})
}

func TestPreferHostedInstallEnterprise(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	orgID := "11111111-1111-1111-1111-111111111111"
	assert.False(t, PreferHostedInstall(orgID, "github", map[string]any{"host": "acme.ghe.com"}))
	assert.True(t, WantsPrivateApp("github", map[string]any{"host": "acme.ghe.com"}))
	assert.True(t, HostRequiresPrivateApp(map[string]any{"host": "acme.ghe.com"}))
}

func TestGitHubSyncRefusesEnterpriseHost(t *testing.T) {
	err := (&GitHub{}).Sync(core.SyncContext{
		Configuration: Configuration{Host: "acme.ghe.com", PrivateApp: true},
		Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"host": "acme.ghe.com"}},
	})
	require.EqualError(t, err, "GitHub Enterprise uses the setup wizard")
}

func TestGitHubClientUsesEnterpriseURLs(t *testing.T) {
	host, err := common.ParseHost("acme.ghe.com")
	require.NoError(t, err)

	httpCtx := &rewriteHTTP{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/acme/app":
			_, _ = w.Write([]byte(`{"id":1,"full_name":"acme/app","name":"app"}`))
		case r.URL.Path == "/graphql":
			_, _ = w.Write([]byte(`{"data":{"viewer":{"login":"octocat"}}}`))
		case strings.HasPrefix(r.URL.Path, "/app-manifests/") && strings.HasSuffix(r.URL.Path, "/conversions"):
			_, _ = w.Write([]byte(`{"id":99,"slug":"superplane","pem":"pem","webhook_secret":"whsec"}`))
		default:
			http.NotFound(w, r)
		}
	})}

	intCtx := &contexts.IntegrationContext{
		NewSetupFlow: true,
		CurrentProperties: map[string]any{
			common.PropertyAuthMethod: common.AuthMethodPAT,
			common.PropertyOwner:      "acme",
			common.PropertyOwnerType:  common.OwnerTypeOrganization,
			common.PropertyHost:       "acme.ghe.com",
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			common.SecretPAT: {Name: common.SecretPAT, Value: []byte("ghe_token")},
		},
		Configuration: map[string]any{"host": "acme.ghe.com"},
	}

	client, err := common.NewClient(intCtx, httpCtx)
	require.NoError(t, err)
	assert.Equal(t, host.APIURL(), client.Host().APIURL())
	assert.Equal(t, "https://api.acme.ghe.com/graphql", client.Host().GraphQLURL())

	repo, err := client.FindRepository("acme/app")
	require.NoError(t, err)
	assert.Equal(t, "acme/app", repo.GetFullName())
	require.NotEmpty(t, httpCtx.urls)
	assert.Contains(t, httpCtx.urls[0], "https://api.acme.ghe.com/repos/acme/app")

	req, err := http.NewRequest(http.MethodPost, client.Host().GraphQLURL(), strings.NewReader(`{"query":"{ viewer { login } }"}`))
	require.NoError(t, err)
	res, err := client.HTTPDo(req)
	require.NoError(t, err)
	defer res.Body.Close()
	var payload struct {
		Data struct {
			Viewer struct {
				Login string `json:"login"`
			} `json:"viewer"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&payload))
	assert.Equal(t, "octocat", payload.Data.Viewer.Login)
	assert.Contains(t, strings.Join(httpCtx.urls, "\n"), "https://api.acme.ghe.com/graphql")

	ghes, err := common.ParseHost("ghes.corp")
	require.NoError(t, err)
	assert.Equal(t, "https://ghes.corp/api/graphql", ghes.GraphQLURL())
	assert.Equal(t, "https://ghes.corp/api/v3/", ghes.APIURL())

	appData, err := (&GitHub{}).createAppFromManifest(httpCtx, host, "code1")
	require.NoError(t, err)
	assert.Equal(t, int64(99), appData.ID)
	assert.Contains(t, strings.Join(httpCtx.urls, "\n"), "https://api.acme.ghe.com/app-manifests/code1/conversions")
}

func TestGitHubResolveSecretsEnterprise(t *testing.T) {
	integrationCtx := &contexts.IntegrationContext{
		NewSetupFlow: true,
		CurrentProperties: map[string]any{
			common.PropertyAuthMethod: common.AuthMethodPAT,
			common.PropertyHost:       "acme.ghe.com",
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			common.SecretPAT: {Name: common.SecretPAT, Value: []byte("ghe_token")},
		},
		Configuration: map[string]any{"host": "acme.ghe.com"},
	}

	secrets, err := (&GitHub{}).ResolveSecrets(core.IntegrationSecretContext{
		HTTP:        &contexts.HTTPContext{},
		Integration: integrationCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("ghe_token"), secrets.Values[integrationSecretGitHubToken])
	assert.Equal(t, []byte("acme.ghe.com"), secrets.Values["GH_HOST"])
	assert.Equal(t, []byte("ghe_token"), secrets.Values["GH_ENTERPRISE_TOKEN"])
	assert.Equal(t, []byte("https://acme.ghe.com"), secrets.Values["GITHUB_SERVER_URL"])
	assert.Equal(t, []byte("https://api.acme.ghe.com"), secrets.Values["GITHUB_API_URL"])
	assert.Equal(t, []byte("https://api.acme.ghe.com/graphql"), secrets.Values["GITHUB_GRAPHQL_URL"])
	assert.Contains(t, secrets.Usage, "GH_HOST")
	assert.Contains(t, secrets.Setup, "gh auth setup-git --hostname acme.ghe.com --force")
	assert.Equal(t, []string{"acme.ghe.com", "*.acme.ghe.com"}, common.HostFromIntegration(integrationCtx).AllowedHosts())
	assert.NotContains(t, common.HostFromIntegration(integrationCtx).AllowedHosts(), "*.ghe.com")
}

func TestFindWebhookSecretRefusesHostedFallbackOnEnterprise(t *testing.T) {
	setHostedAppEnv(t)
	ctx := core.HTTPRequestContext{
		Integration: &contexts.IntegrationContext{
			Metadata:      common.Metadata{HostedApp: true},
			Configuration: map[string]any{"host": "acme.ghe.com"},
		},
	}
	_, err := (&GitHub{}).findWebhookSecret(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub Enterprise cannot use SuperPlane's public GitHub App")
}

type rewriteHTTP struct {
	handler http.Handler
	urls    []string
}

func (c *rewriteHTTP) Do(req *http.Request) (*http.Response, error) {
	c.urls = append(c.urls, req.URL.String())
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}
