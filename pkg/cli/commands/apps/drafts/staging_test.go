package drafts

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

const (
	stagingTestAppID   = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	stagingTestDraftID = "draft-1"
	stagingTestUserID  = "user-1"
)

func stagingCommitPath(appID, draftID string) string {
	return "/api/v1/canvases/" + appID + "/versions/" + draftID + "/staging/commit"
}

func stagingDiscardPath(appID, draftID string) string {
	return "/api/v1/canvases/" + appID + "/versions/" + draftID + "/staging/discard"
}

func TestStagingCommitCallsCommitEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"` + stagingTestUserID + `"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/"+stagingTestAppID+"/versions/"+stagingTestDraftID:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"` + stagingTestDraftID + `","state":"STATE_DRAFT","owner":{"id":"` + stagingTestUserID + `"}}}}`))
		case r.Method == http.MethodPost && r.URL.Path == stagingCommitPath(stagingTestAppID, stagingTestDraftID):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"` + stagingTestDraftID + `"}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: stagingTestAppID})
	ctx.Args = []string{stagingTestDraftID}

	require.NoError(t, (&stagingCommitCommand{}).Execute(ctx))
	require.Contains(t, stdout.String(), "Staging committed for draft "+stagingTestDraftID)
}

func TestStagingResetDiscardsAllWhenNoPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"` + stagingTestUserID + `"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/"+stagingTestAppID+"/versions/"+stagingTestDraftID:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"` + stagingTestDraftID + `","state":"STATE_DRAFT","owner":{"id":"` + stagingTestUserID + `"}}}}`))
		case r.Method == http.MethodPost && r.URL.Path == stagingDiscardPath(stagingTestAppID, stagingTestDraftID):
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var parsed map[string]any
			require.NoError(t, json.Unmarshal(body, &parsed))
			_, hasPaths := parsed["paths"]
			require.False(t, hasPaths)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stagingState":{"hasStaging":false}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: stagingTestAppID})
	ctx.Args = []string{stagingTestDraftID}

	paths := []string{}
	require.NoError(t, (&stagingResetCommand{paths: &paths}).Execute(ctx))
	require.Contains(t, stdout.String(), "All staging discarded for draft "+stagingTestDraftID)
}
