package index

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

func TestWidgetsCommandExecuteListText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/widgets", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"widgets":[{"name":"annotation","label":"Annotation","description":"Canvas note"}]}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newWidgetsCommandContextForTest(t, server, "text")
	name := ""
	command := widgetsCommand{name: &name}

	err := command.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "NAME")
	require.Contains(t, stdout.String(), "annotation")
	require.Contains(t, stdout.String(), "Canvas note")
}

func TestWidgetsCommandExecuteDescribeJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/widgets/annotation", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"widget":{"name":"annotation","label":"Annotation","description":"Canvas note"}}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newWidgetsCommandContextForTest(t, server, "json")
	name := "annotation"
	command := widgetsCommand{name: &name}

	err := command.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"name": "annotation"`)
	require.Contains(t, stdout.String(), `"label": "Annotation"`)
}

func TestWidgetsCommandExecuteListYAML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/widgets", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"widgets":[{"name":"annotation","label":"Annotation","description":"Canvas note"}]}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newWidgetsCommandContextForTest(t, server, "yaml")
	name := ""
	command := widgetsCommand{name: &name}

	err := command.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "name: annotation")
	require.Contains(t, stdout.String(), "label: Annotation")
}

func TestWidgetsCommandExecuteReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/widgets/annotation", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":13,"message":"widget lookup failed"}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := newWidgetsCommandContextForTest(t, server, "text")
	name := "annotation"
	command := widgetsCommand{name: &name}

	err := command.Execute(ctx)
	require.Error(t, err)
}

func newWidgetsCommandContextForTest(
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
