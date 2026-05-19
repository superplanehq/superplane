package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/test/support"
)

func TestOrgListFleets(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	h := testHandler(t, r)

	t.Run("requires user in context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runner-fleets", nil)
		h.OrgListFleets(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("empty when runner feature disabled", func(t *testing.T) {
		_ = models.DisableExperimentalFeature(r.Organization.ID, "runner")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runner-fleets", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, r.UserModel))
		h.OrgListFleets(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var items []orgFleetOption
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &items))
		assert.Empty(t, items)
	})

	t.Run("lists fleets when runner enabled", func(t *testing.T) {
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, "runner"))

		created := adminPOST(t, h, map[string]any{"name": "fleet-" + uuid.New().String()})
		require.Equal(t, http.StatusCreated, created.Code)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runner-fleets", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, r.UserModel))
		h.OrgListFleets(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var items []orgFleetOption
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &items))
		require.NotEmpty(t, items)
		assert.NotEmpty(t, items[0].ID)
		assert.NotEmpty(t, items[0].Name)
	})
}
