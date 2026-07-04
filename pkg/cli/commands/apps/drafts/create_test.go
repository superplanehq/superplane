package drafts

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

const testAppID = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

func draftVersionsPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/versions"
}

func TestCreateCommandPrintsDraftDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == draftVersionsPath(testAppID):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1","displayName":"wip"}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	name := "wip"
	err := (&createCommand{name: &name}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Draft created for app "+testAppID)
	require.Contains(t, stdout.String(), "Draft ID: draft-1")
	require.Contains(t, stdout.String(), "Name:     wip")
}

func TestCreateCommandReturnsJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == draftVersionsPath(testAppID):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1"}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "json")
	ctx.Args = []string{testAppID}

	err := (&createCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"id": "draft-1"`)
}

func TestListCommandFiltersToCurrentUserDrafts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"user-1","name":"you@example.com"}}`))
		case r.Method == http.MethodGet && r.URL.Path == draftVersionsPath(testAppID):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","displayName":"mine","owner":{"id":"user-1","name":"you@example.com"},"updatedAt":"2026-06-09T12:00:00Z"}},{"metadata":{"id":"draft-2","displayName":"theirs","owner":{"id":"user-2","name":"alice@acme.io"},"updatedAt":"2026-06-09T11:00:00Z"}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	all := false
	err := (&listCommand{all: &all}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "draft-1")
	require.Contains(t, stdout.String(), "you")
	require.NotContains(t, stdout.String(), "draft-2")
}

func TestListCommandShowsAllOwnersWithFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == draftVersionsPath(testAppID):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","owner":{"id":"user-1","name":"you@example.com"},"updatedAt":"2026-06-09T12:00:00Z"}},{"metadata":{"id":"draft-2","owner":{"id":"user-2","name":"alice@acme.io"},"updatedAt":"2026-06-09T11:00:00Z"}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	all := true
	err := (&listCommand{all: &all}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "draft-1")
	require.Contains(t, stdout.String(), "draft-2")
	require.Contains(t, stdout.String(), "alice@acme.io")
}

func TestListCommandPrintsEmptyMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == draftVersionsPath(testAppID):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	all := false
	err := (&listCommand{all: &all}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "No drafts found for app "+testAppID)
}

func TestDeleteCommandPrintsConfirmation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == draftVersionsPath(testAppID)+"/draft-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: testAppID})
	ctx.Args = []string{"draft-1"}

	err := (&deleteCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Draft deleted: draft-1")
}

func TestDeleteCommandResolvesAppFromArgument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == draftVersionsPath(testAppID)+"/draft-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{"draft-1", testAppID}

	err := (&deleteCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Draft deleted: draft-1")
}

func TestDeleteCommandErrorsWhenNoAppAndNoActive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{"draft-1"}

	err := (&deleteCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "is required")
}

func TestDeleteCommandMapsPermissionDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":7,"message":"version owner mismatch"}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: testAppID})
	ctx.Args = []string{"draft-1"}

	err := (&deleteCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "you can only delete your own drafts")
}

func TestDeleteCommandMapsFailedPrecondition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusFailedDependency)
		_, _ = w.Write([]byte(`{"code":9,"message":"only draft versions can be discarded"}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: testAppID})
	ctx.Args = []string{"live-1"}

	err := (&deleteCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only draft versions can be deleted")
}

func TestDeleteCommandMapsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":5,"message":"version not found"}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: testAppID})
	ctx.Args = []string{"missing"}

	err := (&deleteCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "draft not found")
}
