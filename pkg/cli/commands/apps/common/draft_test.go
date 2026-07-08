package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

func TestRequireCommitMessageErrorsOnEmpty(t *testing.T) {
	_, err := RequireCommitMessage("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "--message is required")
}

func TestEnsureLiveVersionIDReturnsNewestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-123" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"canvas-123","versionId":"version-1"}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	versionID, err := EnsureLiveVersionID(ctx, "canvas-123")
	require.NoError(t, err)
	require.Equal(t, "version-1", versionID)
}
