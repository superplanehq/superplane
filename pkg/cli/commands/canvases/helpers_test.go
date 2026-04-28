package canvases

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func loadCanvasFromExisting(ctx core.CommandContext) (string, openapi_client.CanvasesCanvas, error) {
	if len(ctx.Args) > 1 {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("update accepts at most one positional argument")
	}

	var canvasID string
	var err error

	if len(ctx.Args) == 1 {
		canvasID, err = findCanvasID(ctx, ctx.API, ctx.Args[0])
		if err != nil {
			return "", openapi_client.CanvasesCanvas{}, err
		}
	} else {
		if ctx.Config == nil {
			return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("no canvas specified: pass a canvas name or id, or set an active canvas with `superplane canvases active`")
		}
		active := strings.TrimSpace(ctx.Config.GetActiveCanvas())
		if active == "" {
			return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("no canvas specified: pass a canvas name or id, or set an active canvas with `superplane canvases active`")
		}
		canvasID, err = findCanvasID(ctx, ctx.API, active)
		if err != nil {
			return "", openapi_client.CanvasesCanvas{}, err
		}
	}

	canvas, err := describeCanvasByID(ctx, canvasID)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	return canvasID, canvas, nil
}

func describeCanvasByID(ctx core.CommandContext, canvasID string) (openapi_client.CanvasesCanvas, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return openapi_client.CanvasesCanvas{}, err
	}
	if response.Canvas == nil {
		return openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas %q not found", canvasID)
	}
	return *response.Canvas, nil
}

type testConfigContext struct {
	activeCanvas string
}

func (c *testConfigContext) GetActiveCanvas() string {
	return c.activeCanvas
}

func (c *testConfigContext) SetActiveCanvas(canvasID string) error {
	c.activeCanvas = canvasID
	return nil
}

func TestLoadCanvasFromExistingUsesConfiguredActiveCanvas(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvases":[{"metadata":{"id":"canvas-123","name":"active-canvas"}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"canvas-123","name":"active-canvas"},"spec":{"nodes":[],"edges":[]}}}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Config = &testConfigContext{activeCanvas: "active-canvas"}

	canvasID, canvas, err := loadCanvasFromExisting(ctx)
	require.NoError(t, err)
	require.Equal(t, "canvas-123", canvasID)
	require.NotNil(t, canvas.Metadata)
	require.Equal(t, "active-canvas", canvas.Metadata.GetName())

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases",
		http.MethodGet + " /api/v1/canvases/canvas-123",
	})
}

func TestLoadCanvasFromExistingRejectsMultipleArgs(t *testing.T) {
	ctx, _ := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{"one", "two"}

	_, _, err := loadCanvasFromExisting(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "update accepts at most one positional argument")
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

func TestDescribeCanvasByIDReturnsErrorWhenCanvasMissing(t *testing.T) {
	config := openapi_client.NewConfiguration()
	client := openapi_client.NewAPIClient(config)
	ctx := core.CommandContext{
		Context: context.Background(),
		API:     client,
	}

	_, err := describeCanvasByID(ctx, "canvas-123")
	require.Error(t, err)
}
