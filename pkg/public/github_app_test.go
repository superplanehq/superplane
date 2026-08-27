package public

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gh "github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__HandleGitHubAppSetup_missingState(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/github/app/setup", nil)
	rec := httptest.NewRecorder()

	server.HandleGitHubAppSetup(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test__HandleGitHubAppOAuthCallback_missingState(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/github/app/oauth/callback", nil)
	rec := httptest.NewRecorder()

	server.HandleGitHubAppOAuthCallback(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test__HandleGitHubAppBind_missingState(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/github/app/bind", nil)
	rec := httptest.NewRecorder()

	server.HandleGitHubAppBind(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test__githubInstallationID(t *testing.T) {
	id := int64(42)
	got, ok := githubInstallationID(&gh.InstallationEvent{
		Installation: &gh.Installation{ID: &id},
	})
	require.True(t, ok)
	assert.Equal(t, "42", got)

	_, ok = githubInstallationID(&gh.PushEvent{})
	assert.False(t, ok)
}

func Test__authorizeHostedGitHubAppCallback(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	state := "csrf-state-1"
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		"github-hosted",
		nil,
	)
	require.NoError(t, err)

	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"state":           state,
		"hostedApp":       true,
		"startedByUserID": r.User.String(),
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	t.Run("missing session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/github/app/setup?state="+state, nil)
		rec := httptest.NewRecorder()

		(&Server{}).HandleGitHubAppSetup(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("starter account", func(t *testing.T) {
		status := authorizeHostedGitHubAppCallback(
			accountContext(r.Account),
			integration,
		)
		assert.Equal(t, 0, status)
	})

	t.Run("same org teammate is rejected", func(t *testing.T) {
		teammateAccount, err := models.CreateAccount("teammate", "teammate@example.com")
		require.NoError(t, err)
		_, err = models.CreateUser(r.Organization.ID, teammateAccount.ID, teammateAccount.Email, teammateAccount.Name)
		require.NoError(t, err)

		status := authorizeHostedGitHubAppCallback(
			accountContext(teammateAccount),
			integration,
		)
		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("other org account is rejected", func(t *testing.T) {
		otherOrg, err := models.CreateOrganization("other-org", "other org")
		require.NoError(t, err)
		otherAccount, err := models.CreateAccount("other", "other@example.com")
		require.NoError(t, err)
		_, err = models.CreateUser(otherOrg.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
		require.NoError(t, err)

		status := authorizeHostedGitHubAppCallback(
			accountContext(otherAccount),
			integration,
		)
		assert.Equal(t, http.StatusForbidden, status)
	})
}

func Test__dispatchGitHubAppByState_rejectsNonHosted(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	state := "legacy-state"
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		"github-legacy",
		nil,
	)
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"state":     state,
		"hostedApp": false,
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/github/app/setup?state="+state, nil)
	req = req.WithContext(accountContext(r.Account))
	rec := httptest.NewRecorder()

	(&Server{}).HandleGitHubAppSetup(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func Test__isHostedGitHubAppBrowserCallback(t *testing.T) {
	hosted := &models.Integration{
		AppName: "github",
		Metadata: datatypes.NewJSONType(map[string]any{
			"hostedApp": true,
		}),
	}
	legacy := &models.Integration{
		AppName: "github",
		Metadata: datatypes.NewJSONType(map[string]any{
			"hostedApp": false,
		}),
	}

	setup := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/"+uuid.NewString()+"/setup", nil)
	webhook := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+uuid.NewString()+"/webhook", nil)

	assert.True(t, isHostedGitHubAppBrowserCallback(setup, hosted))
	assert.False(t, isHostedGitHubAppBrowserCallback(setup, legacy))
	assert.False(t, isHostedGitHubAppBrowserCallback(webhook, hosted))
}

func accountContext(account *models.Account) context.Context {
	return context.WithValue(context.Background(), middleware.AccountContextKey, account)
}
