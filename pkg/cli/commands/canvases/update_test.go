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

func TestUpdateFromFileAppliesVersioningEnabledAfterSpecUpdateWhenNotDraft(t *testing.T) {
	t.Helper()

	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	calls := make([]string, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/"+canvasID:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check","versioningEnabled":false}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/canvases/"+canvasID+"/versions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"},"spec":{"nodes":[],"edges":[]}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/canvases/"+canvasID:
			rawBody, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(rawBody, &payload)
			require.Equal(t, true, payload["versioningEnabled"])
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check","versioningEnabled":true}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	filePath := writeTestCanvasFileWithVersioningEnabled(t, canvasID, true)
	file := filePath
	draft := false
	ctx, _ := newCreateCommandContextForTest(t, server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.NoError(t, err)

	require.Equal(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID,
	}, calls)
}

func TestUpdateFromFileDisablesVersioningBeforeSpecUpdate(t *testing.T) {
	t.Helper()

	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	calls := make([]string, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/"+canvasID:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check","versioningEnabled":true}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/canvases/"+canvasID:
			rawBody, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(rawBody, &payload)
			require.Equal(t, false, payload["versioningEnabled"])
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check","versioningEnabled":false}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/canvases/"+canvasID+"/versions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"},"spec":{"nodes":[],"edges":[]}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	filePath := writeTestCanvasFileWithVersioningEnabled(t, canvasID, false)
	file := filePath
	draft := false
	ctx, _ := newCreateCommandContextForTest(t, server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.NoError(t, err)

	require.Equal(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodPut + " /api/v1/canvases/" + canvasID,
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
	}, calls)
}

func TestUpdateFromFileEnablesVersioningBeforeDraftUpdate(t *testing.T) {
	t.Helper()

	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	calls := make([]string, 0, 6)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/"+canvasID:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check","versioningEnabled":false}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/canvases/"+canvasID:
			rawBody, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(rawBody, &payload)
			require.Equal(t, true, payload["versioningEnabled"])
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"parse-check","versioningEnabled":true}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/"+canvasID+"/versions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `","isPublished":false}}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/canvases/"+canvasID+"/versions":
			rawBody, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(rawBody, &payload)
			require.Equal(t, "draft-1", payload["versionId"])
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `"},"spec":{"nodes":[],"edges":[]}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	filePath := writeTestCanvasFileWithVersioningEnabled(t, canvasID, true)
	file := filePath
	draft := true
	ctx, _ := newCreateCommandContextForTest(t, server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.NoError(t, err)

	require.Equal(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodPut + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
	}, calls)
}

func writeTestCanvasFileWithVersioningEnabled(t *testing.T, canvasID string, enabled bool) string {
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
			"  versioningEnabled: " + enabledValue + "\n" +
			"spec:\n" +
			"  nodes: []\n" +
			"  edges: []\n",
	)
	require.NoError(t, os.WriteFile(filePath, content, 0o644))
	return filePath
}
