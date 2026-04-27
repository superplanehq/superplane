package canvases

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type requestExpectation struct {
	method string
	path   string
	handle func(t *testing.T, w http.ResponseWriter, r *http.Request)
}

type apiTestServer struct {
	t            *testing.T
	expectations []requestExpectation
	calls        []string
	server       *httptest.Server
}

func newAPITestServer(t *testing.T, expectations ...requestExpectation) *apiTestServer {
	t.Helper()

	s := &apiTestServer{t: t, expectations: expectations}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls = append(s.calls, r.Method+" "+r.URL.Path)

		if len(s.expectations) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		next := s.expectations[0]
		require.Equal(t, next.method, r.Method)
		require.Equal(t, next.path, r.URL.Path)

		s.expectations = s.expectations[1:]
		if next.handle != nil {
			next.handle(t, w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *apiTestServer) AssertCalls(t *testing.T, calls []string) {
	t.Helper()
	require.Equal(t, calls, s.calls)
	require.Len(t, s.expectations, 0, "unused request expectations")
}

func TestUpdateWithoutFileReturnsError(t *testing.T) {
	server := newAPITestServer(t)
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	err := (&updateCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "file")
}

type stubActiveCanvasConfig struct {
	active string
}

func (s stubActiveCanvasConfig) GetActiveCanvas() string { return s.active }

func (s stubActiveCanvasConfig) SetActiveCanvas(string) error { return nil }

func writeTestCanvasFileWithoutMetadataID(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")
	content := []byte(
		"apiVersion: v1\nkind: Canvas\nmetadata:\n  name: " + name + "\nspec:\n  nodes: []\n  edges: []\n",
	)
	require.NoError(t, os.WriteFile(filePath, content, 0o644))
	return filePath
}

func writeTestCanvasFileWithMetadataID(t *testing.T, name, canvasID string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")
	content := []byte(
		"apiVersion: v1\nkind: Canvas\nmetadata:\n  id: " + canvasID + "\n  name: " + name + "\nspec:\n  nodes: []\n  edges: []\n",
	)
	require.NoError(t, os.WriteFile(filePath, content, 0o644))
	return filePath
}

func TestResolveCanvasForFileUpdateWithoutFileIDUsesPositional(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvases":[{"metadata":{"id":"` + canvasID + `","name":"parse-check"}}]}`))
		},
	})

	filePath := writeTestCanvasFileWithoutMetadataID(t, "parse-check")
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{"parse-check"}

	gotID, canvas, err := resolveCanvasForFileUpdate(ctx, filePath)
	require.NoError(t, err)
	require.Equal(t, canvasID, gotID)
	md := canvas.GetMetadata()
	require.Equal(t, canvasID, (&md).GetId())

	server.AssertCalls(t, []string{http.MethodGet + " /api/v1/canvases"})
}

func TestResolveCanvasForFileUpdateWithoutFileIDUsesActiveCanvas(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvases":[{"metadata":{"id":"` + canvasID + `","name":"parse-check"}}]}`))
		},
	})

	filePath := writeTestCanvasFileWithoutMetadataID(t, "parse-check")
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{}
	ctx.Config = stubActiveCanvasConfig{active: "parse-check"}

	gotID, canvas, err := resolveCanvasForFileUpdate(ctx, filePath)
	require.NoError(t, err)
	require.Equal(t, canvasID, gotID)
	md := canvas.GetMetadata()
	require.Equal(t, canvasID, (&md).GetId())

	server.AssertCalls(t, []string{http.MethodGet + " /api/v1/canvases"})
}

func TestResolveCanvasForFileUpdateWithoutFileIDRequiresTarget(t *testing.T) {
	server := newAPITestServer(t)
	filePath := writeTestCanvasFileWithoutMetadataID(t, "parse-check")
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{}

	_, _, err := resolveCanvasForFileUpdate(ctx, filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "metadata.id is empty")

	require.Empty(t, server.calls)
	require.Empty(t, server.expectations)
}

