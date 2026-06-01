package public

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestAgentSessionWebSocketRequiresManagedAgentsFeature(t *testing.T) {
	// The managed-agents feature is released by default, so override the
	// registry to keep it gated and exercise the forbidden path.
	t.Cleanup(features.WithRegistryForTest([]features.Feature{
		{ID: features.FeatureClaudeManagedAgents, Label: features.FeatureClaudeManagedAgents},
	}))

	r := support.Setup(t)
	defer r.Close()
	server, _, token := setupTestServer(r, t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := &models.AgentSession{
		OrganizationID:    r.Organization.ID,
		UserID:            r.User,
		CanvasID:          canvas.ID,
		Provider:          "anthropic",
		ProviderSessionID: "provider-session",
		Status:            models.AgentSessionStatusIdle,
	}
	require.NoError(t, models.CreateAgentSessionInTransaction(database.Conn(), session))

	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/ws/agents/sessions/" + session.ID.String() + "?organization_id=" + r.Organization.ID.String(),
		authCookie: token,
	})

	assert.Equal(t, http.StatusForbidden, response.Code)
}
