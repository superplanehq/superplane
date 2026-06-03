package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/test/support"
)

func runnerFleetsGET(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	r *support.ResourceRegistry,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/fleets", nil)
	req.Header.Set("x-organization-id", r.Organization.ID.String())
	token, err := signer.Generate(r.Account.ID.String(), time.Hour)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})

	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func TestRunnerFleets(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)

	t.Run("broker not configured", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "")

		response := runnerFleetsGET(t, server, signer, r)
		assert.Equal(t, http.StatusOK, response.Code)

		var body runnerFleetsResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.False(t, body.Configured)
		assert.Empty(t, body.Fleets)
	})

	t.Run("returns broker fleets", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/v1/fleets", r.URL.Path)
			assert.Equal(t, "Bearer broker-token", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"aws-standard-amd64","provisioner":"aws","arch":"amd64","size":"t3.micro","created_at_unix":1710000000}]`))
		}))
		defer upstream.Close()

		t.Setenv("TASK_BROKER_BASE_URL", upstream.URL)
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "broker-token")

		response := runnerFleetsGET(t, server, signer, r)
		assert.Equal(t, http.StatusOK, response.Code)

		var body runnerFleetsResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.True(t, body.Configured)
		require.Len(t, body.Fleets, 1)
		assert.Equal(t, "aws-standard-amd64", body.Fleets[0].ID)
		assert.Equal(t, "aws", body.Fleets[0].Provisioner)
		assert.Equal(t, "amd64", body.Fleets[0].Arch)
		assert.Equal(t, "t3.micro", body.Fleets[0].Size)
		assert.Equal(t, int64(1710000000), body.Fleets[0].CreatedAt)
	})
}