func TestResolveCanvasForFileUpdateRejectsIDMismatchWithUUIDArg(t *testing.T) {
	server := newAPITestServer(t)
	fileID := "11111111-1111-1111-1111-111111111111"
	otherID := "22222222-2222-2222-2222-222222222222"
	filePath := writeTestCanvasFileWithMetadataID(t, "a", fileID)
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{otherID}

	_, _, err := resolveCanvasForFileUpdate(ctx, filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match")

	require.Empty(t, server.calls)
	require.Empty(t, server.expectations)
}

func TestBuildDefaultAutoLayoutUsesFullCanvas(t *testing.T) {
	autoLayout := buildDefaultAutoLayout()

	if autoLayout.GetAlgorithm() != openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL {
		t.Fatalf("expected horizontal auto-layout, got %s", autoLayout.GetAlgorithm())
	}
	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS {
		t.Fatalf("expected full-canvas scope, got %s", autoLayout.GetScope())
	}
	if autoLayout.HasNodeIds() {
		t.Fatalf("expected no node ids for default full-canvas strategy, got %v", autoLayout.GetNodeIds())
	}
}

func TestParseAutoLayoutDefaultsAlgorithmToHorizontal(t *testing.T) {
	autoLayout, err := parseAutoLayout("", "connected-component", []string{"node-1", " node-2 ", "node-1"})
	if err != nil {
		t.Fatalf("parseAutoLayout returned error: %v", err)
	}
	if autoLayout == nil {
		t.Fatalf("expected autoLayout to be set")
	}
	if autoLayout.GetAlgorithm() != openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL {
		t.Fatalf("expected horizontal auto-layout, got %s", autoLayout.GetAlgorithm())
	}
	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_CONNECTED_COMPONENT {
		t.Fatalf("expected connected-component scope, got %s", autoLayout.GetScope())
	}
	if !reflect.DeepEqual(autoLayout.GetNodeIds(), []string{"node-1", "node-2"}) {
		t.Fatalf("expected node ids [node-1 node-2], got %v", autoLayout.GetNodeIds())
	}
}

func TestParseAutoLayoutDisable(t *testing.T) {
	autoLayout, err := parseAutoLayout("disable", "", nil)
	if err != nil {
		t.Fatalf("parseAutoLayout returned error: %v", err)
	}
	if autoLayout != nil {
		t.Fatalf("expected nil autoLayout when disabled, got %#v", autoLayout)
	}
}

func TestParseAutoLayoutDisableRejectsScopeOrNodes(t *testing.T) {
	if _, err := parseAutoLayout("disable", "connected-component", nil); err == nil {
		t.Fatalf("expected error when scope is set together with disable")
	}
	if _, err := parseAutoLayout("disable", "", []string{"node-1"}); err == nil {
		t.Fatalf("expected error when node ids are set together with disable")
	}
}

func TestUpdateFromFileAppliesChangeManagementEnabledAfterSpecUpdateWhenNotDraft(t *testing.T) {
	t.Helper()

	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	server := newAPITestServer(
		t,
		// 1. Describe canvas
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check"},"spec":{"changeManagement":{"enabled":false}}}}`))
			},
		},
		// 2. List versions (no existing draft)
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[]}`))
			},
		},
		// 3. Create draft version
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"}}}`))
			},
		},
		// 4. Update draft version
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"},"spec":{"nodes":[],"edges":[]}}}`))
			},
		},
		// 5. Auto-publish (not in draft mode)
		requestExpectation{
			method: http.MethodPatch,
			path:   "/api/v1/canvases/" + canvasID + "/versions/ver-1/publish",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"}}}`))
			},
		},
		// 6. Enable change management after spec update
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID,
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				rawBody, _ := io.ReadAll(r.Body)
				var payload map[string]any
				_ = json.Unmarshal(rawBody, &payload)
				cm := payload["changeManagement"].(map[string]any)
				require.Equal(t, true, cm["enabled"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check"},"spec":{"changeManagement":{"enabled":true}}}}`))
			},
		},
	)

	filePath := writeTestCanvasFileWithChangeManagementEnabled(t, canvasID, true)
	file := filePath
	draft := false
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.NoError(t, err)

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPost + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPatch + " /api/v1/canvases/" + canvasID + "/versions/ver-1/publish",
		http.MethodPut + " /api/v1/canvases/" + canvasID,
	})
}

