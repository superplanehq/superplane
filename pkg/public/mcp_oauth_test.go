package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support"
)

func Test__HandleMCPOAuthCallback_missingState(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp-oauth/callback", nil)
	rec := httptest.NewRecorder()
	(&Server{}).HandleMCPOAuthCallback(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test__HandleMCPOAuthCallback_unknownState(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp-oauth/callback?state=missing", nil)
	rec := httptest.NewRecorder()
	(&Server{}).HandleMCPOAuthCallback(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func Test__HandleMCPOAuthClientMetadata(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-client", nil)
	rec := httptest.NewRecorder()
	(&Server{WebhooksBaseURL: "https://app.example"}).HandleMCPOAuthClientMetadata(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "SuperPlane", body["client_name"])
	assert.Equal(t, "https://app.example/.well-known/oauth-client", body["client_id"])
	redirects, ok := body["redirect_uris"].([]any)
	require.True(t, ok)
	require.Len(t, redirects, 1)
	assert.Equal(t, "https://app.example/api/v1/mcp-oauth/callback", redirects[0])
}
