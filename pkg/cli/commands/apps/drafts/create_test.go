package drafts

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

const testAppID = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

func versionsPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/versions"
}

func stagingPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/staging"
}

func TestCreateCommandPrintsLiveVersionDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == versionsPath(testAppID) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"version-1"}}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	err := (&createCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "App "+testAppID)
	require.Contains(t, stdout.String(), "Live version: version-1")
}

func TestCreateCommandReturnsJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == versionsPath(testAppID) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"version-1"}}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "json")
	ctx.Args = []string{testAppID}

	err := (&createCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"versionId": "version-1"`)
}

func TestListCommandPrintsStagingSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == stagingPath(testAppID) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":true,"stagedPaths":["canvas.yaml"],"baseVersionId":"version-1","stale":false}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	err := (&listCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "canvas.yaml")
	require.Contains(t, stdout.String(), "version-1")
}

func TestListCommandPrintsEmptyMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == stagingPath(testAppID) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":false}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	err := (&listCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "No staged changes for app "+testAppID)
}

func TestDeleteCommandPrintsConfirmation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == stagingPath(testAppID) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":false}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: testAppID})
	ctx.Args = []string{}

	err := (&deleteCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Staged changes discarded for app "+testAppID)
}

func TestDeleteCommandResolvesAppFromArgument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == stagingPath(testAppID) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":false}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	err := (&deleteCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Staged changes discarded for app "+testAppID)
}

func TestDeleteCommandErrorsWhenNoAppAndNoActive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{}

	err := (&deleteCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "is required")
}