func TestUpdateFromFileDisablesChangeManagementBeforeSpecUpdate(t *testing.T) {
	t.Helper()

	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	server := newAPITestServer(
		t,
		// 1. Describe canvas
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check"},"spec":{"changeManagement":{"enabled":true}}}}`))
			},
		},
		// 2. Resolve org to check if disable is allowed
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/me",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"user":{"id":"user-1","organizationId":"org-1"}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/organizations/org-1",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"organization":{"metadata":{"id":"org-1"},"spec":{"changeManagementEnabled":false}}}`))
			},
		},
		// 3. Disable change management
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID,
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				rawBody, _ := io.ReadAll(r.Body)
				var payload map[string]any
				_ = json.Unmarshal(rawBody, &payload)
				cm := payload["changeManagement"].(map[string]any)
				require.Equal(t, false, cm["enabled"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check"},"spec":{"changeManagement":{"enabled":false}}}}`))
			},
		},
		// 4. List versions (no existing draft)
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[]}`))
			},
		},
		// 5. Create draft version
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"}}}`))
			},
		},
		// 6. Update draft version
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"},"spec":{"nodes":[],"edges":[]}}}`))
			},
		},
		// 7. Auto-publish
		requestExpectation{
			method: http.MethodPatch,
			path:   "/api/v1/canvases/" + canvasID + "/versions/ver-1/publish",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"}}}`))
			},
		},
	)

	filePath := writeTestCanvasFileWithChangeManagementEnabled(t, canvasID, false)
	file := filePath
	draft := false
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.NoError(t, err)

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/me",
		http.MethodGet + " /api/v1/organizations/org-1",
		http.MethodPut + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPost + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPatch + " /api/v1/canvases/" + canvasID + "/versions/ver-1/publish",
	})
}

func TestUpdateFromFileEnablesChangeManagementBeforeDraftUpdate(t *testing.T) {
	t.Helper()

	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check"},"spec":{"changeManagement":{"enabled":false}}}}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID,
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				rawBody, _ := io.ReadAll(r.Body)
				var payload map[string]any
				_ = json.Unmarshal(rawBody, &payload)
				cm := payload["changeManagement"].(map[string]any)
				require.Equal(t, true, cm["enabled"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check"},"spec":{"changeManagement":{"enabled":true}}}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `","isPublished":false}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				rawBody, _ := io.ReadAll(r.Body)
				var payload map[string]any
				_ = json.Unmarshal(rawBody, &payload)
				require.Equal(t, "draft-1", payload["versionId"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `"},"spec":{"nodes":[],"edges":[]}}}`))
			},
		},
	)

	filePath := writeTestCanvasFileWithChangeManagementEnabled(t, canvasID, true)
	file := filePath
	draft := true
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.NoError(t, err)

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodPut + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
	})
}

func TestUpdateFromFileDisableChangeManagementFailsWhenOrganizationEnforcesIt(t *testing.T) {
	t.Helper()

	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check"},"spec":{"changeManagement":{"enabled":true}}}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/me",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"user":{"id":"user-1","organizationId":"org-1"}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/organizations/org-1",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"organization":{"metadata":{"id":"org-1"},"spec":{"changeManagementEnabled":true}}}`))
			},
		},
	)

	filePath := writeTestCanvasFileWithChangeManagementEnabled(t, canvasID, false)
	file := filePath
	draft := false
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot disable change management")

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/me",
		http.MethodGet + " /api/v1/organizations/org-1",
	})
}

func writeTestCanvasFileWithChangeManagementEnabled(t *testing.T, canvasID string, enabled bool) string {
	t.Helper()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")

	enabledValue := "false"
	if enabled {
		enabledValue = "true"
	}

	content := []byte(
		"apiVersion: v1\n" +
			"kind: Canvas\n" +
			"metadata:\n" +
			"  id: " + canvasID + "\n" +
			"  name: parse-check\n" +
			"spec:\n" +
			"  nodes: []\n" +
			"  edges: []\n" +
			"  changeManagement:\n" +
			"    enabled: " + enabledValue + "\n",
	)
	require.NoError(t, os.WriteFile(filePath, content, 0o644))
	return filePath
}
