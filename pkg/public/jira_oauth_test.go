package public

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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
