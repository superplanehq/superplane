package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gh "github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestSetupCallbackStateAccepted(t *testing.T) {
	t.Run("matching state", func(t *testing.T) {
		assert.True(t, setupCallbackStateAccepted("abc", "abc", false, 0, 0))
	})

	t.Run("wrong state", func(t *testing.T) {
		assert.False(t, setupCallbackStateAccepted("other", "abc", false, 99, 99))
	})

	t.Run("missing state hosted app", func(t *testing.T) {
		assert.False(t, setupCallbackStateAccepted("", "abc", true, 99, 99))
	})

	t.Run("missing state matching app id", func(t *testing.T) {
		assert.True(t, setupCallbackStateAccepted("", "abc", false, 99, 99))
	})

	t.Run("missing state mismatched app id", func(t *testing.T) {
		assert.False(t, setupCallbackStateAccepted("", "abc", false, 99, 100))
	})
}

func TestSetupCallbackAccepted(t *testing.T) {
	t.Run("matching state", func(t *testing.T) {
		ctx := core.HTTPRequestContext{Integration: &contexts.IntegrationContext{}}
		assert.True(t, setupCallbackAccepted(ctx, "abc", "abc", "139366"))
	})

	t.Run("wrong state", func(t *testing.T) {
		ctx := core.HTTPRequestContext{
			Integration: &contexts.IntegrationContext{
				CurrentProperties: map[string]any{common.PropertyAppID: "99"},
			},
		}
		assert.False(t, setupCallbackAccepted(ctx, "other", "abc", "139366"))
	})

	t.Run("missing state hosted app", func(t *testing.T) {
		ctx := core.HTTPRequestContext{
			Integration: &contexts.IntegrationContext{
				Metadata:          common.Metadata{HostedApp: true, GitHubApp: common.GitHubAppMetadata{ID: 99}},
				CurrentProperties: map[string]any{common.PropertyAppID: "99"},
			},
		}
		assert.False(t, setupCallbackAccepted(ctx, "", "abc", "139366"))
	})

	t.Run("missing state matching app id", func(t *testing.T) {
		previous := lookupInstallationAppID
		lookupInstallationAppID = func(core.HTTPRequestContext, int64, string) (int64, error) {
			return 99, nil
		}
		t.Cleanup(func() { lookupInstallationAppID = previous })

		ctx := core.HTTPRequestContext{
			Integration: &contexts.IntegrationContext{
				CurrentProperties: map[string]any{common.PropertyAppID: "99"},
			},
		}
		assert.True(t, setupCallbackAccepted(ctx, "", "abc", "139366"))
	})

	t.Run("missing state mismatched app id", func(t *testing.T) {
		previous := lookupInstallationAppID
		lookupInstallationAppID = func(core.HTTPRequestContext, int64, string) (int64, error) {
			return 100, nil
		}
		t.Cleanup(func() { lookupInstallationAppID = previous })

		ctx := core.HTTPRequestContext{
			Integration: &contexts.IntegrationContext{
				CurrentProperties: map[string]any{common.PropertyAppID: "99"},
			},
		}
		assert.False(t, setupCallbackAccepted(ctx, "", "abc", "139366"))
	})
}

func TestInstallationAppIDFromClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/app/installations/139366", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":139366,"app_id":99}`))
	}))
	t.Cleanup(server.Close)

	client := gh.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL

	appID, err := installationAppIDFromClient(context.Background(), client, "139366")
	require.NoError(t, err)
	assert.Equal(t, int64(99), appID)
}

func TestAfterAppInstallationHonorsSetupReturnPath(t *testing.T) {
	integrationID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	returnPath := "/org-1/workspaces/pay/setup?step=repo"

	request := httptest.NewRequest(http.MethodGet, "/setup", nil)
	request.AddCookie(&http.Cookie{
		Name:  integrationSetupReturnCookie,
		Value: url.QueryEscape(returnPath),
	})
	response := httptest.NewRecorder()

	(&GitHub{}).afterAppInstallation(core.HTTPRequestContext{
		Logger:         log.New().WithField("test", t.Name()),
		Request:        request,
		Response:       response,
		OrganizationID: "org-1",
		BaseURL:        "https://app.example",
		Integration: &contexts.IntegrationContext{
			NewSetupFlow:  true,
			IntegrationID: integrationID,
			CurrentProperties: map[string]any{
				common.PropertyAppInstallationID: "139366",
			},
		},
	})

	assert.Equal(t, http.StatusSeeOther, response.Code)
	assert.Equal(t, "https://app.example"+returnPath, response.Header().Get("Location"))
	assert.NotContains(t, response.Header().Get("Location"), "/settings/integrations/")
}

func TestAfterAppCreationGitHubEnterpriseRequiresLicense(t *testing.T) {
	t.Cleanup(withGitHubEnterpriseForTest(communityGitHubEnterprise()))
	request := httptest.NewRequest(http.MethodGet, "/redirect?code=abc&state=s", nil)
	response := httptest.NewRecorder()

	(&GitHub{}).afterAppCreation(core.HTTPRequestContext{
		Logger:   log.New().WithField("test", t.Name()),
		Request:  request,
		Response: response,
		Integration: &contexts.IntegrationContext{
			CurrentProperties: map[string]any{common.PropertyHost: "acme.ghe.com"},
		},
	})

	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Contains(t, response.Body.String(), licensing.ErrNotLicensed.Error())
}

func TestAfterAppCreationGitHubDotComDoesNotRequireLicense(t *testing.T) {
	t.Cleanup(withGitHubEnterpriseForTest(communityGitHubEnterprise()))
	request := httptest.NewRequest(http.MethodGet, "/redirect", nil)
	response := httptest.NewRecorder()

	(&GitHub{}).afterAppCreation(core.HTTPRequestContext{
		Logger:   log.New().WithField("test", t.Name()),
		Request:  request,
		Response: response,
		Integration: &contexts.IntegrationContext{
			NewSetupFlow: true,
		},
	})

	assert.NotEqual(t, http.StatusForbidden, response.Code)
}

func TestAfterAppInstallationExistingGitHubEnterpriseAppWithoutLicense(t *testing.T) {
	t.Cleanup(withGitHubEnterpriseForTest(communityGitHubEnterprise()))
	request := httptest.NewRequest(http.MethodGet, "/setup?installation_id=139366&setup_action=install&state=abc", nil)
	response := httptest.NewRecorder()

	(&GitHub{}).afterAppInstallation(core.HTTPRequestContext{
		Logger:         log.New().WithField("test", t.Name()),
		Request:        request,
		Response:       response,
		OrganizationID: "org-1",
		BaseURL:        "https://app.example",
		Integration: &contexts.IntegrationContext{
			NewSetupFlow:  true,
			IntegrationID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			CurrentProperties: map[string]any{
				common.PropertyHost:              "acme.ghe.com",
				common.PropertyAppID:             "99",
				common.PropertyAppInstallationID: "139366",
			},
		},
	})

	assert.Equal(t, http.StatusSeeOther, response.Code)
}

func TestAfterAppInstallationNewGitHubEnterpriseAppRequiresLicense(t *testing.T) {
	t.Cleanup(withGitHubEnterpriseForTest(communityGitHubEnterprise()))
	request := httptest.NewRequest(http.MethodGet, "/setup?installation_id=139366&setup_action=install&state=abc", nil)
	response := httptest.NewRecorder()

	(&GitHub{}).afterAppInstallation(core.HTTPRequestContext{
		Logger:         log.New().WithField("test", t.Name()),
		Request:        request,
		Response:       response,
		OrganizationID: "org-1",
		BaseURL:        "https://app.example",
		Integration: &contexts.IntegrationContext{
			NewSetupFlow: true,
			CurrentProperties: map[string]any{
				common.PropertyHost: "acme.ghe.com",
			},
		},
	})

	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Contains(t, response.Body.String(), licensing.ErrNotLicensed.Error())
}
