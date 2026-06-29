package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support"
)

func TestListExperimentalFeatures(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	t.Run("returns the static registry to a non-admin account", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/account/experimental-features",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)

		var body struct {
			Features []map[string]any `json:"features"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.NotEmpty(t, body.Features)

		ids := make([]string, 0, len(body.Features))
		for _, f := range body.Features {
			id, _ := f["id"].(string)
			ids = append(ids, id)
		}
		assert.Contains(t, ids, "claude_managed_agents")
	})

	t.Run("rejects unauthenticated requests", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/account/experimental-features",
		})
		assert.NotEqual(t, http.StatusOK, response.Code)
	})
}
