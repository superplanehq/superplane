package canvases

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
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

func TestResolveCanvasForFileUpdateWithoutIDReturnsError(t *testing.T) {
	filePath := writeTestCanvasFileWithoutMetadataID(t, "parse-check")
	_, _, err := resolveCanvasForFileUpdate(filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "metadata.id is required")
}

func TestResolveCanvasForFileUpdateWithIDSucceeds(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	filePath := writeTestCanvasFileWithMetadataID(t, "my-canvas", canvasID)
	gotID, canvas, err := resolveCanvasForFileUpdate(filePath)
	require.NoError(t, err)
	require.Equal(t, canvasID, gotID)
	md := canvas.GetMetadata()
	require.Equal(t, canvasID, (&md).GetId())
}

func TestResolveCanvasForFileUpdateWhitespaceIDReturnsError(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")
	content := []byte(
		"apiVersion: v1\nkind: Canvas\nmetadata:\n  id: \"   \"\n  name: x\nspec:\n  nodes: []\n  edges: []\n",
	)
	require.NoError(t, os.WriteFile(filePath, content, 0o644))

	_, _, err := resolveCanvasForFileUpdate(filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "metadata.id is required")
}

func TestAutoLayoutFlagsWereSet(t *testing.T) {
	t.Run("nil command", func(t *testing.T) {
		require.False(t, autoLayoutFlagsWereSet(core.CommandContext{}))
	})
	t.Run("auto-layout flag changed", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String("auto-layout", "", "")
		cmd.Flags().String("auto-layout-scope", "", "")
		cmd.Flags().StringArray("auto-layout-node", nil, "")
		require.NoError(t, cmd.ParseFlags([]string{"--auto-layout=horizontal"}))
		require.True(t, autoLayoutFlagsWereSet(core.CommandContext{Cmd: &cmd}))
	})
	t.Run("auto-layout-scope flag changed", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String("auto-layout", "", "")
		cmd.Flags().String("auto-layout-scope", "", "")
		cmd.Flags().StringArray("auto-layout-node", nil, "")
		require.NoError(t, cmd.ParseFlags([]string{"--auto-layout-scope=full-canvas"}))
		require.True(t, autoLayoutFlagsWereSet(core.CommandContext{Cmd: &cmd}))
	})
	t.Run("auto-layout-node flag changed", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String("auto-layout", "", "")
		cmd.Flags().String("auto-layout-scope", "", "")
		cmd.Flags().StringArray("auto-layout-node", nil, "")
		require.NoError(t, cmd.ParseFlags([]string{"--auto-layout-node=a"}))
		require.True(t, autoLayoutFlagsWereSet(core.CommandContext{Cmd: &cmd}))
	})
	t.Run("no auto-layout flags parsed", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String("auto-layout", "", "")
		require.NoError(t, cmd.ParseFlags(nil))
		require.False(t, autoLayoutFlagsWereSet(core.CommandContext{Cmd: &cmd}))
	})
}

