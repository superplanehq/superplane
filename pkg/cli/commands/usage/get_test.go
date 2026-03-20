package usage

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestGetCommandExecuteText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

		switch r.URL.Path {
		case "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"user-1","organizationId":"org-123"}`))
		case "/api/v1/organizations/org-123/usage":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"enabled": true,
				"statusMessage": "usage tracking active",
				"limits": {
					"maxCanvases": 10,
					"maxNodesPerCanvas": 200,
					"maxUsers": 25,
					"retentionWindowDays": 30,
					"maxEventsPerMonth": "100000",
					"maxIntegrations": -1
				},
				"usage": {
					"canvases": 3,
					"eventBucketLevel": 12,
					"eventBucketCapacity": 100,
					"eventBucketLastUpdatedAt": "2026-03-19T15:04:05Z"
				}
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newUsageCommandContextForTest(t, server, "text")

	err := (&getCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Enabled: true")
	require.Contains(t, stdout.String(), "Status: usage tracking active")
	require.Contains(t, stdout.String(), "Canvases")
	require.Contains(t, stdout.String(), "3")
	require.Contains(t, stdout.String(), "Event bucket")
	require.Contains(t, stdout.String(), "12 / 100")
	require.Contains(t, stdout.String(), "Max integrations")
	require.Contains(t, stdout.String(), "unlimited")
}

func TestGetCommandExecuteJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

		switch r.URL.Path {
		case "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"user-1","organizationId":"org-123"}`))
		case "/api/v1/organizations/org-123/usage":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"enabled":false,"statusMessage":"usage disabled"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newUsageCommandContextForTest(t, server, "json")

	err := (&getCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"enabled": false`)
	require.Contains(t, stdout.String(), `"statusMessage": "usage disabled"`)
}

func TestGetCommandExecuteRequiresOrganizationID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/me", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"user-1"}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := newUsageCommandContextForTest(t, server, "text")

	err := (&getCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "organization id not found")
}

func newUsageCommandContextForTest(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{
		{
			URL: server.URL,
		},
	}
	client := openapi_client.NewAPIClient(config)

	return core.CommandContext{
		Context:  context.Background(),
		API:      client,
		Renderer: renderer,
	}, stdout
}
