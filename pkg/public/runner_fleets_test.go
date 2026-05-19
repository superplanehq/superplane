package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/runners"
	"github.com/superplanehq/superplane/test/support"
)

func TestAdminRegisterRunnerFleet(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	t.Run("bridge mode without fleet_url", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"name": "bridge-fleet-" + uuid.New().String(),
			"mode": runners.FleetModeBridge,
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      http.MethodPost,
			path:        "/admin/api/runner/fleets",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		require.Equal(t, http.StatusCreated, response.Code)

		var created struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Mode      string `json:"mode"`
			AuthToken string `json:"auth_token"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &created))
		assert.Contains(t, created.Name, "bridge-fleet-")
		assert.Equal(t, runners.FleetModeBridge, created.Mode)
		assert.NotEmpty(t, created.AuthToken)
	})

	t.Run("push mode requires fleet_url", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"name": "push-fleet",
			"mode": runners.FleetModePush,
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      http.MethodPost,
			path:        "/admin/api/runner/fleets",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "fleet_url")
	})

	t.Run("invalid mode", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"name": "bad",
			"mode": "invalid",
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      http.MethodPost,
			path:        "/admin/api/runner/fleets",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("list fleets includes mode", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/admin/api/runner/fleets",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)

		var items []fleetResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &items))
		require.NotEmpty(t, items)
		assert.NotEmpty(t, items[0].Mode)
	})
}

func TestAdminRegisterRunnerFleetRequiresAdmin(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, token := setupTestServer(r, t)

	body, err := json.Marshal(map[string]any{
		"name": "x",
		"mode": runners.FleetModeBridge,
	})
	require.NoError(t, err)
	response := execRequest(server, requestParams{
		method:      http.MethodPost,
		path:        "/admin/api/runner/fleets",
		authCookie:  token,
		body:        body,
		contentType: "application/json",
	})
	assert.Equal(t, http.StatusNotFound, response.Code)
}
