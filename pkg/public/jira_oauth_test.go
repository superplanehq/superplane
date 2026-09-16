package public

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appconfig "github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__HandleJiraOAuthCallback_missingState(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/oauth/callback", nil)
	rec := httptest.NewRecorder()

	server.HandleJiraOAuthCallback(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test__HandleJiraOAuthCallback_unknownState(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/oauth/callback?state=missing", nil)
	rec := httptest.NewRecorder()

	(&Server{}).HandleJiraOAuthCallback(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func Test__HandleJiraOAuthCallback_rejectsNonHosted(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	state := "legacy-jira-state"
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"jira",
		"jira-legacy",
		nil,
	)
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"state":       state,
		"hostedOAuth": false,
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/oauth/callback?state="+state, nil)
	rec := httptest.NewRecorder()

	(&Server{}).HandleJiraOAuthCallback(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	found, err := models.FindJiraIntegrationByOAuthState(database.Conn(), state)
	require.NoError(t, err)
	assert.Equal(t, integration.ID, found.ID)
}

func Test__HandleJiraOAuthCallback_consumesHostedState(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Setenv(appconfig.EnvJiraOAuthClientID, "jira-client")
	t.Setenv(appconfig.EnvJiraOAuthClientSecret, "jira-secret")

	state := "hosted-jira-state"
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"jira",
		"jira-hosted",
		nil,
	)
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"state":       state,
		"hostedOAuth": true,
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	encryptor := &crypto.NoOpEncryptor{}
	reg, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/oauth/callback?error=access_denied&state="+state, nil)
	rec := httptest.NewRecorder()
	(&Server{registry: reg, encryptor: encryptor}).HandleJiraOAuthCallback(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)

	_, err = models.FindJiraIntegrationByOAuthState(database.Conn(), state)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = models.ClaimHostedJiraOAuthState(database.Conn(), state)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test__ClaimHostedJiraOAuthState(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Run("consumes hosted state and keeps it in memory", func(t *testing.T) {
		state := "claim-hosted-state"
		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"jira",
			"jira-claim-hosted",
			nil,
		)
		require.NoError(t, err)
		integration.Metadata = datatypes.NewJSONType(map[string]any{
			"state":       state,
			"hostedOAuth": true,
			"siteName":    "keep-me",
		})
		require.NoError(t, database.Conn().Save(integration).Error)

		claimed, err := models.ClaimHostedJiraOAuthState(database.Conn(), state)
		require.NoError(t, err)
		require.NotNil(t, claimed)
		assert.Equal(t, integration.ID, claimed.ID)
		assert.Equal(t, state, claimed.Metadata.Data()["state"])
		assert.Equal(t, "keep-me", claimed.Metadata.Data()["siteName"])

		_, err = models.FindJiraIntegrationByOAuthState(database.Conn(), state)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("does not consume a legacy connection", func(t *testing.T) {
		state := "claim-legacy-state"
		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"jira",
			"jira-claim-legacy",
			nil,
		)
		require.NoError(t, err)
		integration.Metadata = datatypes.NewJSONType(map[string]any{
			"state":       state,
			"hostedOAuth": false,
		})
		require.NoError(t, database.Conn().Save(integration).Error)

		_, err = models.ClaimHostedJiraOAuthState(database.Conn(), state)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)

		found, err := models.FindJiraIntegrationByOAuthState(database.Conn(), state)
		require.NoError(t, err)
		assert.Equal(t, integration.ID, found.ID)
	})
}

func Test__isHostedJiraOAuth(t *testing.T) {
	hosted := &models.Integration{
		AppName: "jira",
		Metadata: datatypes.NewJSONType(map[string]any{
			"hostedOAuth": true,
		}),
	}
	legacy := &models.Integration{
		AppName: "jira",
		Metadata: datatypes.NewJSONType(map[string]any{
			"hostedOAuth": false,
		}),
	}
	github := &models.Integration{
		AppName: "github",
		Metadata: datatypes.NewJSONType(map[string]any{
			"hostedOAuth": true,
		}),
	}

	assert.True(t, isHostedJiraOAuth(hosted))
	assert.False(t, isHostedJiraOAuth(legacy))
	assert.False(t, isHostedJiraOAuth(github))
	assert.False(t, isHostedJiraOAuth(nil))
}
