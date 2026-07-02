package canvas

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/test/support/cli"
)

func TestFindCurrentUserDraftVersionIDReturnsFirstDraft(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   draftVersionsPath("canvas-123"),
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","owner":{"id":"user-1"}}},{"metadata":{"id":"draft-2","owner":{"id":"user-1"}}}]}`))
			},
		},
	)

	ctx, _ := cli.NewCommandContext(t, server.server, "text")

	versionID, err := common.FindCurrentUserDraftVersionID(ctx, "canvas-123")
	require.NoError(t, err)
	require.Equal(t, "draft-1", versionID)

	server.AssertCalls(t, []string{
		http.MethodGet + " " + draftVersionsPath("canvas-123"),
	})
}

func TestEnsureCurrentUserDraftVersionIDCreatesDraftWhenMissing(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   draftVersionsPath("canvas-123"),
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[]}`))
			},
		},
		requestExpectation{
			method: http.MethodPost,
			path:   draftVersionsPath("canvas-123"),
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1"}}}`))
			},
		},
	)

	ctx, _ := cli.NewCommandContext(t, server.server, "text")

	versionID, err := common.EnsureCurrentUserDraftVersionID(ctx, "canvas-123")
	require.NoError(t, err)
	require.Equal(t, "draft-1", versionID)

	server.AssertCalls(t, []string{
		http.MethodGet + " " + draftVersionsPath("canvas-123"),
		http.MethodPost + " " + draftVersionsPath("canvas-123"),
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

	ctx, _ := cli.NewCommandContext(t, server.server, "text")

	_, err := common.DescribeAppVersionByID(ctx, "canvas-123", "version-123")
	require.Error(t, err)
	require.Contains(t, err.Error(), `app version "version-123" not found`)
}