func TestParseAutoLayoutRejectsInvalidAlgorithm(t *testing.T) {
	_, err := parseAutoLayout("vertical", "", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported auto layout")
}

func TestParseAutoLayoutRejectsInvalidScope(t *testing.T) {
	_, err := parseAutoLayout("horizontal", "whole-world", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported auto layout scope")
}

func TestParseAutoLayoutScopeFullAlias(t *testing.T) {
	autoLayout, err := parseAutoLayout("horizontal", "full", nil)
	require.NoError(t, err)
	require.NotNil(t, autoLayout)
	require.Equal(t, openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS, autoLayout.GetScope())
}

func TestParseAutoLayoutDisableAcceptsOffAndNone(t *testing.T) {
	for _, v := range []string{"off", "none", "NONE"} {
		t.Run(v, func(t *testing.T) {
			al, err := parseAutoLayout(v, "", nil)
			require.NoError(t, err)
			require.Nil(t, al)
		})
	}
}

func TestUpdateFromFileJSONOutputWhenDraft(t *testing.T) {
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
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `","state":"STATE_DRAFT"}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `"},"spec":{"nodes":[],"edges":[]}}}`))
			},
		},
	)

	filePath := writeTestCanvasFileWithMetadataID(t, "json-out", canvasID)
	file := filePath
	draft := true
	ctx, stdout := newCreateCommandContextForTest(t, server.server, "json")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"metadata"`)

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
	})
}

func TestUpdateFromFileWhenPublishFailsReturnsWrappedError(t *testing.T) {
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
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[]}`))
			},
		},
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"}}}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"ver-1","canvasId":"` + canvasID + `"},"spec":{"nodes":[],"edges":[]}}}`))
			},
		},
		requestExpectation{
			method: http.MethodPatch,
			path:   "/api/v1/canvases/" + canvasID + "/versions/ver-1/publish",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"publish failed"}`))
			},
		},
	)

	filePath := writeTestCanvasFileWithMetadataID(t, "pub-fail", canvasID)
	file := filePath
	draft := false
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "draft was updated but publish failed"), err.Error())

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPost + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPatch + " /api/v1/canvases/" + canvasID + "/versions/ver-1/publish",
	})
}

func TestUpdateFromFileWhenCanvasesUpdateFailsReturnsError(t *testing.T) {
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
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `","state":"STATE_DRAFT"}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"message":"version conflict"}`))
			},
		},
	)

	filePath := writeTestCanvasFileWithMetadataID(t, "put-fail", canvasID)
	file := filePath
	draft := true
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.Error(t, err)

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
	})
}

func TestUpdateFromFileTextOutputCountsIntegrations(t *testing.T) {
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
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `","state":"STATE_DRAFT"}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(
					`{"version":{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `"},` +
						`"spec":{"nodes":[{"id":"n1","integration":{"id":"int-1"}}],` +
						`"edges":[]}}}`,
				))
			},
		},
	)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")
	yaml := "" +
		"apiVersion: v1\nkind: Canvas\nmetadata:\n  id: " + canvasID + "\n  name: integ\nspec:\n  nodes: []\n  edges: []\n"
	require.NoError(t, os.WriteFile(filePath, []byte(yaml), 0o644))

	file := filePath
	draft := true
	ctx, stdout := newCreateCommandContextForTest(t, server.server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Integrations: 1")

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
		http.MethodGet + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
	})
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
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				rawBody, _ := io.ReadAll(r.Body)
				var payload map[string]any
				_ = json.Unmarshal(rawBody, &payload)
				canvasPayload := payload["canvas"].(map[string]any)
				specPayload := canvasPayload["spec"].(map[string]any)
				cm := specPayload["changeManagement"].(map[string]any)
				require.Equal(t, true, cm["enabled"])

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
	})
}

func TestUpdateFromFileRequiresDraftWhenLiveChangeManagementIsEnabled(t *testing.T) {
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
	)

	filePath := writeTestCanvasFileWithChangeManagementEnabled(t, canvasID, false)
	file := filePath
	draft := false
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.Error(t, err)
	require.Contains(
		t,
		err.Error(),
		"change management is enabled for this canvas; use --draft to update your draft version, then publish with `superplane canvases change-requests create`",
	)

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
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
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","canvasId":"` + canvasID + `","state":"STATE_DRAFT"}}]}`))
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
				canvasPayload := payload["canvas"].(map[string]any)
				specPayload := canvasPayload["spec"].(map[string]any)
				cm := specPayload["changeManagement"].(map[string]any)
				require.Equal(t, true, cm["enabled"])
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
		http.MethodGet + " /api/v1/canvases/" + canvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + canvasID + "/versions",
	})
}

func TestUpdateFromFileDisableChangeManagementRequiresDraftWhenLiveChangeManagementIsEnabled(t *testing.T) {
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
	)

	filePath := writeTestCanvasFileWithChangeManagementEnabled(t, canvasID, false)
	file := filePath
	draft := false
	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")

	err := (&updateCommand{file: &file, draft: &draft}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "change management is enabled for this canvas")

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + canvasID,
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
