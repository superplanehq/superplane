package canvas

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

const (
	testGetCanvasID    = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	testGetCanvasName  = "my-canvas"
	testDescribeCanvas = "/api/v1/canvases/" + testGetCanvasID
)

const sampleCanvasYAMLBody = "apiVersion: v1\nkind: Canvas\nmetadata:\n  id: " + testGetCanvasID + "\n  name: " + testGetCanvasName + "\nspec:\n  nodes: []\n  edges: []\n"

func describeCanvasMetadataResponse(t *testing.T, w http.ResponseWriter, _ *http.Request) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testGetCanvasID + `","name":"` + testGetCanvasName + `"}}}`))
}

func describeCanvasMetadataWithOrgResponse(t *testing.T, w http.ResponseWriter, _ *http.Request) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testGetCanvasID + `","name":"` + testGetCanvasName + `","organizationId":"org-uuid"}}}`))
}

func expectFetchLiveCanvasYAML(canvasID, yamlBody string) requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   repositoryCanvasFilePath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "canvas.yaml", r.URL.Query().Get("path"))
			require.Empty(t, r.URL.Query().Get("version_id"))
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(yamlBody))
		},
	}
}

func TestGetCommandPrintsCanvasOnSuccess(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasMetadataResponse,
		},
		expectFetchLiveCanvasYAML(testGetCanvasID, sampleCanvasYAMLBody),
	)
	ctx, stdout := cli.NewCommandContext(t, server.server, "text")
	ctx.Args = []string{testGetCanvasID}

	err := (&getCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ID: "+testGetCanvasID)
	require.Contains(t, stdout.String(), "Name: "+testGetCanvasName)
	require.Contains(t, stdout.String(), "Nodes: 0")
	require.Contains(t, stdout.String(), "Edges: 0")
	require.NotContains(t, stdout.String(), "App URL:")

	server.AssertCalls(t, []string{
		http.MethodGet + " " + testDescribeCanvas,
		http.MethodGet + " " + repositoryCanvasFilePath(testGetCanvasID),
	})
}

func TestGetCommandPrintsURLFromResponseOrgID(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasMetadataWithOrgResponse,
		},
		expectFetchLiveCanvasYAML(testGetCanvasID, sampleCanvasYAMLBody),
	)
	ctx, stdout := cli.NewCommandContextWithConfig(t, server.server, "text", &cli.FakeConfig{
		URL: "https://app.superplane.com",
	})
	ctx.Args = []string{testGetCanvasID}

	err := (&getCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ID: "+testGetCanvasID)
	require.Contains(t, stdout.String(), "Name: "+testGetCanvasName)
	require.Contains(t, stdout.String(), "App URL: https://app.superplane.com/org-uuid/apps/"+testGetCanvasID)
	require.Contains(t, stdout.String(), "Nodes: 0")
	require.Contains(t, stdout.String(), "Edges: 0")
}

func TestGetCommandSkipsURLWhenResponseMissingOrgID(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasMetadataResponse,
		},
		expectFetchLiveCanvasYAML(testGetCanvasID, sampleCanvasYAMLBody),
	)
	ctx, stdout := cli.NewCommandContextWithConfig(t, server.server, "text", &cli.FakeConfig{
		URL: "https://app.superplane.com",
	})
	ctx.Args = []string{testGetCanvasID}

	err := (&getCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ID: "+testGetCanvasID)
	require.Contains(t, stdout.String(), "Name: "+testGetCanvasName)
	require.NotContains(t, stdout.String(), "App URL:")
}

func TestGetUsesActiveAppWhenNoArg(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasMetadataResponse,
		},
		expectFetchLiveCanvasYAML(testGetCanvasID, sampleCanvasYAMLBody),
	)
	ctx, stdout := cli.NewCommandContextWithConfig(t, server.server, "text", &cli.FakeConfig{
		ActiveApp: testGetCanvasID,
	})
	ctx.Args = []string{}

	err := (&getCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ID: "+testGetCanvasID)
}

func TestGetErrorsWhenNoAppAndNoActive(t *testing.T) {
	server := newAPITestServer(t)
	ctx, _ := cli.NewCommandContext(t, server.server, "text")
	ctx.Args = []string{}

	err := (&getCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "app-name-or-id")
}

func TestGetLiveYAMLOutput(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasMetadataResponse,
		},
		expectFetchLiveCanvasYAML(testGetCanvasID, sampleCanvasYAMLBody),
	)

	ctx, stdout := cli.NewCommandContext(t, server.server, "yaml")
	ctx.Args = []string{testGetCanvasID}

	require.NoError(t, (&getCommand{}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, "apiVersion: v1")
	require.Contains(t, out, "kind: Canvas")
	require.Contains(t, out, "id: "+testGetCanvasID)
	require.Contains(t, out, "name: "+testGetCanvasName)
}
