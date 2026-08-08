package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/buildinfo"
)

func TestServeVersion(t *testing.T) {
	previous := buildinfo.Version
	buildinfo.Version = "v9.9.9-test"
	t.Cleanup(func() { buildinfo.Version = previous })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	rec := httptest.NewRecorder()

	serveVersion(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var payload versionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "v9.9.9-test", payload.Version)
}
