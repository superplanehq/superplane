package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestAdminRunnersGatewaySupportsCookieAndPersonalTokenAuthentication(t *testing.T) {
	server, resource, accountToken := setupAdminTestServer(t)
	registerTestGRPCGateway(
		t,
		server,
		resource.AuthService,
		resource.Registry,
		resource.Encryptor,
		support.NewOIDCProvider(),
	)

	fleet := &models.RunnerFleet{
		ID:        uuid.New(),
		ScopeType: models.RunnerFleetScopeInstallation,
		Slug:      "linux-amd64",
		Enabled:   true,
		Spec: datatypes.NewJSONType(models.RunnerFleetSpec{
			Architecture: "amd64",
		}),
		RunnerVersion: "0.1.0",
	}
	require.NoError(t, fleet.Create(database.DB(t.Context())))

	t.Run("account cookie", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/admin/api/installation/fleets",
			authCookie: accountToken,
		})
		assertFleetListResponse(t, response.Code, response.Body.Bytes(), fleet.Slug)
	})

	t.Run("personal token", func(t *testing.T) {
		rawToken, err := crypto.Base64String(32)
		require.NoError(t, err)
		token := models.NewUserAPIToken(resource.User, "Fleet Manager", crypto.HashToken(rawToken))
		require.NoError(t, models.CreateUserAPIToken(database.Conn(), token))

		response := execRequest(server, requestParams{
			method:    http.MethodGet,
			path:      "/admin/api/installation/fleets",
			authToken: rawToken,
		})
		assertFleetListResponse(t, response.Code, response.Body.Bytes(), fleet.Slug)
	})
}

func assertFleetListResponse(t *testing.T, status int, body []byte, fleetID string) {
	t.Helper()
	assert.Equal(t, http.StatusOK, status, string(body))

	var response struct {
		Fleets []struct {
			ID string `json:"id"`
		} `json:"fleets"`
	}
	require.NoError(t, json.Unmarshal(body, &response))
	require.Len(t, response.Fleets, 1)
	assert.Equal(t, fleetID, response.Fleets[0].ID)
}
