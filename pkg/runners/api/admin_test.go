package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support"
)

func adminPOST(t *testing.T, h *Handler, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(body))
	req := httptest.NewRequest(http.MethodPost, "/admin/api/runner/fleets", &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.AdminRegisterFleet(rec, req)
	return rec
}

func adminGET(t *testing.T, h *Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/runner/fleets", nil)
	rec := httptest.NewRecorder()
	h.AdminListFleets(rec, req)
	return rec
}

func TestAdminRegisterRunnerFleet(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	h := testHandler(t, r)

	t.Run("registers fleet and returns auth token", func(t *testing.T) {
		rec := adminPOST(t, h, map[string]any{
			"name": "fleet-" + uuid.New().String(),
		})
		require.Equal(t, http.StatusCreated, rec.Code)

		var created struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			AuthToken string `json:"auth_token"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		assert.Contains(t, created.Name, "fleet-")
		assert.NotEmpty(t, created.AuthToken)
		assert.NotEmpty(t, created.ID)
	})

	t.Run("list fleets", func(t *testing.T) {
		rec := adminGET(t, h)
		require.Equal(t, http.StatusOK, rec.Code)

		var items []fleetResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &items))
		require.NotEmpty(t, items)
		assert.NotEmpty(t, items[0].Name)
	})
}
