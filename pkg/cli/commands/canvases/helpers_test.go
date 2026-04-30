package canvases

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

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
