package canvases

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// fakeConfig is a test stub implementation of core.ConfigContext. The
// canvas-active accessors are no-ops because the canvas commands under test
// here do not use them.
type fakeConfig struct {
	url string
}

func (f *fakeConfig) GetActiveCanvas() string               { return "" }
func (f *fakeConfig) SetActiveCanvas(canvasID string) error { return nil }
func (f *fakeConfig) GetURL() string                        { return f.url }

// newCommandContextWithConfigForTest builds a command context whose Config is
// set to the provided stub. Useful for exercising URL output behavior.
func newCommandContextWithConfigForTest(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
	config core.ConfigContext,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	ctx, stdout := newCreateCommandContextForTest(t, server, outputFormat)
	ctx.Config = config
	return ctx, stdout
}

func TestFindCurrentUserDraftVersionIDSkipsPublishedVersions(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"pub-1","state":"STATE_PUBLISHED"}},{"metadata":{"id":"draft-1","state":"STATE_DRAFT"}}]}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	versionID, err := findCurrentUserDraftVersionID(ctx, "canvas-123")
	require.NoError(t, err)
	require.Equal(t, "draft-1", versionID)
}

func TestEnsureCurrentUserDraftVersionIDCreatesDraftWhenMissing(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[]}`))
			},
		},
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/canvas-123/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1"}}}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	versionID, err := ensureCurrentUserDraftVersionID(ctx, "canvas-123")
	require.NoError(t, err)
	require.Equal(t, "draft-1", versionID)

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/canvas-123/versions",
		http.MethodPost + " /api/v1/canvases/canvas-123/versions",
	})
}

func TestDescribeCanvasVersionByIDReturnsErrorWhenVersionMissing(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/versions/version-123",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	_, err := describeCanvasVersionByID(ctx, "canvas-123", "version-123")
	require.Error(t, err)
	require.Contains(t, err.Error(), `canvas version "version-123" not found`)
}

func TestCanvasFromVersionCopiesSpec(t *testing.T) {
	version := openapi_client.CanvasesCanvasVersion{}
	spec := openapi_client.CanvasesCanvasSpec{}
	spec.SetNodes([]openapi_client.SuperplaneComponentsNode{{Name: openapi_client.PtrString("first")}})
	spec.SetEdges([]openapi_client.SuperplaneComponentsEdge{{SourceId: openapi_client.PtrString("a")}})
	version.SetSpec(spec)

	canvas := canvasFromVersion(version)

	require.NotNil(t, canvas.Spec)
	require.Len(t, canvas.Spec.GetNodes(), 1)
	require.Len(t, canvas.Spec.GetEdges(), 1)
	require.Equal(t, "first", canvas.Spec.GetNodes()[0].GetName())
}
